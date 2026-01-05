export function buildHpa(form, ctx) {
  const h = form?.hpa || {}
  if (!h.enabled) return null

  const ns = (form?.basic?.namespace || '').trim()
  const metrics = (h.metrics || [])
    .filter((m) => Number(m?.target))
    .map((m) => ({
      type: 'Resource',
      resource: {
        name: m.type === 'memoryUtilization' ? 'memory' : 'cpu',
        target: {
          type: 'Utilization',
          averageUtilization: Number(m.target)
        }
      }
    }))

  return {
    apiVersion: 'autoscaling/v2',
    kind: 'HorizontalPodAutoscaler',
    metadata: {
      name: ctx.name,
      ...(ns ? { namespace: ns } : {})
    },
    spec: {
      scaleTargetRef: {
        apiVersion: 'apps/v1',
        kind: 'Deployment',
        name: ctx.name
      },
      ...(Number.isFinite(+h.minReplicas)
        ? { minReplicas: +h.minReplicas }
        : {}),
      ...(Number.isFinite(+h.maxReplicas)
        ? { maxReplicas: +h.maxReplicas }
        : {}),
      ...(metrics.length ? { metrics } : {})
    }
  }
}
