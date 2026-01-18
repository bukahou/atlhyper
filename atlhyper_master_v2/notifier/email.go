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
	body.WriteString("<html><body>")
	body.WriteString(fmt.Sprintf("<h2>%s</h2>", msg.Title))
	body.WriteString(fmt.Sprintf("<p>%s</p>", msg.Content))

	if len(msg.Fields) > 0 {
		body.WriteString("<table border='1' cellpadding='5'>")
		for k, v := range msg.Fields {
			body.WriteString(fmt.Sprintf("<tr><td><strong>%s</strong></td><td>%s</td></tr>", k, v))
		}
		body.WriteString("</table>")
	}

	body.WriteString("<hr><p><small>Sent by AtlHyper</small></p>")
	body.WriteString("</body></html>")
	return body.String()
}

// 确保实现了接口
var _ Notifier = (*EmailNotifier)(nil)
