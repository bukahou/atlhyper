package slack

import (
	"NeuroController/internal/types"
	"fmt"
	"strings"
)

// BuildSlackBlockFromAlert 构造 Slack BlockKit 消息 JSON
func BuildSlackBlockFromAlert(data types.AlertGroupData, subject string) map[string]interface{} {
	blocks := []map[string]interface{}{
		// 🚨 顶部标题
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": "🚨 " + subject,
			},
		},
		// 📊 命名空间 / 节点 / 告警数量（字段卡片）
		{
			"type": "section",
			"fields": []map[string]string{
				{"type": "mrkdwn", "text": "*命名空间:*\n" + strings.Join(data.NamespaceList, ", ")},
				{"type": "mrkdwn", "text": "*节点:*\n" + strings.Join(data.NodeList, ", ")},
				{"type": "mrkdwn", "text": "*告警数量:*\n" + fmt.Sprintf("%d", data.AlertCount)},
			},
		},
		{"type": "divider"},
	}

	// 🧾 每条告警转为卡片式 fields 展示（限制最多 5 条）
	const maxItems = 30
	for i, item := range data.Alerts {
		if i >= maxItems {
			break
		}

		fields := []map[string]string{
			{"type": "mrkdwn", "text": "*资源类型:*\n" + item.Kind},
			{"type": "mrkdwn", "text": "*名称:*\n" + item.Name},
			{"type": "mrkdwn", "text": "*命名空间:*\n" + item.Namespace},
			{"type": "mrkdwn", "text": "*节点:*\n" + nonEmpty(item.Node)},
			{"type": "mrkdwn", "text": "*时间:*\n" + item.Time},
			{"type": "mrkdwn", "text": "*原因:*\n" + item.Reason},
			{"type": "mrkdwn", "text": "*描述:*\n" + item.Message},
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
