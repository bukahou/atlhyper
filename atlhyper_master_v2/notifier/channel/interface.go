// atlhyper_master_v2/notifier/channel/interface.go
// 通知渠道接口与工厂
package channel

import (
	"context"
	"encoding/json"
	"errors"

	"AtlHyper/atlhyper_master_v2/database"
)

// Message 渲染后的消息
type Message struct {
	Subject string // 主题（Email 用）
	Body    string // 正文
	Format  string // 格式：text, markdown, html
}

// Notifier 通知器接口
type Notifier interface {
	Name() string
	Send(ctx context.Context, msg *Message) error
}

// Factory 通知器工厂
type Factory struct{}

// NewFactory 创建工厂
func NewFactory() *Factory {
	return &Factory{}
}

// Create 根据渠道配置创建 Notifier
func (f *Factory) Create(ch *database.NotifyChannel) (Notifier, error) {
	switch ch.Type {
	case "slack":
		var cfg database.SlackConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return nil, errors.New("invalid slack config")
		}
		if cfg.WebhookURL == "" {
			return nil, errors.New("slack webhook url required")
		}
		return NewSlackNotifier(cfg.WebhookURL), nil

	case "email":
		var cfg database.EmailConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return nil, errors.New("invalid email config")
		}
		if cfg.SMTPHost == "" || len(cfg.ToAddresses) == 0 {
			return nil, errors.New("email smtp host and recipients required")
		}
		return NewEmailNotifier(EmailConfig{
			SMTPHost:     cfg.SMTPHost,
			SMTPPort:     cfg.SMTPPort,
			SMTPUser:     cfg.SMTPUser,
			SMTPPassword: cfg.SMTPPassword,
			UseTLS:       cfg.SMTPTLS,
			FromAddress:  cfg.FromAddress,
			ToAddresses:  cfg.ToAddresses,
		}), nil

	default:
		return nil, errors.New("unsupported channel type: " + ch.Type)
	}
}
