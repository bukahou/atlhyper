// atlhyper_master_v2/notifier/channel/receiver.go
// 按接收人创建发送器
package channel

import (
	"context"
	"errors"
	"fmt"

	"AtlHyper/atlhyper_master_v2/database"
)

var (
	// ErrReceiverConfigIncomplete 接收人的配置缺少必填项（含配置解密失败后为空的情况）
	ErrReceiverConfigIncomplete = errors.New("接收人配置不完整")
	// ErrSMTPNotConfigured 邮件接收人要用的全局发件设置还没配齐。也算「配置不完整」
	ErrSMTPNotConfigured = fmt.Errorf("%w：未配置发件设置（通知设置 → 接收人 → 邮件发件设置）", ErrReceiverConfigIncomplete)
	// ErrUnsupportedReceiverType 没有对应发送器的接收人类型
	ErrUnsupportedReceiverType = errors.New("不支持的接收人类型")
)

// SMTPSettingsReader 读全局发件设置。没有设置时返回 (nil, nil)
type SMTPSettingsReader interface {
	Get(ctx context.Context) (*database.NotifySMTP, error)
}

// NewReceiverNotifier 按接收人的类型与配置创建发送器。
// 邮件接收人用全局发件设置 smtpSettings（Slack 不需要，可传 nil）。
// 错误信息只说明缺什么，不带配置内容
func NewReceiverNotifier(receiver *database.NotifyReceiver, smtpSettings *database.NotifySMTP) (Notifier, error) {
	switch receiver.Type {
	case database.NotifyReceiverSlack:
		if receiver.Slack == nil || receiver.Slack.WebhookURL == "" {
			return nil, fmt.Errorf("%w：未配置 Webhook 地址", ErrReceiverConfigIncomplete)
		}
		return NewSlackNotifier(receiver.Slack.WebhookURL), nil

	case database.NotifyReceiverEmail:
		if receiver.Email == nil || len(receiver.Email.ToAddresses) == 0 {
			return nil, fmt.Errorf("%w：未配置收件人", ErrReceiverConfigIncomplete)
		}
		if !smtpSettings.Configured() {
			return nil, ErrSMTPNotConfigured
		}
		return NewEmailNotifier(EmailConfig{
			SMTPHost:     smtpSettings.Host,
			SMTPPort:     smtpSettings.Port,
			SMTPUser:     smtpSettings.User,
			SMTPPassword: smtpSettings.Password,
			UseTLS:       smtpSettings.TLS,
			FromAddress:  smtpSettings.FromAddress,
			ToAddresses:  receiver.Email.ToAddresses,
		}), nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedReceiverType, receiver.Type)
	}
}

// ReceiverSender 把消息发给一个接收人：按它的类型与配置创建发送器并发送。
// 邮件接收人每次发送时读一次全局发件设置（单行，读取很轻），改完设置立即生效。
//
// 返回的错误已经去掉凭据（Webhook 地址、SMTP 密码），可以直接写日志与送达记录
type ReceiverSender struct {
	smtp SMTPSettingsReader
}

// NewReceiverSender 创建 ReceiverSender
func NewReceiverSender(smtp SMTPSettingsReader) *ReceiverSender {
	return &ReceiverSender{smtp: smtp}
}

// Send 发送一条消息
func (s *ReceiverSender) Send(ctx context.Context, receiver *database.NotifyReceiver, msg *Message) error {
	var smtpSettings *database.NotifySMTP
	if receiver.Type == database.NotifyReceiverEmail {
		settings, err := s.smtp.Get(ctx)
		if err != nil {
			return fmt.Errorf("读取发件设置失败: %w", err)
		}
		smtpSettings = settings
	}
	notifier, err := NewReceiverNotifier(receiver, smtpSettings)
	if err != nil {
		return err
	}
	if err := notifier.Send(ctx, msg); err != nil {
		secrets := ReceiverSecrets(receiver)
		if smtpSettings != nil && smtpSettings.Password != "" {
			secrets = append(secrets, smtpSettings.Password)
		}
		return errors.New(ScrubSecrets(err.Error(), secrets...))
	}
	return nil
}
