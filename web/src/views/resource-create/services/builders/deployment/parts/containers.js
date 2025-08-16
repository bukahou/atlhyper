export function buildContainers(form) {
  const c = form?.container || {}
  const image = (c.image || '').trim()
  if (!image) return [] // 至少需要镜像，index.js 已做兜底，这里走安全线

  const ports = (c.ports || [])
    .filter((p) => Number(p?.containerPort))
    .map((p) => ({
      containerPort: Number(p.containerPort),
      ...(p.name ? { name: String(p.name).trim() } : {})
    }))

  const env = (c.env || [])
    .filter((e) => (e.name || '').trim())
    .map((e) => ({
      name: e.name.trim(),
      value: String(e.value ?? '')
    }))

  const envFrom = (c.envFrom || [])
    .map((x) => ({
      type: x.type === 'secretRef' ? 'secretRef' : 'configMapRef',
      name: (x.name || '').trim()
    }))
    .filter((x) => x.name)
    .map((x) => ({ [x.type]: { name: x.name }}))

  const volumeMounts = (c.volumeMounts || [])
    .filter((m) => (m.name || '').trim() && (m.mountPath || '').trim())
    .map((m) => ({
      name: m.name.trim(),
      mountPath: m.mountPath.trim(),
      ...(m.subPath ? { subPath: String(m.subPath).trim() } : {}),
      ...(m.readOnly === true ? { readOnly: true } : {})
    }))

  const resources = {}
  if (
    c.resources?.requests &&
    (c.resources.requests.cpu || c.resources.requests.memory)
  ) {
    resources.requests = {}
    if (c.resources.requests.cpu) { resources.requests.cpu = c.resources.requests.cpu }
    if (c.resources.requests.memory) { resources.requests.memory = c.resources.requests.memory }
  }
  if (
    c.resources?.limits &&
    (c.resources.limits.cpu || c.resources.limits.memory)
  ) {
    resources.limits = {}
    if (c.resources.limits.cpu) resources.limits.cpu = c.resources.limits.cpu
    if (c.resources.limits.memory) { resources.limits.memory = c.resources.limits.memory }
  }

  const container = {
    name: (c.name || 'container').trim() || 'container',
    image,
    ...(c.pullPolicy ? { imagePullPolicy: c.pullPolicy } : {}),
    ...(ports.length ? { ports } : {}),
    ...(envFrom.length ? { envFrom } : {}),
    ...(env.length ? { env } : {}),
    ...(volumeMounts.length ? { volumeMounts } : {}),
    ...(Object.keys(resources).length ? { resources } : {})
  }

  return [container]
}
