package model

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func boolPtr(v bool) *bool    { return &v }
func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

func validPolicyRequest() *UpdateNotifyPolicyRequest {
	return &UpdateNotifyPolicyRequest{
		IncidentAlertsEnabled: boolPtr(true), MinSeverity: strPtr("low"), NotifyResolved: boolPtr(true),
		WarningFlapThreshold: intPtr(3), GroupWaitSeconds: intPtr(30), GroupIntervalSeconds: intPtr(300),
		RepeatIntervalSeconds: intPtr(14400), K8sEventAlertsEnabled: boolPtr(false), DefaultReceiverIDs: &[]int64{},
	}
}

func TestValidateNotifyPolicy(t *testing.T) {
	cases := []struct {
		name   string
		change func(*UpdateNotifyPolicyRequest)
		wantOK bool
	}{
		{"出厂值", func(*UpdateNotifyPolicyRequest) {}, true},
		{"边界：全部取下限", func(r *UpdateNotifyPolicyRequest) {
			r.WarningFlapThreshold, r.GroupWaitSeconds, r.GroupIntervalSeconds, r.RepeatIntervalSeconds = intPtr(0), intPtr(0), intPtr(30), intPtr(300)
		}, true},
		{"边界：全部取上限", func(r *UpdateNotifyPolicyRequest) {
			r.WarningFlapThreshold, r.GroupWaitSeconds, r.GroupIntervalSeconds, r.RepeatIntervalSeconds = intPtr(20), intPtr(600), intPtr(86400), intPtr(604800)
		}, true},
		{"严重度不在四档里", func(r *UpdateNotifyPolicyRequest) { r.MinSeverity = strPtr("warning") }, false},
		{"反复阈值超上限", func(r *UpdateNotifyPolicyRequest) { r.WarningFlapThreshold = intPtr(21) }, false},
		{"反复阈值为负", func(r *UpdateNotifyPolicyRequest) { r.WarningFlapThreshold = intPtr(-1) }, false},
		{"分组等待超上限", func(r *UpdateNotifyPolicyRequest) { r.GroupWaitSeconds = intPtr(601) }, false},
		{"分组间隔低于下限", func(r *UpdateNotifyPolicyRequest) { r.GroupIntervalSeconds = intPtr(29) }, false},
		{"重复间隔低于下限", func(r *UpdateNotifyPolicyRequest) { r.RepeatIntervalSeconds = intPtr(299) }, false},
		{"重复间隔超上限", func(r *UpdateNotifyPolicyRequest) { r.RepeatIntervalSeconds = intPtr(604801) }, false},
		{"缺字段：开关没传不能当成关", func(r *UpdateNotifyPolicyRequest) { r.IncidentAlertsEnabled = nil }, false},
		{"缺字段：默认接收人", func(r *UpdateNotifyPolicyRequest) { r.DefaultReceiverIDs = nil }, false},
		{"默认接收人有重复", func(r *UpdateNotifyPolicyRequest) { r.DefaultReceiverIDs = &[]int64{1, 1} }, false},
		{"默认接收人 id 非正", func(r *UpdateNotifyPolicyRequest) { r.DefaultReceiverIDs = &[]int64{0} }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := validPolicyRequest()
			c.change(req)
			err := ValidateNotifyPolicy(req)
			if (err == nil) != c.wantOK {
				t.Fatalf("err = %v，期望通过 = %v", err, c.wantOK)
			}
			if err != nil && !errors.Is(err, ErrInvalidNotifyInput) {
				t.Errorf("校验错误应是 ErrInvalidNotifyInput，实际 %v", err)
			}
		})
	}
}

func slackRequest(webhookURL string) *NotifyReceiverRequest {
	return &NotifyReceiverRequest{Name: "值班频道", Type: "slack", Slack: &NotifySlackInput{WebhookURL: webhookURL}}
}

func emailRequest(change func(*NotifyEmailInput)) *NotifyReceiverRequest {
	email := &NotifyEmailInput{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPUser: "ops@example.com",
		SMTPPassword: "placeholder-not-a-real-password", SMTPTLS: true, FromAddress: "ops@example.com",
		ToAddresses: []string{"a@example.com"}}
	if change != nil {
		change(email)
	}
	return &NotifyReceiverRequest{Name: "运维邮件", Type: "email", Email: email}
}

