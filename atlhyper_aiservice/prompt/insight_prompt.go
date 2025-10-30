package prompt

import "fmt"

// BuildInsightPrompt —— 通用运维洞察提示词模板
// ------------------------------------------------------------
// 输入系统摘要，生成用于 AI 模型的完整 Prompt。
func BuildInsightPrompt(summary string) string {
	return fmt.Sprintf(`
你是一名经验丰富的运维专家。
请根据以下描述，分析问题现象、可能原因、影响及建议。
要求以严格的 JSON 格式输出。

输入内容：
%s

输出模板（必须为合法 JSON）：
{
  "problem": "string —— 问题现象的简要描述",
  "possibleCause": "string —— 可能的根本原因",
  "impact": "string —— 潜在影响范围",
  "recommendation": "string —— 修复或排查建议",
  "confidence": 0.0 —— 分析置信度，范围 0~1"
}

请仅输出合法 JSON，不要添加解释性文字。
`, summary)
}
