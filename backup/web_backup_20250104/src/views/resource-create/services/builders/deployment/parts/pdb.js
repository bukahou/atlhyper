export function buildPdb(form, ctx) {
  const pdb = form?.pdb || {}
  if (!pdb.enabled) return null

  const ns = (form?.basic?.namespace || '').trim()
  const mode = (pdb.mode || '').trim() // "minAvailable" | "maxUnavailable"
  const value = (pdb.value || '').trim() // "1" | "10%" ...

  if (!mode || !value) return null

  // 尝试把纯数字转为 number，否则保留字符串（支持 "10%"）
  const parsed = /^\d+$/.test(value) ? Number(value) : value

  return {
    apiVersion: 'policy/v1',
    kind: 'PodDisruptionBudget',
    metadata: {
      name: ctx.name,
      ...(ns ? { namespace: ns } : {})
    },
    spec: {
      selector: { matchLabels: ctx.selector || {}},
      [mode]: parsed
    }
  }
}
