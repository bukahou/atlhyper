export function buildIngress(form, ctx) {
  const ing = form?.ing || {}
  if (!ing.enabled) return null

  const name = (ing.name || `${ctx.name}-ing`).trim()
  const ns = (ing.namespace || form?.basic?.namespace || '').trim()
  const cls = (ing.ingressClassName || '').trim()

  const rules = (ing.rules || []).map((r) => {
    const host = (r.host || '').trim()
    const pathType = r.pathType === 'Exact' ? 'Exact' : 'Prefix'
    const paths = (r.paths || []).map((p) => {
      const path = (p.path || '/').trim() || '/'
      const sName = (p.serviceName || form?.svc?.name || ctx.name).trim()
      const sPort = Number(p.servicePort) || 80
      return {
        path,
        pathType,
        backend: {
          service: {
            name: sName,
            port: { number: sPort }
          }
        }
      }
    })

    const rule = { http: { paths }}
    if (host) rule.host = host
    return rule
  })

  const tls =
    ing.tlsEnabled && ing.tlsSecretName
      ? [
        {
          secretName: ing.tlsSecretName,
          ...(ing.tlsHost ? { hosts: [ing.tlsHost] } : {})
        }
      ]
      : undefined

  return {
    apiVersion: 'networking.k8s.io/v1',
    kind: 'Ingress',
    metadata: {
      name,
      ...(ns ? { namespace: ns } : {}),
      ...(ctx.annotations && Object.keys(ctx.annotations).length
        ? { annotations: ctx.annotations }
        : {})
    },
    spec: {
      ...(cls ? { ingressClassName: cls } : {}),
      ...(tls ? { tls } : {}),
      rules: rules.length
        ? rules
        : [
          {
            http: {
              paths: [
                {
                  path: '/',
                  pathType: 'Prefix',
                  backend: {
                    service: {
                      name: form?.svc?.name || ctx.name,
                      port: { number: 80 }
                    }
                  }
                }
              ]
            }
          }
        ]
    }
  }
}
