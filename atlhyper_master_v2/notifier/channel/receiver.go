// atlhyper_master_v2/notifier/channel/receiver.go
// 按接收人创建发送器
package channel

import (
	"errors"
	"fmt"

	"AtlHyper/atlhyper_master_v2/database"
)

var (
	// ErrReceiverConfigIncomplete 接收人的配置缺少必填项（含配置解密失败后为空的情况）
	ErrReceiverConfigIncomplete = errors.New("接收人配置不完整")
	// ErrUnsupportedReceiverType 没有对应发送器的接收人类型
	ErrUnsupportedReceiverType = errors.New("不支持的接收人类型")
)

// NewReceiverNotifier 按接收人的类型与配置创建发送器。
// 错误信息只说明缺什么，不带配置内容
func NewReceiverNotifier(receiver *database.NotifyReceiver) (Notifier, error) {
	switch receiver.Type {
	case database.NotifyReceiverSlack:
		if receiver.Slack == nil || receiver.Slack.WebhookURL == "" {
			return nil, fmt.Errorf("%w：未配置 Webhook 地址", ErrReceiverConfigIncomplete)
		}
		return NewSlackNotifier(receiver.Slack.WebhookURL), nil

	case database.NotifyReceiverEmail:
		config := receiver.Email
		if config == nil || config.SMTPHost == "" || config.SMTPPort == 0 || config.SMTPUser == "" ||
			config.SMTPPassword == "" || len(config.ToAddresses) == 0 {
			return nil, fmt.Errorf("%w：SMTP 主机、端口、账号、密码与收件人都必须配置", ErrReceiverConfigIncomplete)
		}
		return NewEmailNotifier(EmailConfig{
			SMTPHost:     config.SMTPHost,
			SMTPPort:     config.SMTPPort,
			SMTPUser:     config.SMTPUser,
			SMTPPassword: config.SMTPPassword,
			UseTLS:       config.SMTPTLS,
			FromAddress:  config.FromAddress,
			ToAddresses:  config.ToAddresses,
		}), nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedReceiverType, receiver.Type)
	}
}
