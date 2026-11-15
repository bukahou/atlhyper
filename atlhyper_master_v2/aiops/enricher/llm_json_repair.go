// atlhyper_master_v2/aiops/enricher/llm_json_repair.go
// 模型输出的 JSON 常见格式错误的修复：字段名上的 Markdown 加粗
package enricher

import (
	"encoding/json"
	"regexp"
)

// boldKey 字段名被 Markdown 加粗包住的几种写法（只匹配「紧跟冒号」的字段名位置，不碰值里的 Markdown）：
//
//	**"key**":   "**key**":   **"key"**:
var boldKey = regexp.MustCompile(`\*\*"([A-Za-z_][A-Za-z0-9_]*)\*\*"\s*:|"\*\*([A-Za-z_][A-Za-z0-9_]*)\*\*"\s*:|\*\*"([A-Za-z_][A-Za-z0-9_]*)"\*\*\s*:`)

// repairLLMJSON 修掉字段名上的 Markdown 加粗
func repairLLMJSON(text string) string {
	return boldKey.ReplaceAllStringFunc(text, func(match string) string {
		groups := boldKey.FindStringSubmatch(match)
		for _, name := range groups[1:] {
			if name != "" {
				return `"` + name + `":`
			}
		}
		return match
	})
}

// unmarshalLLMJSON 解析模型输出的 JSON：先修字段名再解析（「"**key**"」本身是合法 JSON，只是字段对不上会被静默丢掉，
// 所以不能只在解析失败时才修）；修过的解析失败再退回原文
func unmarshalLLMJSON(text string, target any) error {
	repaired := repairLLMJSON(text)
	if repaired != text {
		if err := json.Unmarshal([]byte(repaired), target); err == nil {
			log.Info("模型输出的 JSON 字段名带 Markdown 加粗，已修复后解析")
			return nil
		}
	}
	return json.Unmarshal([]byte(text), target)
}
