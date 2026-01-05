package mailer

import (
	"AtlHyper/atlhyper_master/config"
	"AtlHyper/model/integration"
	"fmt"
	"net/smtp"
)

// =======================================================================================
// ✅ SendAlertEmail - 发送告警邮件（支持 HTML 内容）
//
// 📌 参数：
//     - to      : 接收者邮箱地址列表（支持多人）
//     - subject : 邮件标题（通常为告警级别 + 事件摘要）
//     - data    : 告警内容数据，将被渲染为 HTML 模板
//
// 🧩 调用链：
//     - RenderAlertTemplate → 构造 HTML 内容
//     - smtp.SendMail → 使用配置中提供的 SMTP 凭证发送邮件
//
// ⚠️ 要求：
//     - config.GlobalConfig.Mailer 必须提前初始化（含 SMTPHost, Username, Password 等）
//     - 模板渲染失败或 SMTP 错误将返回非 nil 错误
// =======================================================================================
func SendAlertEmail(to []string, subject string, data integration.AlertGroupData) error {
	// ✅ 从全局配置读取邮件参数
	mailCfg := config.GlobalConfig.Mailer

	// ✅ 渲染 HTML 模板
	htmlBody, err := RenderAlertTemplate(data)
	if err != nil {
		return fmt.Errorf("渲染邮件模板失败: %w", err)
	}

	// ✅ 构造 SMTP 认证对象（PlainAuth 使用用户名密码认证）
	auth := smtp.PlainAuth("", mailCfg.Username, mailCfg.Password, mailCfg.SMTPHost)
	addr := fmt.Sprintf("%s:%s", mailCfg.SMTPHost, mailCfg.SMTPPort)

	// ✅ 构造邮件内容（From、To、Subject、Content-Type、HTML Body）
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		mailCfg.From, to[0], subject, htmlBody,
	))

	// ✅ 使用 smtp.SendMail 发送邮件
	return smtp.SendMail(addr, auth, mailCfg.From, to, msg)
}
