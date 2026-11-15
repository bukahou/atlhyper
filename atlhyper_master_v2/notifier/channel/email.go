// atlhyper_master_v2/notifier/channel/email.go
// Email 通知器（SMTP）
package channel

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const (
	// smtpDialTimeout 建立连接的超时
	smtpDialTimeout = 10 * time.Second
	// smtpTotalTimeout 一次发送（连接 + 握手 + 认证 + 投递）的整体超时
	smtpTotalTimeout = 30 * time.Second
	// smtpSubmissionPort 587 端口约定先明文连接、再用 STARTTLS 升级；其余端口开启 TLS 时直接走 TLS
	smtpSubmissionPort = 587
	// mailBodyLineLength base64 正文每行的字符数（RFC 2045 上限 76）
	mailBodyLineLength = 76
)

// EmailConfig Email 配置
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	UseTLS       bool
	FromAddress  string // 空 = 使用 SMTPUser
	ToAddresses  []string
}

// EmailNotifier Email 通知器
type EmailNotifier struct {
	config       EmailConfig
	dialTimeout  time.Duration
	totalTimeout time.Duration
}

// NewEmailNotifier 创建 Email 通知器
func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{config: config, dialTimeout: smtpDialTimeout, totalTimeout: smtpTotalTimeout}
}

// Name 返回通知器名称
func (e *EmailNotifier) Name() string {
	return "email"
}

// Send 发送邮件。连接与整体各有超时；调用方的 ctx 更早到期时以它为准
func (e *EmailNotifier) Send(ctx context.Context, msg *Message) error {
	deadline := time.Now().Add(e.totalTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	err := e.deliver(ctx, deadline, e.buildMailMessage(msg.Subject, msg.Body, time.Now()))
	if err != nil {
		// 认证失败等报错来自对端，理论上不带密码；这里再兜一层
		return errors.New(ScrubSecrets(err.Error(), e.config.SMTPPassword))
	}
	return nil
}

// deliver 完成一次 SMTP 对话
func (e *EmailNotifier) deliver(ctx context.Context, deadline time.Time, message string) error {
	address := net.JoinHostPort(e.config.SMTPHost, strconv.Itoa(e.config.SMTPPort))
	tlsConfig := &tls.Config{ServerName: e.config.SMTPHost, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: e.dialTimeout}
	implicitTLS := e.config.UseTLS && e.config.SMTPPort != smtpSubmissionPort

	var conn net.Conn
	var err error
	if implicitTLS {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", address)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	defer conn.Close()
	// 之后的每一步读写都受整体超时约束：对端不回话时不会一直挂着
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("设置 SMTP 超时失败: %w", err)
	}

	client, err := smtp.NewClient(conn, e.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("SMTP 握手失败: %w", err)
	}
	defer client.Close()

	if !implicitTLS {
		supportsStartTLS, _ := client.Extension("STARTTLS")
		switch {
		case supportsStartTLS:
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS 失败: %w", err)
			}
		case e.config.UseTLS:
			return errors.New("SMTP 服务器不支持 STARTTLS，而配置要求使用 TLS")
		}
	}

	// PlainAuth 自身拒绝在未加密的非本机连接上发送密码
	auth := smtp.PlainAuth("", e.config.SMTPUser, e.config.SMTPPassword, e.config.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}
	if err := client.Mail(e.fromAddress()); err != nil {
		return fmt.Errorf("SMTP 发件人被拒绝: %w", err)
	}
	for _, recipient := range e.config.ToAddresses {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP 收件人 %s 被拒绝: %w", recipient, err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP 发送正文失败: %w", err)
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("SMTP 写入正文失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP 提交正文失败: %w", err)
	}
	return client.Quit()
}

// fromAddress 发件人；没填时使用 SMTP 账号
func (e *EmailNotifier) fromAddress() string {
	if e.config.FromAddress != "" {
		return e.config.FromAddress
	}
	return e.config.SMTPUser
}

// buildMailMessage 构建邮件：主题按 RFC 2047 编码，正文按 base64 传输（UTF-8）
func (e *EmailNotifier) buildMailMessage(subject, body string, sentAt time.Time) string {
	recipients := make([]string, len(e.config.ToAddresses))
	for i, recipient := range e.config.ToAddresses {
		recipients[i] = singleLine(recipient)
	}

	var sb strings.Builder
	sb.WriteString("From: " + singleLine(e.fromAddress()) + "\r\n")
	sb.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	sb.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", singleLine(subject)) + "\r\n")
	sb.WriteString("Date: " + sentAt.Format(time.RFC1123Z) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: base64\r\n")
	sb.WriteString("\r\n")

	normalized := strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(normalized))
	for len(encoded) > mailBodyLineLength {
		sb.WriteString(encoded[:mailBodyLineLength] + "\r\n")
		encoded = encoded[mailBodyLineLength:]
	}
	sb.WriteString(encoded + "\r\n")
	return sb.String()
}

// singleLine 去掉回车换行：头部的值里出现换行就能伪造出新的邮件头
func singleLine(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}
