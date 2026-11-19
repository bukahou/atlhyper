package prompts

import (
	"strings"
	"testing"
)

// 告警分析的提示词由后端按界面语言拼（此前前端写死中文：日文界面也发中文指令）
func TestBuildAlertAnalysisMessage(t *testing.T) {
	alerts := []Alert{{Kind: "Pod", Namespace: "shop", Name: "api-1", Reason: "BackOff", Message: "restarting", Timestamp: "2026-11-20T00:00:00Z"}}

	ja := BuildAlertAnalysisMessage(alerts, "ja")
	if !strings.Contains(ja, "アラート 1:") || !strings.Contains(ja, "- リソース名: api-1") || strings.Contains(ja, "告警") {
		t.Errorf("日文消息不对:\n%s", ja)
	}
	zh := BuildAlertAnalysisMessage(alerts, "xx")
	if !strings.Contains(zh, "告警 1:") || !strings.Contains(zh, "- 告警原因: BackOff") {
		t.Errorf("不认识的语言应回落中文:\n%s", zh)
	}

	many := make([]Alert, MaxAlertsPerAnalysis+5)
	if got := strings.Count(BuildAlertAnalysisMessage(many, "zh"), "告警 "); got != MaxAlertsPerAnalysis {
		t.Errorf("最多带 %d 条，实际 %d", MaxAlertsPerAnalysis, got)
	}
}