func TestValidateNotifyReceiver(t *testing.T) {
	cases := []struct {
		name          string
		req           *NotifyReceiverRequest
		requireSecret bool
		wantOK        bool
	}{
		{"Slack", slackRequest("https://hooks.example.com/services/T000/B000/placeholder"), true, true},
		{"Slack：只接受 https", slackRequest("http://hooks.example.com/services/x"), true, false},
		{"Slack：不是 URL", slackRequest("hooks.example.com/services/x"), true, false},
		{"Slack：没有主机", slackRequest("https:///services/x"), true, false},
		{"Slack：地址里带账号密码", slackRequest("https://user:pw@hooks.example.com/x"), true, false},
		{"Slack：新建时必须有地址", slackRequest(""), true, false},
		{"Slack：更新时留空 = 保持原值", slackRequest(""), false, true},
		{"Slack：更新时不带 slack 段", &NotifyReceiverRequest{Name: "值班频道", Type: "slack"}, false, true},
		{"名称为空", &NotifyReceiverRequest{Name: "  ", Type: "slack", Slack: &NotifySlackInput{WebhookURL: "https://hooks.example.com/x"}}, true, false},
		{"名称过长", &NotifyReceiverRequest{Name: strings.Repeat("长", 65), Type: "slack", Slack: &NotifySlackInput{WebhookURL: "https://hooks.example.com/x"}}, true, false},
		{"类型不支持", &NotifyReceiverRequest{Name: "钉钉", Type: "dingtalk"}, true, false},
		{"邮件", emailRequest(nil), true, true},
		{"邮件：发件人为空（用 SMTP 账号）", emailRequest(func(e *NotifyEmailInput) { e.FromAddress = "" }), true, true},
		{"邮件：新建时必须有密码", emailRequest(func(e *NotifyEmailInput) { e.SMTPPassword = "" }), true, false},
		{"邮件：更新时密码留空 = 保持原值", emailRequest(func(e *NotifyEmailInput) { e.SMTPPassword = "" }), false, true},
		{"邮件：没有主机", emailRequest(func(e *NotifyEmailInput) { e.SMTPHost = "" }), true, false},
		{"邮件：主机里有空白", emailRequest(func(e *NotifyEmailInput) { e.SMTPHost = "smtp.example.com\r\nX: y" }), true, false},
		{"邮件：端口越界", emailRequest(func(e *NotifyEmailInput) { e.SMTPPort = 70000 }), true, false},
		{"邮件：端口为 0", emailRequest(func(e *NotifyEmailInput) { e.SMTPPort = 0 }), true, false},
		{"邮件：没有账号", emailRequest(func(e *NotifyEmailInput) { e.SMTPUser = "" }), true, false},
		{"邮件：没有收件人", emailRequest(func(e *NotifyEmailInput) { e.ToAddresses = nil }), true, false},
		{"邮件：收件人不是邮箱", emailRequest(func(e *NotifyEmailInput) { e.ToAddresses = []string{"not-an-address"} }), true, false},
		{"邮件：收件人里夹带换行（伪造邮件头）", emailRequest(func(e *NotifyEmailInput) { e.ToAddresses = []string{"a@example.com\r\nBcc: b@example.com"} }), true, false},
		{"邮件：发件人不是邮箱", emailRequest(func(e *NotifyEmailInput) { e.FromAddress = "ops" }), true, false},
		{"邮件：新建时不带 email 段", &NotifyReceiverRequest{Name: "运维邮件", Type: "email"}, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateNotifyReceiver(c.req, c.requireSecret)
			if (err == nil) != c.wantOK {
				t.Fatalf("err = %v，期望通过 = %v", err, c.wantOK)
			}
			if err != nil && !errors.Is(err, ErrInvalidNotifyInput) {
				t.Errorf("校验错误应是 ErrInvalidNotifyInput，实际 %v", err)
			}
		})
	}
}

// 校验失败的文案会原样返回给调用方：里面不能带用户填的 Webhook 地址或密码
func TestValidateNotifyReceiver_ErrorsDoNotEchoSecrets(t *testing.T) {
	const badWebhook = "http://hooks.example.com/services/T000/B000/placeholder-not-real"
	err := ValidateNotifyReceiver(slackRequest(badWebhook), true)
	if err == nil || strings.Contains(err.Error(), "placeholder-not-real") || strings.Contains(err.Error(), "hooks.example.com") {
		t.Errorf("错误文案不应回显 Webhook 地址: %v", err)
	}
	withPassword := emailRequest(func(e *NotifyEmailInput) { e.SMTPPort = 0 })
	err = ValidateNotifyReceiver(withPassword, true)
	if err == nil || strings.Contains(err.Error(), "placeholder-not-a-real-password") {
		t.Errorf("错误文案不应回显密码: %v", err)
	}
}

