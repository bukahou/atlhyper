// 新增：把 UI 中的 probes 映射为 K8s 的 Probe 对象
function buildProbe(p, isReadiness = false) {
  if (!p || !p.type) return undefined

  const base = {}
  if (Number.isFinite(p.initialDelaySeconds)) { base.initialDelaySeconds = p.initialDelaySeconds }
  if (Number.isFinite(p.periodSeconds)) base.periodSeconds = p.periodSeconds
  if (Number.isFinite(p.timeoutSeconds)) base.timeoutSeconds = p.timeoutSeconds
  if (Number.isFinite(p.failureThreshold)) { base.failureThreshold = p.failureThreshold }
  if (isReadiness && Number.isFinite(p.successThreshold)) {
    base.successThreshold = p.successThreshold
  }

  switch (p.type) {
    case 'http': {
      const port = Number(p.http?.port)
      const path = String(p.http?.path || '').trim()
      if (!Number.isFinite(port) || !path) return undefined
      return {
        httpGet: {
          path,
          port,
          scheme: p.http?.scheme || 'HTTP'
        },
        ...base
      }
    }
    case 'tcp': {
      const port = Number(p.tcp?.port)
      if (!Number.isFinite(port)) return undefined
      return { tcpSocket: { port }, ...base }
    }
    case 'grpc': {
      const port = Number(p.grpc?.port)
      if (!Number.isFinite(port)) return undefined
      const pr = { port }
      const svc = String(p.grpc?.service || '').trim()
      if (svc) pr.service = svc
      return { grpc: pr, ...base }
    }
    case 'exec': {
      const cmd = Array.isArray(p.exec?.command) ? p.exec.command : []
      if (!cmd.length) return undefined
      return { exec: { command: cmd }, ...base }
    }
    default:
      return undefined
  }
}

export function buildContainers(form) {
  const c = form?.container || {}
  const image = (c.image || '').trim()
  if (!image) return [] // 至少需要镜像

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

  // 🔽 新增：从表单构建探针
  const readinessProbe = buildProbe(c.probes?.readiness, true)
  const livenessProbe = buildProbe(c.probes?.liveness, false)

  const container = {
    name: (c.name || 'container').trim() || 'container',
    image,
    ...(c.pullPolicy ? { imagePullPolicy: c.pullPolicy } : {}),
    ...(ports.length ? { ports } : {}),
    ...(envFrom.length ? { envFrom } : {}),
    ...(env.length ? { env } : {}),
    ...(volumeMounts.length ? { volumeMounts } : {}),
    ...(Object.keys(resources).length ? { resources } : {}),
    ...(readinessProbe ? { readinessProbe } : {}),
    ...(livenessProbe ? { livenessProbe } : {})
  }

  return [container]
}
