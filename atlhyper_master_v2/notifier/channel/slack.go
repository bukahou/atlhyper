// atlhyper_master_v2/notifier/channel/slack.go
// Slack 通知渠道
package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/notifier"
)

// SlackChannel Slack 通知渠道
type SlackChannel struct {
	webhookURL string
	httpClient *http.Client
}

// SlackConfig Slack 配置
type SlackConfig struct {
	WebhookURL string
}

// NewSlackChannel 创建 Slack 通知渠道
func NewSlackChannel(cfg SlackConfig) *SlackChannel {
	return &SlackChannel{
		webhookURL: cfg.WebhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Type 返回渠道类型
func (c *SlackChannel) Type() string {
	return "slack"
}

// Send 发送 Slack 通知
func (c *SlackChannel) Send(ctx context.Context, msg *notifier.Message) error {
	// 构建 Slack BlockKit 消息
	payload := c.buildBlockKitPayload(msg)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// buildBlockKitPayload 构建 Slack BlockKit 消息体
func (c *SlackChannel) buildBlockKitPayload(msg *notifier.Message) map[string]interface{} {
	blocks := []interface{}{}

	// 1. Header
	emoji := c.severityEmoji(msg.Severity)
	headerText := fmt.Sprintf("%s %s", emoji, msg.Title)
	if count, ok := msg.Fields["告警总数"]; ok {
		headerText = fmt.Sprintf("%s（共 %s 条）", headerText, count)
	}

	blocks = append(blocks, map[string]interface{}{
		"type": "header",
		"text": map[string]interface{}{
			"type":  "plain_text",
			"text":  headerText,
			"emoji": true,
		},
	})

	// 2. Divider
	blocks = append(blocks, map[string]interface{}{"type": "divider"})

	// 3. Stats section (fields)
	if len(msg.Fields) > 0 {
		var fieldBlocks []interface{}
		for k, v := range msg.Fields {
			if k == "告警总数" {
				continue // 已在标题中显示
			}
			fieldBlocks = append(fieldBlocks, map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*%s*\n%s", k, v),
			})
		}
		if len(fieldBlocks) > 0 {
			blocks = append(blocks, map[string]interface{}{
				"type":   "section",
				"fields": fieldBlocks,
			})
			blocks = append(blocks, map[string]interface{}{"type": "divider"})
		}
	}

	// 4. Content (alert details)
	if msg.Content != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": msg.Content,
			},
		})
	}

	// 5. Divider
	blocks = append(blocks, map[string]interface{}{"type": "divider"})

	// 6. Footer
	blocks = append(blocks, map[string]interface{}{
		"type": "context",
		"elements": []interface{}{
			map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("⏰ %s | *AtlHyper*", time.Now().Format("2006-01-02 15:04:05")),
			},
		},
	})

	return map[string]interface{}{
		"blocks": blocks,
	}
}

// severityEmoji 返回严重级别对应的 emoji
func (c *SlackChannel) severityEmoji(severity string) string {
	switch severity {
	case notifier.SeverityCritical:
		return "🚨"
	case notifier.SeverityWarning:
		return "⚠️"
	default:
		return "ℹ️"
	}
}

// 确保实现了接口
var _ Channel = (*SlackChannel)(nil)
