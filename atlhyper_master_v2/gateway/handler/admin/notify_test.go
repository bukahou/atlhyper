package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/service"
)

type stubNotifySvc struct {
	service.Service
	channels map[string]*database.NotifyChannel
	saved    *database.NotifyChannel
}

func (s *stubNotifySvc) GetNotifyChannelByType(_ context.Context, t string) (*database.NotifyChannel, error) {
	return s.channels[t], nil
}
func (s *stubNotifySvc) UpdateNotifyChannel(_ context.Context, ch *database.NotifyChannel) error {
	s.saved = ch
	return nil
}
func (s *stubNotifySvc) CreateNotifyChannel(_ context.Context, ch *database.NotifyChannel) error {
	s.saved = ch
	return nil
}

const (
	storedWebhook  = "https://hooks.slack.com/services/T000/B000/not-a-real-token"
	storedPassword = "not-a-real-password"
)

func storedChannels() map[string]*database.NotifyChannel {
	return map[string]*database.NotifyChannel{
		"slack": {ID: 1, Type: "slack", Enabled: true, Config: `{"webhookUrl":"` + storedWebhook + `"}`},
		"email": {ID: 2, Type: "email", Enabled: true, Config: `{"smtpHost":"smtp.example.com","smtpPort":587,"smtpUser":"bot@example.com",` +
			`"smtpPassword":"` + storedPassword + `","smtpTLS":true,"toAddresses":["ops@example.com"]}`},
	}
}

func configOf(t *testing.T, raw json.RawMessage) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// 查询不回显凭据：值置空，另标出哪些凭据已设置；配置完整性仍按真实配置判断
func TestToChannelResponse_HidesSecrets(t *testing.T) {
	ch := storedChannels()

	slack := toChannelResponse(ch["slack"])
	if strings.Contains(string(slack.Config), storedWebhook) || configOf(t, slack.Config)["webhookUrl"] != "" {
		t.Errorf("Slack webhook 不应回显：%s", slack.Config)
	}
	if len(slack.ConfiguredSecrets) != 1 || slack.ConfiguredSecrets[0] != "webhookUrl" || !slack.EffectiveEnabled {
		t.Errorf("slack: configuredSecrets=%v effective=%v", slack.ConfiguredSecrets, slack.EffectiveEnabled)
	}

	email := toChannelResponse(ch["email"])
	cfg := configOf(t, email.Config)
	if strings.Contains(string(email.Config), storedPassword) || cfg["smtpPassword"] != "" {
		t.Errorf("SMTP 密码不应回显：%s", email.Config)
	}
	if cfg["smtpHost"] != "smtp.example.com" || cfg["smtpUser"] != "bot@example.com" {
		t.Errorf("非凭据字段应原样返回：%v", cfg)
	}
	if len(email.ValidationErrors) != 0 || !email.EffectiveEnabled {
		t.Errorf("密码已设置时不应报未配置：%v", email.ValidationErrors)
	}
}

func putChannel(svc *stubNotifySvc, channelType, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v2/notify/channels/"+channelType, strings.NewReader(body))
	NewNotifyHandler(svc).ChannelHandler(rec, req)
	return rec
}

// 保存时凭据字段不传或为空 = 保留原值；传了新值才替换。其他字段仍以请求为准。
func TestUpdateChannel_KeepsSecretUnlessReplaced(t *testing.T) {
	cases := []struct {
		name, channelType, body string
		wantSecret              string
		secretField             string
	}{
		{"只改收件人，不带密码", "email",
			`{"config":{"smtpHost":"smtp.example.com","smtpPort":587,"smtpUser":"bot@example.com","toAddresses":["new@example.com"]}}`,
			storedPassword, "smtpPassword"},
		{"密码为空串", "email",
			`{"config":{"smtpHost":"smtp.example.com","smtpPort":587,"smtpUser":"bot@example.com","smtpPassword":"","toAddresses":["ops@example.com"]}}`,
			storedPassword, "smtpPassword"},
		{"填了新密码", "email",
			`{"config":{"smtpHost":"smtp.example.com","smtpPort":587,"smtpUser":"bot@example.com","smtpPassword":"new-pass","toAddresses":["ops@example.com"]}}`,
			"new-pass", "smtpPassword"},
		{"只切换启用", "slack", `{"enabled":false}`, storedWebhook, "webhookUrl"},
		{"webhook 留空", "slack", `{"config":{"webhookUrl":""}}`, storedWebhook, "webhookUrl"},
	}
	for _, c := range cases {
		svc := &stubNotifySvc{channels: storedChannels()}
		rec := putChannel(svc, c.channelType, c.body)
		if rec.Code != http.StatusOK || svc.saved == nil {
			t.Fatalf("%s: code=%d body=%s", c.name, rec.Code, rec.Body.String())
		}
		stored := configOf(t, json.RawMessage(svc.saved.Config))
		if stored[c.secretField] != c.wantSecret {
			t.Errorf("%s: 存下的 %s = %v, want %q", c.name, c.secretField, stored[c.secretField], c.wantSecret)
		}
		if strings.Contains(rec.Body.String(), c.wantSecret) {
			t.Errorf("%s: 响应不应回显凭据", c.name)
		}
	}

	svc := &stubNotifySvc{channels: storedChannels()}
	putChannel(svc, "email", `{"config":{"smtpHost":"smtp.example.com","smtpPort":587,"smtpUser":"bot@example.com","toAddresses":["new@example.com"]}}`)
	if to := configOf(t, json.RawMessage(svc.saved.Config))["toAddresses"]; to == nil || to.([]interface{})[0] != "new@example.com" {
		t.Errorf("非凭据字段以请求为准：toAddresses=%v", to)
	}
}

type fakeNotifyTester struct {
	err    error
	called string
}

func (f *fakeNotifyTester) Test(_ context.Context, channelType string) error {
	f.called = channelType
	return f.err
}

func postTest(h *NotifyHandler, channelType string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ChannelHandler(rec, httptest.NewRequest(http.MethodPost, "/api/v2/notify/channels/"+channelType+"/test", nil))
	return rec
}

// 测试发送走网关（与其他通知接口同一套权限与审计），按告警模块的错误给出可操作的提示
func TestTestChannel_SendsThroughNotifier(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"发送成功", nil, http.StatusOK},
		{"渠道不存在", notifier.ErrChannelNotFound, http.StatusNotFound},
		{"渠道未启用", notifier.ErrChannelDisabled, http.StatusConflict},
		{"配置不完整", notifier.ErrInvalidConfig, http.StatusBadRequest},
		{"不支持的渠道", notifier.ErrUnsupportedChannel, http.StatusBadRequest},
		{"上游发送失败", errors.New("smtp: 535 auth failed"), http.StatusBadGateway},
	}
	for _, c := range cases {
		tester := &fakeNotifyTester{err: c.err}
		h := NewNotifyHandler(&stubNotifySvc{})
		h.SetTester(tester)
		rec := postTest(h, "email")
		if rec.Code != c.want || tester.called != "email" {
			t.Errorf("%s: code=%d called=%q, want %d", c.name, rec.Code, tester.called, c.want)
		}
	}

	h := NewNotifyHandler(&stubNotifySvc{})
	if rec := postTest(h, "email"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("未注入测试能力时 = %d, want 503", rec.Code)
	}
	h.SetTester(&fakeNotifyTester{})
	if rec := postTest(h, "pigeon"); rec.Code != http.StatusBadRequest {
		t.Errorf("未知渠道类型 = %d, want 400", rec.Code)
	}
}