func TestValidateNotifyRule(t *testing.T) {
	valid := func() *NotifyRuleRequest {
		return &NotifyRuleRequest{Name: "生产高严重度", ClusterIDs: []string{"prod"}, Sources: []string{"incident"},
			MinSeverity: "high", ReceiverIDs: []int64{1, 2}, RepeatIntervalSeconds: intPtr(3600)}
	}
	cases := []struct {
		name   string
		change func(*NotifyRuleRequest)
		wantOK bool
	}{
		{"完整", func(*NotifyRuleRequest) {}, true},
		{"条件全空（匹配一切）、不发给任何人", func(r *NotifyRuleRequest) { r.ClusterIDs, r.Sources, r.ReceiverIDs = nil, nil, nil }, true},
		{"三个时间参数都不覆盖", func(r *NotifyRuleRequest) { r.RepeatIntervalSeconds = nil }, true},
		{"时间参数取边界", func(r *NotifyRuleRequest) {
			r.GroupWaitSeconds, r.GroupIntervalSeconds, r.RepeatIntervalSeconds = intPtr(0), intPtr(86400), intPtr(300)
		}, true},
		{"名称为空", func(r *NotifyRuleRequest) { r.Name = "" }, false},
		{"严重度不合法", func(r *NotifyRuleRequest) { r.MinSeverity = "urgent" }, false},
		{"来源不合法", func(r *NotifyRuleRequest) { r.Sources = []string{"incident", "slo"} }, false},
		{"集群 ID 为空串", func(r *NotifyRuleRequest) { r.ClusterIDs = []string{""} }, false},
		{"接收人重复", func(r *NotifyRuleRequest) { r.ReceiverIDs = []int64{1, 1} }, false},
		{"分组等待越界", func(r *NotifyRuleRequest) { r.GroupWaitSeconds = intPtr(601) }, false},
		{"分组间隔越界", func(r *NotifyRuleRequest) { r.GroupIntervalSeconds = intPtr(10) }, false},
		{"重复间隔越界", func(r *NotifyRuleRequest) { r.RepeatIntervalSeconds = intPtr(60) }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := valid()
			c.change(req)
			err := ValidateNotifyRule(req)
			if (err == nil) != c.wantOK {
				t.Fatalf("err = %v，期望通过 = %v", err, c.wantOK)
			}
		})
	}
}

func TestParseNotifySilence(t *testing.T) {
	now := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	stamp := func(offset time.Duration) string { return now.Add(offset).Format(time.RFC3339) }
	valid := func() *CreateNotifySilenceRequest {
		return &CreateNotifySilenceRequest{ClusterID: "prod", Namespace: "shop", StartsAt: stamp(0), EndsAt: stamp(2 * time.Hour), Comment: "数据库升级"}
	}
	cases := []struct {
		name   string
		change func(*CreateNotifySilenceRequest)
		wantOK bool
	}{
		{"两小时的静默", func(*CreateNotifySilenceRequest) {}, true},
		{"未来才开始", func(r *CreateNotifySilenceRequest) { r.StartsAt, r.EndsAt = stamp(time.Hour), stamp(2*time.Hour) }, true},
		{"开始时间留空 = 现在", func(r *CreateNotifySilenceRequest) { r.StartsAt = "" }, true},
		{"恰好 30 天", func(r *CreateNotifySilenceRequest) { r.EndsAt = stamp(30 * 24 * time.Hour) }, true},
		{"结束不晚于开始", func(r *CreateNotifySilenceRequest) { r.EndsAt = stamp(0) }, false},
		{"结束时间已过", func(r *CreateNotifySilenceRequest) { r.StartsAt, r.EndsAt = stamp(-2*time.Hour), stamp(-time.Hour) }, false},
		{"超过 30 天", func(r *CreateNotifySilenceRequest) { r.EndsAt = stamp(30*24*time.Hour + time.Second) }, false},
		{"没有说明", func(r *CreateNotifySilenceRequest) { r.Comment = "  " }, false},
		{"说明过长", func(r *CreateNotifySilenceRequest) { r.Comment = strings.Repeat("长", 257) }, false},
		{"时间格式不对", func(r *CreateNotifySilenceRequest) { r.EndsAt = "明天" }, false},
		{"没有结束时间", func(r *CreateNotifySilenceRequest) { r.EndsAt = "" }, false},
		{"条件里有控制字符", func(r *CreateNotifySilenceRequest) { r.Namespace = "shop\n" }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := valid()
			c.change(req)
			startsAt, endsAt, err := ParseNotifySilence(req, now)
			if (err == nil) != c.wantOK {
				t.Fatalf("err = %v，期望通过 = %v", err, c.wantOK)
			}
			if err == nil && (!endsAt.After(startsAt) || startsAt.IsZero()) {
				t.Errorf("解析出的时间不对: %v – %v", startsAt, endsAt)
			}
		})
	}
}

func TestValidateNotifyRuleMatch(t *testing.T) {
	if err := ValidateNotifyRuleMatch(&NotifyRuleMatchRequest{ClusterID: "prod", Source: "incident", Severity: "high"}); err != nil {
		t.Errorf("合法请求不应报错: %v", err)
	}
	for _, bad := range []*NotifyRuleMatchRequest{
		{ClusterID: "prod", Source: "slo", Severity: "high"},
		{ClusterID: "prod", Source: "incident", Severity: "urgent"},
		{ClusterID: "prod", Source: "", Severity: "high"},
	} {
		if err := ValidateNotifyRuleMatch(bad); !errors.Is(err, ErrInvalidNotifyInput) {
			t.Errorf("%+v 应报校验错误，实际 %v", bad, err)
		}
	}
}
