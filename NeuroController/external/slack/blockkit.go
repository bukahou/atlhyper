package slack

import (
	"NeuroController/internal/types"
	"fmt"
	"strings"
)

// BuildSlackBlockFromAlert 构造 Slack BlockKit 消息 JSON
func BuildSlackBlockFromAlert(data types.AlertGroupData, subject string) map[string]interface{} {
	blocks := []map[string]interface{}{
		// 🚨 ヘッダータイトル
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": "🚨 " + subject,
			},
		},
		// 📊 ネームスペース / ノード / アラート数（概要フィールド）
		{
			"type": "section",
			"fields": []map[string]string{
				{"type": "mrkdwn", "text": "*ネームスペース:*\n" + strings.Join(data.NamespaceList, ", ")},
				{"type": "mrkdwn", "text": "*ノード:*\n" + strings.Join(data.NodeList, ", ")},
				{"type": "mrkdwn", "text": "*アラート数:*\n" + fmt.Sprintf("%d", data.AlertCount)},
			},
		},
		{"type": "divider"},
	}

	// 🧾 各アラートをフィールドカードに変換（最大 30 件まで）
	const maxItems = 30
	for i, item := range data.Alerts {
		if i >= maxItems {
			break
		}

		fields := []map[string]string{
			{"type": "mrkdwn", "text": "*リソース種別:*\n" + item.Kind},
			{"type": "mrkdwn", "text": "*名前:*\n" + item.Name},
			{"type": "mrkdwn", "text": "*ネームスペース:*\n" + item.Namespace},
			{"type": "mrkdwn", "text": "*ノード:*\n" + nonEmpty(item.Node)},
			{"type": "mrkdwn", "text": "*時刻:*\n" + item.Time},
			{"type": "mrkdwn", "text": "*理由:*\n" + item.Reason},
			{"type": "mrkdwn", "text": "*詳細:*\n" + item.Message},
		}

		blocks = append(blocks,
			map[string]interface{}{"type": "section", "fields": fields},
			map[string]interface{}{"type": "divider"},
		)
	}

	return map[string]interface{}{
		"blocks": blocks,
	}
}

// nonEmpty returns a fallback string if input is empty
func nonEmpty(s string) string {
	if s == "" {
		return "（无）"
	}
	return s
}
