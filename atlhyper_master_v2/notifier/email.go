// atlhyper_master_v2/notifier/email.go
// Email 通知发送
package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailNotifier Email 通知器
type EmailNotifier struct {
	host        string
	port        int
	user        string
	password    string
	useTLS      bool
	fromAddress string
	toAddresses []string
}

// EmailConfig Email 配置
type EmailConfig struct {
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPassword string
	SMTPTLS     bool
	FromAddress string
	ToAddresses []string
}

// NewEmailNotifier 创建 Email 通知器
func NewEmailNotifier(cfg EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		host:        cfg.SMTPHost,
		port:        cfg.SMTPPort,
		user:        cfg.SMTPUser,
		password:    cfg.SMTPPassword,
		useTLS:      cfg.SMTPTLS,
		fromAddress: cfg.FromAddress,
		toAddresses: cfg.ToAddresses,
	}
}

// Type 返回通知类型
func (n *EmailNotifier) Type() string {
	return "email"
}

// Send 发送 Email 通知
func (n *EmailNotifier) Send(ctx context.Context, msg *Message) error {
	// 构建邮件内容
	subject := n.buildSubject(msg)
	body := n.buildBody(msg)

	// 构建邮件头
	headers := make(map[string]string)
	headers["From"] = n.fromAddress
	headers["To"] = strings.Join(n.toAddresses, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 构建完整邮件
	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", n.host, n.port)
	auth := smtp.PlainAuth("", n.user, n.password, n.host)

	if n.useTLS {
		return n.sendWithTLS(addr, auth, message.String())
	}
	return smtp.SendMail(addr, auth, n.fromAddress, n.toAddresses, []byte(message.String()))
}

// sendWithTLS 使用 TLS 发送邮件
func (n *EmailNotifier) sendWithTLS(addr string, auth smtp.Auth, message string) error {
	tlsConfig := &tls.Config{
		ServerName: n.host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, n.host)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to auth: %w", err)
	}

	if err := client.Mail(n.fromAddress); err != nil {
		return fmt.Errorf("failed to set from: %w", err)
	}

	for _, to := range n.toAddresses {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set to %s: %w", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get writer: %w", err)
	}

	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

// buildSubject 构建邮件主题
func (n *EmailNotifier) buildSubject(msg *Message) string {
	prefix := "[INFO]"
	switch msg.Severity {
	case "warning":
		prefix = "[WARNING]"
	case "critical":
		prefix = "[CRITICAL]"
	}
	return fmt.Sprintf("%s %s", prefix, msg.Title)
}

// buildBody 构建邮件正文
func (n *EmailNotifier) buildBody(msg *Message) string {
	var body strings.Builder

	// HTML 头部和样式
	body.WriteString(`<html>
<head>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
.container { max-width: 600px; margin: 0 auto; background: #fff; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.header { padding: 20px; border-bottom: 1px solid #eee; }
.header h2 { margin: 0; color: #333; }
.content { padding: 20px; }
.content pre { background: #f8f9fa; padding: 12px; border-radius: 4px; white-space: pre-wrap; font-size: 13px; }
.fields { margin-top: 16px; }
.fields table { width: 100%; border-collapse: collapse; }
.fields td { padding: 8px 12px; border: 1px solid #eee; }
.fields td:first-child { background: #f8f9fa; font-weight: 600; width: 120px; }
.footer { padding: 16px 20px; border-top: 1px solid #eee; color: #666; font-size: 12px; }
.severity-critical { color: #dc3545; }
.severity-warning { color: #fd7e14; }
.severity-info { color: #0d6efd; }
</style>
</head>
<body>
<div class="container">`)

	// Header
	severityClass := "severity-info"
	switch msg.Severity {
	case "critical":
		severityClass = "severity-critical"
	case "warning":
		severityClass = "severity-warning"
	}
	body.WriteString(fmt.Sprintf(`<div class="header"><h2 class="%s">%s</h2></div>`, severityClass, msg.Title))

	// Content
	body.WriteString(`<div class="content">`)
	if msg.Content != "" {
		body.WriteString(fmt.Sprintf("<pre>%s</pre>", msg.Content))
	}

	// Fields
	if len(msg.Fields) > 0 {
		body.WriteString(`<div class="fields"><table>`)
		for k, v := range msg.Fields {
			body.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", k, v))
		}
		body.WriteString("</table></div>")
	}
	body.WriteString("</div>")

	// Footer
	body.WriteString(`<div class="footer">Sent by AtlHyper</div>`)
	body.WriteString("</div></body></html>")

	return body.String()
}

// 确保实现了接口
var _ Notifier = (*EmailNotifier)(nil)
