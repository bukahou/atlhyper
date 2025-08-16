export function buildPodSpec(form) {
  const s = form?.sched || {}

  const imagePullSecrets = (s.imagePullSecrets || [])
    .map((x) => (typeof x === 'string' ? x : x?.name))
    .filter(Boolean)
    .map((name) => ({ name: String(name).trim() }))

  const volumes = (form?.volumes || [])
    .map((v) => {
      const name = (v?.name || '').trim()
      if (!name) return null

      if (v.type === 'hostPath' && v.hostPath?.path) {
        const hp = {
          path: String(v.hostPath.path).trim(),
          ...(v.hostPath.type ? { type: String(v.hostPath.type).trim() } : {})
        }
        return { name, hostPath: hp }
      }

      if (v.type === 'emptyDir') {
        const ed = {}
        if (v.emptyDir?.medium) ed.medium = String(v.emptyDir.medium).trim()
        if (v.emptyDir?.sizeLimit) { ed.sizeLimit = String(v.emptyDir.sizeLimit).trim() }
        return { name, emptyDir: ed }
      }

      if (v.type === 'pvc' && v.pvc?.claimName) {
        return {
          name,
          persistentVolumeClaim: { claimName: String(v.pvc.claimName).trim() }
        }
      }

      return null
    })
    .filter(Boolean)

  const podSpec = {
    ...(imagePullSecrets.length ? { imagePullSecrets } : {}),
    ...(s.nodeSelector && Object.keys(s.nodeSelector).length
      ? { nodeSelector: s.nodeSelector }
      : {}),
    ...(s.affinity ? { affinity: s.affinity } : {}),
    ...(volumes.length ? { volumes } : {}),
    containers: [] // index.js 会装进去
  }

  return podSpec
}
