// 递归删除空值/空结构，但保留 0/false
export function prune(v) {
  if (Array.isArray(v)) {
    const arr = v.map(prune).filter((x) => !isEmpty(x))
    return arr.length ? arr : undefined
  }
  if (v && typeof v === 'object') {
    const out = {}
    Object.keys(v).forEach((k) => {
      const pv = prune(v[k])
      if (!isEmpty(pv)) out[k] = pv
    })
    return Object.keys(out).length ? out : undefined
  }
  return v // 基本类型按原样返回（包含 0/false）
}

function isEmpty(x) {
  if (x == null) return true // null / undefined
  if (x === '') return true // 空字符串
  if (Array.isArray(x)) return x.length === 0
  if (typeof x === 'object') return Object.keys(x).length === 0
  return false // 0 / false / 其它都算“非空”
}
