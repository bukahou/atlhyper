// atlhyper_master/gateway/integration/slack/blockkit.go
// Slack BlockKit 消息构建器 - 美化版
package slack

import (
	"AtlHyper/model/integration"
	"fmt"
	"strings"
)

// 严重级别对应的 emoji 和颜色
var severityConfig = map[string]struct {
	emoji string
	color string
}{
	"Critical": {"🔴", "#E53935"},
	"Warning":  {"🟠", "#FB8C00"},
	"Info":     {"🔵", "#1E88E5"},
	"Normal":   {"🟢", "#43A047"},
}

// BuildSlackBlockFromAlert 构建美化的 Slack BlockKit 消息
func BuildSlackBlockFromAlert(data integration.AlertGroupData, subject string) map[string]interface{} {
	blocks := []map[string]interface{}{}

	// 1. 标题区域
	blocks = append(blocks, buildHeader(subject, data.AlertCount))
	blocks = append(blocks, divider())

	// 2. 摘要统计区域
	blocks = append(blocks, buildSummarySection(data))
	blocks = append(blocks, divider())

	// 3. 告警明细（最多显示15条，避免消息过长）
	maxItems := 15
	for i, item := range data.Alerts {
		if i >= maxItems {
			blocks = append(blocks, buildOverflowNotice(len(data.Alerts)-maxItems))
			break
		}
		blocks = append(blocks, buildAlertItem(item, i+1))
	}

	// 4. 页脚
	blocks = append(blocks, divider())
	blocks = append(blocks, buildFooter())

	return map[string]interface{}{"blocks": blocks}
}

// buildHeader 构建标题区块
func buildHeader(subject string, count int) map[string]interface{} {
	emoji := "🚨"
	if count <= 3 {
		emoji = "⚠️"
	}
	return map[string]interface{}{
		"type": "header",
		"text": map[string]string{
			"type": "plain_text",
			"text": fmt.Sprintf("%s %s（共 %d 条）", emoji, subject, count),
		},
	}
}

// buildSummarySection 构建摘要区块
func buildSummarySection(data integration.AlertGroupData) map[string]interface{} {
	// 统计各严重级别数量
	severityCounts := make(map[string]int)
	for _, alert := range data.Alerts {
		sev := alert.Severity
		if sev == "" {
			sev = "Info"
		}
		severityCounts[sev]++
	}

	// 构建严重级别统计文本
	var sevParts []string
	for _, sev := range []string{"Critical", "Warning", "Info", "Normal"} {
		if count, ok := severityCounts[sev]; ok && count > 0 {
			cfg := severityConfig[sev]
			sevParts = append(sevParts, fmt.Sprintf("%s %s: %d", cfg.emoji, sev, count))
		}
	}
	sevText := strings.Join(sevParts, "  |  ")
	if sevText == "" {
		sevText = "无统计信息"
	}

	return map[string]interface{}{
		"type": "section",
		"fields": []map[string]string{
			{"type": "mrkdwn", "text": "*📊 级别分布*\n" + sevText},
			{"type": "mrkdwn", "text": "*🏷️ 集群*\n" + formatList(data.ClusterID, 5)},
			{"type": "mrkdwn", "text": "*📁 命名空间*\n" + formatList(data.NamespaceList, 5)},
			{"type": "mrkdwn", "text": "*🖥️ 节点*\n" + formatList(data.NodeList, 5)},
		},
	}
}
