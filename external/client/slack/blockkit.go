package slack

import (
	"NeuroController/internal/types"
	"fmt"
	"strings"
)

// BuildSlackBlockFromAlert 根据告警数据构造 Slack BlockKit 消息格式
//
// ✅ 使用 BlockKit 构造富文本消息，包含标题、概要字段、逐条事件展示（支持 markdown）
// ✅ 限制展示最大告警条数（30条），避免消息过长被 Slack 截断
//
// 参数：
//   - data: 告警聚合数据（包含多个告警项）
//   - subject: 消息主题（用于 Block 顶部标题）
//
// 返回：
//   - map[string]interface{} 类型，符合 Slack BlockKit JSON 结构
func BuildSlackBlockFromAlert(data types.AlertGroupData, subject string) map[string]interface{} {
	// ==========================
	// 📌 初始化 Block 列表：包含标题 + 概要字段
	// ==========================
	blocks := []map[string]interface{}{
		// ✅ 标题 Header Block
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": "🚨 " + subject, // 告警主题加图标
			},
		},

		// ✅ 概要 Section Block（字段显示：Namespace / Node / Alert 数量）
		{
			"type": "section",
			"fields": []map[string]string{
				{"type": "mrkdwn", "text": "*ネームスペース:*\n" + strings.Join(data.NamespaceList, ", ")},
				{"type": "mrkdwn", "text": "*ノード:*\n" + strings.Join(data.NodeList, ", ")},
				{"type": "mrkdwn", "text": "*アラート数:*\n" + fmt.Sprintf("%d", data.AlertCount)},
			},
		},

		// ✅ 分隔符 Divider Block
		{"type": "divider"},
	}

	// ==========================
	// 📌 遍历每条告警数据，生成字段卡片（最多 30 条）
	// ==========================
	const maxItems = 30
	for i, item := range data.Alerts {
		if i >= maxItems {
			break
		}

		// ✅ 每个告警的字段展示（使用 markdown 格式）
		fields := []map[string]string{
			{"type": "mrkdwn", "text": "*リソース種別:*\n" + item.Kind},
			{"type": "mrkdwn", "text": "*名前:*\n" + item.Name},
			{"type": "mrkdwn", "text": "*ネームスペース:*\n" + item.Namespace},
			{"type": "mrkdwn", "text": "*ノード:*\n" + nonEmpty(item.Node)},
			{"type": "mrkdwn", "text": "*時刻:*\n" + item.Time},
			{"type": "mrkdwn", "text": "*理由:*\n" + item.Reason},
			{"type": "mrkdwn", "text": "*詳細:*\n" + item.Message},
		}

		// ✅ 加入 Section Block + Divider
		blocks = append(blocks,
			map[string]interface{}{"type": "section", "fields": fields},
			map[string]interface{}{"type": "divider"},
		)
	}

	// ✅ 返回符合 Slack BlockKit 要求的顶级结构体
	return map[string]interface{}{
		"blocks": blocks,
	}
}

// nonEmpty 辅助函数：若字段为空则返回占位符“（无）”，否则原样返回
func nonEmpty(s string) string {
	if s == "" {
		return "（无）"
	}
	return s
}
