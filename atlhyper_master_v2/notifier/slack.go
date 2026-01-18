// atlhyper_master_v2/notifier/slack.go
// Slack 通知发送
package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier Slack 通知器
type SlackNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// SlackConfig Slack 配置
type SlackConfig struct {
	WebhookURL string
}

// NewSlackNotifier 创建 Slack 通知器
func NewSlackNotifier(cfg SlackConfig) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: cfg.WebhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Type 返回通知类型
func (n *SlackNotifier) Type() string {
	return "slack"
}

// Send 发送 Slack 通知
func (n *SlackNotifier) Send(ctx context.Context, msg *Message) error {
	// 构建 Slack 消息
	payload := n.buildPayload(msg)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// buildPayload 构建 Slack 消息体
func (n *SlackNotifier) buildPayload(msg *Message) map[string]interface{} {
	// 根据严重程度设置颜色
	color := "#36a64f" // green
	switch msg.Severity {
	case "warning":
		color = "#ff9800" // orange
	case "critical":
		color = "#f44336" // red
	}

	// 构建 fields
	var fields []map[string]interface{}
	for k, v := range msg.Fields {
		fields = append(fields, map[string]interface{}{
			"title": k,
			"value": v,
			"short": true,
		})
	}

	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  msg.Title,
				"text":   msg.Content,
				"fields": fields,
				"footer": "AtlHyper",
				"ts":     time.Now().Unix(),
			},
		},
	}
}

// 确保实现了接口
var _ Notifier = (*SlackNotifier)(nil)
