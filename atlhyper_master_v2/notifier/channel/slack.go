// atlhyper_master_v2/notifier/channel/slack.go
// Slack 通知器（Incoming Webhook）
package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// slackTimeout 一次发送的整体超时
	slackTimeout = 10 * time.Second
	// slackErrorBodyLimit 失败时读取对端响应体的上限（Slack 的错误原因是很短的文本，如 invalid_payload）
	slackErrorBodyLimit = 200
)

// SlackNotifier Slack 通知器。
//
// ⛔ Webhook 地址本身就是凭据：本类型返回的任何错误都不带它（见 safeError）
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
			Timeout: slackTimeout,
			// 不跟随重定向：Webhook 不应该跳转；跟过去等于把告警内容交给另一个地址
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
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
			{Type: "section", Text: &slackTextBlock{Type: "mrkdwn", Text: msg.Body}},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化 Slack 消息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(data))
	if err != nil {
		// 这里的错误多半是地址写错了，原文会带上整个地址：不透出细节
		return errors.New("Slack Webhook 地址无效")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return s.safeError("请求 Slack 失败", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		reason, _ := io.ReadAll(io.LimitReader(resp.Body, slackErrorBodyLimit))
		detail := strings.TrimSpace(string(reason))
		if detail == "" {
			return fmt.Errorf("Slack 返回状态码 %d", resp.StatusCode)
		}
		return errors.New(ScrubSecrets(fmt.Sprintf("Slack 返回状态码 %d: %s", resp.StatusCode, detail), webhookSecrets(s.webhookURL)...))
	}
	return nil
}

// safeError 把网络库的错误改写成不带 Webhook 地址的错误。
//
// http.Client 的错误是 *url.Error，文案形如 `Post "https://…/services/XXX": dial tcp …` ——
// 完整地址就在里面。这里只取它包着的底层原因，再把地址的各段（主机、路径）一并抹掉。
// 返回的是新建的错误，不保留原错误链：调用方用 %v 或 %+v 打印都拿不到地址
func (s *SlackNotifier) safeError(action string, err error) error {
	cause := err
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		cause = urlErr.Err
	}
	return errors.New(ScrubSecrets(action+": "+cause.Error(), webhookSecrets(s.webhookURL)...))
}
