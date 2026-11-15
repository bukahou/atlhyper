package channel

import (
	"errors"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
)

func TestNewReceiverNotifier(t *testing.T) {
	completeEmail := &database.EmailConfig{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPUser: "ops@example.com",
		SMTPPassword: "placeholder-not-a-real-password", ToAddresses: []string{"a@example.com"}}
	noRecipients := *completeEmail
	noRecipients.ToAddresses = nil

	cases := []struct {
		name     string
		receiver *database.NotifyReceiver
		wantName string
		wantErr  error
	}{
		{"Slack", &database.NotifyReceiver{Type: "slack", Slack: &database.SlackConfig{WebhookURL: "https://hooks.example.com/services/x"}}, "slack", nil},
		{"Slack 没有 Webhook", &database.NotifyReceiver{Type: "slack", Slack: &database.SlackConfig{}}, "", ErrReceiverConfigIncomplete},
		{"Slack 配置解不开（nil）", &database.NotifyReceiver{Type: "slack"}, "", ErrReceiverConfigIncomplete},
		{"邮件", &database.NotifyReceiver{Type: "email", Email: completeEmail}, "email", nil},
		{"邮件没有收件人", &database.NotifyReceiver{Type: "email", Email: &noRecipients}, "", ErrReceiverConfigIncomplete},
		{"邮件配置解不开（nil）", &database.NotifyReceiver{Type: "email"}, "", ErrReceiverConfigIncomplete},
		{"不支持的类型", &database.NotifyReceiver{Type: "dingtalk"}, "", ErrUnsupportedReceiverType},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			notifier, err := NewReceiverNotifier(c.receiver)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("错误是 %v，期望 %v", err, c.wantErr)
			}
			if c.wantErr == nil && notifier.Name() != c.wantName {
				t.Errorf("发送器是 %q，期望 %q", notifier.Name(), c.wantName)
			}
		})
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

	email := &database.NotifyReceiver{Type: "email", Email: &database.EmailConfig{SMTPPassword: "placeholder-not-a-real-password"}}
	if got := ReceiverSecrets(email); len(got) != 1 || got[0] != "placeholder-not-a-real-password" {
		t.Errorf("邮件的凭据应只有密码，实际 %v", got)
	}
	if got := ReceiverSecrets(&database.NotifyReceiver{Type: "slack"}); len(got) != 0 {
		t.Errorf("没有配置时不应有凭据，实际 %v", got)
	}
}
