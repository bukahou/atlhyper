// atlhyper_master_v2/model/notify_receiver_validation.go
// 通知接收人的输入校验（纯函数）。
//
// ⛔ 错误文案会原样返回给调用方：只说哪一项不对，不回显用户填的 Webhook 地址或密码
package model

import (
	"net/mail"
	"net/url"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
)

// ValidateNotifyReceiver 校验接收人。
//
// requireSecret 为真（新建，或修改时换了类型）时凭据必填；为假（修改）时凭据留空 = 保持原值，
// 不带 slack / email 段 = 整段配置保持原值
func ValidateNotifyReceiver(req *NotifyReceiverRequest, requireSecret bool) error {
	if err := validateNotifyName(req.Name); err != nil {
		return err
	}
	switch req.Type {
	case database.NotifyReceiverSlack:
		return validateNotifySlack(req.Slack, requireSecret)
	case database.NotifyReceiverEmail:
		return validateNotifyEmail(req.Email, requireSecret)
	default:
		return invalidNotify("接收人类型须为 slack 或 email")
	}
}

func validateNotifySlack(input *NotifySlackInput, requireSecret bool) error {
	if input == nil || input.WebhookURL == "" {
		if requireSecret {
			return invalidNotify("Slack 接收人必须填写 Webhook 地址")
		}
		return nil
	}
	if len(input.WebhookURL) > maxNotifyWebhookLength {
		return invalidNotify("Webhook 地址过长")
	}
	// 解析失败的错误原文会带上地址本身：不透出
	parsed, err := url.Parse(input.WebhookURL)
	if err != nil || parsed.Host == "" || containsControlRune(input.WebhookURL) {
		return invalidNotify("Webhook 地址不是合法的 URL")
	}
	if parsed.Scheme != "https" {
		return invalidNotify("Webhook 地址必须以 https:// 开头")
	}
	if parsed.User != nil {
		return invalidNotify("Webhook 地址里不能带账号密码")
	}
	return nil
}

func validateNotifyEmail(input *NotifyEmailInput, requireSecret bool) error {
	if input == nil {
		if requireSecret {
			return invalidNotify("邮件接收人必须填写 SMTP 配置")
		}
		return nil
	}
	switch {
	case input.SMTPHost == "":
		return invalidNotify("SMTP 主机不能为空")
	case len(input.SMTPHost) > maxNotifyMatcherLength || strings.ContainsAny(input.SMTPHost, " \t/") || containsControlRune(input.SMTPHost):
		return invalidNotify("SMTP 主机不合法")
	case input.SMTPPort < 1 || input.SMTPPort > 65535:
		return invalidNotify("SMTP 端口须在 1–65535 之间")
	case input.SMTPUser == "":
		return invalidNotify("SMTP 账号不能为空")
	case len(input.SMTPUser) > maxNotifyMatcherLength || containsControlRune(input.SMTPUser):
		return invalidNotify("SMTP 账号不合法")
	case requireSecret && input.SMTPPassword == "":
		return invalidNotify("邮件接收人必须填写 SMTP 密码")
	case len(input.SMTPPassword) > maxNotifySecretLength:
		return invalidNotify("SMTP 密码过长")
	case len(input.ToAddresses) == 0:
		return invalidNotify("至少需要一个收件人")
	case len(input.ToAddresses) > maxNotifyRecipients:
		return invalidNotify("收件人不能超过 %d 个", maxNotifyRecipients)
	}
	if input.FromAddress != "" && !isPlainMailAddress(input.FromAddress) {
		return invalidNotify("发件人地址不是合法的邮箱")
	}
	for _, recipient := range input.ToAddresses {
		if !isPlainMailAddress(recipient) {
			return invalidNotify("收件人地址不是合法的邮箱")
		}
	}
	return nil
}

// isPlainMailAddress 只接受「账号@域名」的纯地址：不带显示名、不带换行（换行能伪造出新的邮件头）
func isPlainMailAddress(address string) bool {
	if len(address) > maxNotifyMatcherLength || containsControlRune(address) {
		return false
	}
	parsed, err := mail.ParseAddress(address)
	return err == nil && parsed.Name == "" && parsed.Address == address
}
