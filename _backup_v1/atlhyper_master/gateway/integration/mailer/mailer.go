package mailer

import (
	"AtlHyper/model/integration"
	"fmt"
	"net/smtp"
)

// MailConfig 邮件配置（从数据库读取）
type MailConfig struct {
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	From     string
	To       []string
}

// SendAlertEmail 发送告警邮件（支持 HTML 内容）
func SendAlertEmail(cfg MailConfig, subject string, data integration.AlertGroupData) error {
	// 渲染 HTML 模板
	htmlBody, err := RenderAlertTemplate(data)
	if err != nil {
		return fmt.Errorf("渲染邮件模板失败: %w", err)
	}

	// 构造 SMTP 认证对象
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)

	// 构造邮件内容
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		cfg.From, cfg.To[0], subject, htmlBody,
	))

	// 发送邮件
	return smtp.SendMail(addr, auth, cfg.From, cfg.To, msg)
}
