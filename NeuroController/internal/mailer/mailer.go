package mailer

import (
	"NeuroController/config"
	"fmt"
	"net/smtp"
)

// SendAlertEmail 发送告警邮件
func SendAlertEmail(to []string, subject string, data AlertGroupData) error {
	// ✅ 从全局配置读取邮件参数
	mailCfg := config.GlobalConfig.Mailer

	// ✅ 渲染 HTML 模板
	htmlBody, err := RenderAlertTemplate(data)
	if err != nil {
		return fmt.Errorf("渲染邮件模板失败: %w", err)
	}

	auth := smtp.PlainAuth("", mailCfg.Username, mailCfg.Password, mailCfg.SMTPHost)
	addr := fmt.Sprintf("%s:%s", mailCfg.SMTPHost, mailCfg.SMTPPort)

	// ✅ 构造邮件内容（支持 HTML）
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		mailCfg.From, to[0], subject, htmlBody,
	))

	// ✅ 发送邮件
	return smtp.SendMail(addr, auth, mailCfg.From, to, msg)
}
