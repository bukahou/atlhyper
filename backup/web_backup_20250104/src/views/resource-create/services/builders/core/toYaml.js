// 对象 -> YAML 文本（统一格式）
// 依赖：npm i yaml
import YAML from 'yaml'

export function toYaml(doc) {
  if (!doc) return ''
  return YAML.stringify(doc, { indent: 2, lineWidth: 120 })
}
