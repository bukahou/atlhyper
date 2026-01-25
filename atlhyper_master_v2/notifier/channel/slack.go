// atlhyper_master_v2/notifier/channel/slack.go
// Slack 通知器
package channel

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
	client     *http.Client
}

// slackPayload Slack Webhook 请求体
type slackPayload struct {
	Text   string       `json:"text"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

type slackBlock struct {
	Type string          `json:"type"`
	Text *slackTextBlock `json:"text,omitempty"`
}

type slackTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewSlackNotifier 创建 Slack 通知器
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name 返回通知器名称
func (s *SlackNotifier) Name() string {
	return "slack"
}

// Send 发送消息到 Slack
func (s *SlackNotifier) Send(ctx context.Context, msg *Message) error {
	payload := &slackPayload{
		Text: msg.Subject,
		Blocks: []slackBlock{
			{
				Type: "section",
				Text: &slackTextBlock{
					Type: "mrkdwn",
					Text: msg.Body,
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}
