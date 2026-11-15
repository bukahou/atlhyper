package enricher

import (
	"encoding/json"
	"testing"
)

// 模型在字段名里混进 Markdown 加粗（2026-11-15 dev 实测：gpt-oss 输出 **"rootCauseAnalysis**"）：修好后能解析，内容不变
func TestRepairLLMJSON_MarkdownBoldKeys(t *testing.T) {
	cases := map[string]string{
		"粗体包住引号与键名后半": `{ "summary": "s", **"rootCauseAnalysis**": "r", "confidence": 0.9 }`,
		"粗体在引号内":      `{ "summary": "s", "**rootCauseAnalysis**": "r", "confidence": 0.9 }`,
		"粗体在引号外":      `{ "summary": "s", **"rootCauseAnalysis"**: "r", "confidence": 0.9 }`,
	}
	for name, raw := range cases {
		var parsed struct {
			Summary           string  `json:"summary"`
			RootCauseAnalysis string  `json:"rootCauseAnalysis"`
			Confidence        float64 `json:"confidence"`
		}
		if err := unmarshalLLMJSON(raw, &parsed); err != nil {
			t.Errorf("%s：修复后仍无法解析: %v", name, err)
			continue
		}
		if parsed.Summary != "s" || parsed.RootCauseAnalysis != "r" || parsed.Confidence != 0.9 {
			t.Errorf("%s：修复后内容不对: %+v", name, parsed)
		}
	}
}

// 值里本来就有的 Markdown（如 **Direct cause**）不能被改动
func TestRepairLLMJSON_KeepsMarkdownInValues(t *testing.T) {
	raw := `{ "summary": "x", **"rootCauseAnalysis**": "1. **Pod spec** – ok\n**Direct cause**: y" }`
	var parsed struct {
		RootCauseAnalysis string `json:"rootCauseAnalysis"`
	}
	if err := json.Unmarshal([]byte(repairLLMJSON(raw)), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.RootCauseAnalysis != "1. **Pod spec** – ok\n**Direct cause**: y" {
		t.Errorf("值里的 Markdown 被改动了: %q", parsed.RootCauseAnalysis)
	}
}

// 解析入口：模型输出带这种错误时，报告的各字段照常解析出来，不再整段原文塞进摘要
func TestParseAnalysisResult_RepairsBoldKeys(t *testing.T) {
	raw := "```json\n" + `{ "summary": "Deployment down", **"rootCauseAnalysis**": "exit 1", "recommendations": [ { "priority": 1, "action": "fix", "reason": "r", "impact": "i" } ], "confidence": 0.95 }` + "\n```"
	got := parseAnalysisResult(raw, "inc-1")
	if got.Summary != "Deployment down" || got.RootCauseAnalysis != "exit 1" || len(got.Recommendations) != 1 {
		t.Errorf("应解析出各字段: %+v", got)
	}
}
