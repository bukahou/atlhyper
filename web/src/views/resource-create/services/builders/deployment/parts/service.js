export function buildService(form, ctx) {
  const svc = form?.svc || {}
  if (!svc.enabled) return null

  const name = (svc.name || ctx.name).trim()
  const ns = (svc.namespace || form?.basic?.namespace || '').trim()
  const type = (svc.type || 'ClusterIP').trim()

  const ports = (svc.ports || [])
    .filter((p) => Number(p?.port))
    .map((p) => {
      const item = {
        name: p.name || `p-${Number(p.port)}`,
        port: Number(p.port),
        targetPort: Number(p.targetPort) || Number(p.port),
        protocol: p.protocol === 'UDP' ? 'UDP' : 'TCP'
      }
      if (type === 'NodePort' && Number(p.nodePort)) {
        item.nodePort = Number(p.nodePort)
      }
      return item
    })

  return {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name,
      ...(ns ? { namespace: ns } : {}),
      ...(ctx.labels && Object.keys(ctx.labels).length
        ? { labels: ctx.labels }
        : {})
    },
    spec: {
      type,
      selector: ctx.selector || {},
      ports: ports.length
        ? ports
        : [{ port: 80, targetPort: 80, protocol: 'TCP' }]
    }
  }
}
