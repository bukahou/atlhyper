// atlhyper_master_v2/notifier/channel/email.go
// Email 通知器
package channel

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailConfig Email 配置
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	UseTLS       bool
	FromAddress  string
	ToAddresses  []string
}

// EmailNotifier Email 通知器
type EmailNotifier struct {
	config EmailConfig
}

// NewEmailNotifier 创建 Email 通知器
func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{config: config}
}

// Name 返回通知器名称
func (e *EmailNotifier) Name() string {
	return "email"
}

// Send 发送邮件
func (e *EmailNotifier) Send(ctx context.Context, msg *Message) error {
	// 构建邮件
	mailMsg := e.buildMailMessage(msg.Subject, msg.Body)

	// 发送
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)
	auth := smtp.PlainAuth("", e.config.SMTPUser, e.config.SMTPPassword, e.config.SMTPHost)

	if e.config.UseTLS {
		// 端口 587 使用 STARTTLS，端口 465 使用直接 TLS
		if e.config.SMTPPort == 587 {
			return e.sendSTARTTLS(addr, auth, mailMsg)
		}
		return e.sendTLS(addr, auth, mailMsg)
	}
	return smtp.SendMail(addr, auth, e.config.FromAddress, e.config.ToAddresses, []byte(mailMsg))
}

// buildMailMessage 构建邮件消息
func (e *EmailNotifier) buildMailMessage(subject, body string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", e.config.FromAddress))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.config.ToAddresses, ",")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return sb.String()
}

// sendSTARTTLS 使用 STARTTLS 发送邮件（端口 587）
func (e *EmailNotifier) sendSTARTTLS(addr string, auth smtp.Auth, msg string) error {
	// 先建立明文连接
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	// 升级到 TLS
	tlsConfig := &tls.Config{ServerName: e.config.SMTPHost}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	// 认证
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	// 发件人
	if err := client.Mail(e.config.FromAddress); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}

	// 收件人
	for _, to := range e.config.ToAddresses {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", to, err)
		}
	}

	// 写入邮件内容
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	return client.Quit()
}

// sendTLS 使用直接 TLS 发送邮件（端口 465）
func (e *EmailNotifier) sendTLS(addr string, auth smtp.Auth, msg string) error {
	tlsConfig := &tls.Config{ServerName: e.config.SMTPHost}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := client.Mail(e.config.FromAddress); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}

	for _, to := range e.config.ToAddresses {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	return client.Quit()
}
