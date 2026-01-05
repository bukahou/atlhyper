package slack

import (
	"AtlHyper/model/integration"
	"fmt"
	"strings"
)

func BuildSlackBlockFromAlert(data integration.AlertGroupData, subject string) map[string]interface{} {
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": "🚨 " + subject,
			},
		},
		// 概览：集群数组渲染为多行（字段内文本）
		{
			"type": "section",
			"fields": []map[string]string{
				{"type": "mrkdwn", "text": "*クラスタ:*\n" + listAsBullets(data.ClusterID, 10)}, // ← 集群数组
				{"type": "mrkdwn", "text": "*ネームスペース:*\n" + nonEmpty(strings.Join(data.NamespaceList, ", "))},
				{"type": "mrkdwn", "text": "*ノード:*\n" + nonEmpty(strings.Join(data.NodeList, ", "))},
				{"type": "mrkdwn", "text": "*アラート数:*\n" + fmt.Sprintf("%d", data.AlertCount)},
			},
		},
		{"type": "divider"},
	}

	// 明细（含クラスタ；不展示重要度）
	const maxItems = 30
	for i, item := range data.Alerts {
		if i >= maxItems {
			break
		}
		fields := []map[string]string{
			{"type": "mrkdwn", "text": "*クラスタ:*\n" + nonEmpty(item.ClusterID)},
			{"type": "mrkdwn", "text": "*リソース種別:*\n" + nonEmpty(item.Kind)},
			{"type": "mrkdwn", "text": "*名前:*\n" + nonEmpty(item.Name)},
			{"type": "mrkdwn", "text": "*ネームスペース:*\n" + nonEmpty(item.Namespace)},
			{"type": "mrkdwn", "text": "*ノード:*\n" + nonEmpty(item.Node)},
			{"type": "mrkdwn", "text": "*時刻:*\n" + nonEmpty(item.Time)},
			{"type": "mrkdwn", "text": "*理由:*\n" + nonEmpty(item.Reason)},
			{"type": "mrkdwn", "text": "*詳細:*\n" + nonEmpty(item.Message)},
		}
		blocks = append(blocks,
			map[string]interface{}{"type": "section", "fields": fields},
			map[string]interface{}{"type": "divider"},
		)
	}

	return map[string]interface{}{"blocks": blocks}
}

func nonEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "（无）"
	}
	return s
}

// listAsBullets 把数组渲染为多行列表（• a\n• b\n...），并在超长时截断显示
func listAsBullets(arr []string, max int) string {
	if len(arr) == 0 {
		return "（无）"
	}
	if max <= 0 || len(arr) <= max {
		return "• " + strings.Join(arr, "\n• ")
	}
	visible := arr[:max]
	hidden := len(arr) - max
	return "• " + strings.Join(visible, "\n• ") + fmt.Sprintf("\n… 他 %d 件", hidden)
}
