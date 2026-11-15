package channel

import (
	"context"
	"errors"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
)

func TestNewReceiverNotifier(t *testing.T) {
	smtpSettings := &database.NotifySMTP{Host: "smtp.example.com", Port: 587, User: "ops@example.com",
		Password: "placeholder-not-a-real-password", TLS: true}
	noPassword := *smtpSettings
	noPassword.Password = ""
	email := &database.EmailConfig{ToAddresses: []string{"a@example.com"}}

	cases := []struct {
		name     string
		receiver *database.NotifyReceiver
		smtp     *database.NotifySMTP
		wantName string
		wantErr  error
	}{
		{"Slack（不需要发件设置）", &database.NotifyReceiver{Type: "slack", Slack: &database.SlackConfig{WebhookURL: "https://hooks.example.com/services/x"}}, nil, "slack", nil},
		{"Slack 没有 Webhook", &database.NotifyReceiver{Type: "slack", Slack: &database.SlackConfig{}}, nil, "", ErrReceiverConfigIncomplete},
		{"Slack 配置解不开（nil）", &database.NotifyReceiver{Type: "slack"}, nil, "", ErrReceiverConfigIncomplete},
		{"邮件", &database.NotifyReceiver{Type: "email", Email: email}, smtpSettings, "email", nil},
		{"邮件没有收件人", &database.NotifyReceiver{Type: "email", Email: &database.EmailConfig{}}, smtpSettings, "", ErrReceiverConfigIncomplete},
		{"邮件配置解不开（nil）", &database.NotifyReceiver{Type: "email"}, smtpSettings, "", ErrReceiverConfigIncomplete},
		{"没有发件设置", &database.NotifyReceiver{Type: "email", Email: email}, nil, "", ErrSMTPNotConfigured},
		{"发件设置缺密码", &database.NotifyReceiver{Type: "email", Email: email}, &noPassword, "", ErrSMTPNotConfigured},
		{"不支持的类型", &database.NotifyReceiver{Type: "dingtalk"}, nil, "", ErrUnsupportedReceiverType},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			notifier, err := NewReceiverNotifier(c.receiver, c.smtp)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("错误是 %v，期望 %v", err, c.wantErr)
			}
			if c.wantErr == nil && notifier.Name() != c.wantName {
				t.Errorf("发送器是 %q，期望 %q", notifier.Name(), c.wantName)
			}
		})
	}
}

// 发件设置未配置也算「配置不完整」：测试发送与送达记录沿用同一类错误
func TestErrSMTPNotConfigured_IsConfigIncomplete(t *testing.T) {
	if !errors.Is(ErrSMTPNotConfigured, ErrReceiverConfigIncomplete) {
		t.Error("ErrSMTPNotConfigured 应能被识别为 ErrReceiverConfigIncomplete")
	}
}

type fakeSMTPReader struct {
	settings *database.NotifySMTP
	err      error
	calls    int
}

func (f *fakeSMTPReader) Get(context.Context) (*database.NotifySMTP, error) {
	f.calls++
	return f.settings, f.err
}

func TestReceiverSender_ReadsGlobalSMTPOnlyForEmail(t *testing.T) {
	reader := &fakeSMTPReader{}
	sender := NewReceiverSender(reader)
	msg := &Message{Subject: "s", Body: "b"}

	// 邮件 + 未配置：不发送，返回「未配置发件设置」
	err := sender.Send(context.Background(), &database.NotifyReceiver{Type: "email",
		Email: &database.EmailConfig{ToAddresses: []string{"a@example.com"}}}, msg)
	if !errors.Is(err, ErrSMTPNotConfigured) || reader.calls != 1 {
		t.Fatalf("应读一次发件设置并返回未配置，实际 err=%v calls=%d", err, reader.calls)
	}

	// 读发件设置出错：原样返回
	reader.err = errors.New("db down")
	if err := sender.Send(context.Background(), &database.NotifyReceiver{Type: "email",
		Email: &database.EmailConfig{ToAddresses: []string{"a@example.com"}}}, msg); err == nil || errors.Is(err, ErrSMTPNotConfigured) {
		t.Fatalf("读发件设置失败应返回该错误，实际 %v", err)
	}

	// Slack 不读发件设置
	reader.calls = 0
	_ = sender.Send(context.Background(), &database.NotifyReceiver{Type: "slack", Slack: &database.SlackConfig{}}, msg)
	if reader.calls != 0 {
		t.Errorf("Slack 接收人不应读发件设置，实际读了 %d 次", reader.calls)
	}
}

func TestReceiverSecrets(t *testing.T) {
	slack := &database.NotifyReceiver{Type: "slack", Slack: &database.SlackConfig{WebhookURL: "https://hooks.example.com/services/T000/B000/placeholder"}}
	secrets := ReceiverSecrets(slack)
	want := map[string]bool{"https://hooks.example.com/services/T000/B000/placeholder": true, "hooks.example.com": true, "/services/T000/B000/placeholder": true}
	for _, secret := range secrets {
		delete(want, secret)
	}
	if len(want) != 0 {
		t.Errorf("Slack 的凭据应包含完整地址、主机、路径，缺少 %v", want)
	}

	email := &database.NotifyReceiver{Type: "email", Email: &database.EmailConfig{ToAddresses: []string{"a@example.com"}}}
	if got := ReceiverSecrets(email); len(got) != 0 {
		t.Errorf("邮件接收人自身没有凭据（SMTP 密码在全局发件设置里），实际 %v", got)
	}
	if got := ReceiverSecrets(&database.NotifyReceiver{Type: "slack"}); len(got) != 0 {
		t.Errorf("没有配置时不应有凭据，实际 %v", got)
	}
}
