// atlhyper_master/gateway/integration/slack/blockkit_helpers.go
// Slack BlockKit 辅助函数
package slack

import (
	"AtlHyper/model/integration"
	"fmt"
	"strings"
	"time"
)

// buildAlertItem 构建单条告警区块
func buildAlertItem(item integration.AlertItem, index int) map[string]interface{} {
	sev := item.Severity
	if sev == "" {
		sev = "Info"
	}
	cfg, ok := severityConfig[sev]
	if !ok {
		cfg = severityConfig["Info"]
	}

	// 构建告警标题行
	title := fmt.Sprintf("%s *#%d* `%s` %s/%s",
		cfg.emoji, index, item.Kind, item.Namespace, item.Name)

	// 构建详情文本
	details := []string{}
	if item.ClusterID != "" {
		details = append(details, fmt.Sprintf("集群: `%s`", item.ClusterID))
	}
	if item.Node != "" {
		details = append(details, fmt.Sprintf("节点: `%s`", item.Node))
	}
	if item.Reason != "" {
		details = append(details, fmt.Sprintf("原因: *%s*", item.Reason))
	}
	if item.Time != "" {
		details = append(details, fmt.Sprintf("时间: %s", item.Time))
	}

	detailText := strings.Join(details, "  |  ")

	// 消息内容（可能较长，单独一行）
	msgText := ""
	if item.Message != "" {
		msg := item.Message
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		msgText = "\n> " + msg
	}

	return map[string]interface{}{
		"type": "section",
		"text": map[string]string{
			"type": "mrkdwn",
			"text": title + "\n" + detailText + msgText,
		},
	}
}

// buildOverflowNotice 构建溢出提示
func buildOverflowNotice(remaining int) map[string]interface{} {
	return map[string]interface{}{
		"type": "context",
		"elements": []map[string]string{
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("⚡ *还有 %d 条告警未显示*，请登录控制台查看完整列表", remaining),
			},
		},
	}
}

// buildFooter 构建页脚
func buildFooter() map[string]interface{} {
	return map[string]interface{}{
		"type": "context",
		"elements": []map[string]string{
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("📅 %s  |  🤖 AtlHyper 告警系统",
					time.Now().Format("2006-01-02 15:04:05")),
			},
		},
	}
}

// divider 分隔线
func divider() map[string]interface{} {
	return map[string]interface{}{"type": "divider"}
}

// formatList 格式化列表显示
func formatList(items []string, max int) string {
	if len(items) == 0 {
		return "_无_"
	}
	if len(items) <= max {
		return "`" + strings.Join(items, "` `") + "`"
	}
	visible := items[:max]
	return "`" + strings.Join(visible, "` `") + fmt.Sprintf("` +%d", len(items)-max)
}

// getSeverityEmoji 获取严重级别对应的 emoji
func getSeverityEmoji(severity string) string {
	if cfg, ok := severityConfig[severity]; ok {
		return cfg.emoji
	}
	return "🔵"
}
