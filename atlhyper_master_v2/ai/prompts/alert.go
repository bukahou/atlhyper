package prompts

import (
	"fmt"
	"strings"
)

// MaxAlertsPerAnalysis 一次告警分析最多带几条告警（控制提示词长度）
const MaxAlertsPerAnalysis = 20

// Alert 用户在告警页选中、交给 AI 分析的一条告警（前端只传结构化字段，提示词由这里拼）
type Alert struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// alertTexts 告警分析消息的固定文案（按用户界面的语言出；模型按同一语言回答）
type alertTexts struct {
	header, item, kind, namespace, name, reason, message, time, footer string
}

var alertTextsByLanguage = map[string]alertTexts{
	"zh": {
		header: "[以下是用户选择的告警信息，请分析并给出诊断结论]", item: "告警 %d:",
		kind: "资源类型", namespace: "命名空间", name: "资源名称", reason: "告警原因", message: "告警消息", time: "发生时间",
		footer: "请查询相关资源的状态、日志和事件，分析告警原因并给出修复建议。",
	},
	"ja": {
		header: "[以下はユーザーが選択したアラートです。分析して診断結果を示してください]", item: "アラート %d:",
		kind: "リソース種別", namespace: "名前空間", name: "リソース名", reason: "原因", message: "メッセージ", time: "発生時刻",
		footer: "関連リソースの状態・ログ・イベントを調べ、アラートの原因と修正方法を示してください。",
	},
}

// BuildAlertAnalysisMessage 告警列表 → 发给模型的用户消息。language 不认识时用中文；
// 超过 MaxAlertsPerAnalysis 条只取前面的
func BuildAlertAnalysisMessage(alerts []Alert, language string) string {
	texts, ok := alertTextsByLanguage[language]
	if !ok {
		texts = alertTextsByLanguage["zh"]
	}
	if len(alerts) > MaxAlertsPerAnalysis {
		alerts = alerts[:MaxAlertsPerAnalysis]
	}
	var b strings.Builder
	b.WriteString(texts.header + "\n\n")
	for i, a := range alerts {
		fmt.Fprintf(&b, texts.item+"\n", i+1)
		for _, field := range [][2]string{
			{texts.kind, a.Kind}, {texts.namespace, a.Namespace}, {texts.name, a.Name},
			{texts.reason, a.Reason}, {texts.message, a.Message}, {texts.time, a.Timestamp},
		} {
			fmt.Fprintf(&b, "- %s: %s\n", field[0], field[1])
		}
		b.WriteString("\n")
	}
	b.WriteString(texts.footer)
	return b.String()
}
