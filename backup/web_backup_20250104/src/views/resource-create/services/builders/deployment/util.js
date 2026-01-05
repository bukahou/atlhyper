export const kvArrayToObject = (arr = []) =>
  (arr || []).reduce((acc, { key, value }) => {
    const k = String(key || '').trim()
    if (!k) return acc
    acc[k] = String(value || '').trim()
    return acc
  }, {})

// 若未提供 labels，则 selector 回落为 { app: name }
export const selectorFrom = (labels = {}, name = '') => {
  return Object.keys(labels || {}).length ? labels : name ? { app: name } : {}
}

// 小工具：是否为非空对象
export const isNonEmptyObject = (o) =>
  !!(o && typeof o === 'object' && !Array.isArray(o) && Object.keys(o).length)
