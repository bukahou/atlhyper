package convert

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

var notifyAt = time.Date(2026, 11, 15, 10, 0, 0, 0, time.FixedZone("JST", 9*3600))

func TestNotifyPolicy(t *testing.T) {
	src := database.DefaultNotifyPolicy()
	src.DefaultReceiverIDs = nil
	got := NotifyPolicy(src)
	if !got.IncidentAlertsEnabled || got.MinSeverity != "low" || got.GroupWaitSeconds != 30 || got.RepeatIntervalSeconds != 14400 {
		t.Errorf("策略转换不对: %+v", got)
	}
	if got.DefaultReceiverIDs == nil {
		t.Error("defaultReceiverIds 应为空数组而不是 null")
	}
}

func TestNotifyAlertItem(t *testing.T) {
	ended, notified := notifyAt.Add(time.Hour), notifyAt.Add(time.Minute)
	src := &database.NotifyAlert{
		Fingerprint: "incident:inc-123", Source: "incident", ClusterID: "prod", Severity: "high", Status: "resolved", Kind: "resolved",
		Title: "标题", Summary: "摘要",
		Labels:   map[string]string{"subject": "shop/deployment/api"},
		Details:  map[string]string{"incidentId": "inc-123"},
		StartsAt: notifyAt, EndsAt: &ended, LastNotifiedAt: &notified, NotifyCount: 2, SuppressedBy: "cluster-offline", SilencedBy: 7,
	}
	got := NotifyAlertItem(src)
	if got.Subject != "shop/deployment/api" || got.IncidentID != "inc-123" || got.NotifyCount != 2 ||
		got.SuppressedBy != "cluster-offline" || got.SilencedBy != 7 {
		t.Errorf("告警转换不对: %+v", got)
	}
	if got.StartsAt != "2026-11-15T01:00:00Z" || got.EndsAt != "2026-11-15T02:00:00Z" || got.LastNotifiedAt != "2026-11-15T01:01:00Z" {
		t.Errorf("时间应统一为 UTC 的 RFC 3339: %+v", got)
	}

	// 持续中、还没通知过：两个时间是空串，不是 null
	firing := NotifyAlertItem(&database.NotifyAlert{Fingerprint: "heartbeat:prod", StartsAt: notifyAt})
	if firing.EndsAt != "" || firing.LastNotifiedAt != "" || firing.Subject != "" || firing.IncidentID != "" {
		t.Errorf("没有的值应为空串: %+v", firing)
	}
	encoded, _ := json.Marshal(firing)
	if strings.Contains(string(encoded), "null") {
		t.Errorf("响应里不应出现 null: %s", encoded)
	}
}

func TestNotifyDeliveryItem_NoReceiver(t *testing.T) {
	got := NotifyDeliveryItem(&database.NotifyDelivery{ID: 1, BatchID: "b-1", Fingerprint: "incident:inc-1", AlertTitle: "标题",
		Kind: "firing", Status: database.NotifyDeliveryNoReceiver, CreatedAt: notifyAt, UpdatedAt: notifyAt})
	if got.ReceiverID != 0 || got.ReceiverName != "" || got.ReceiverType != "" || got.Status != "no-receiver" ||
		got.CreatedAt != "2026-11-15T01:00:00Z" {
		t.Errorf("没有接收人的送达记录不对: %+v", got)
	}
}

// 凭据永不出库：响应里只有「是否已配置」
func TestNotifyReceiverItem_NeverExposesSecrets(t *testing.T) {
	const webhook = "https://hooks.example.com/services/T000/B000/placeholder-not-real"

	slack := NotifyReceiverItem(&database.NotifyReceiver{ID: 1, Name: "值班频道", Type: "slack", Enabled: true,
		Slack: &database.SlackConfig{WebhookURL: webhook}, CreatedAt: notifyAt, UpdatedAt: notifyAt})
	email := NotifyReceiverItem(&database.NotifyReceiver{ID: 2, Name: "运维邮件", Type: "email", Enabled: true,
		Email: &database.EmailConfig{ToAddresses: []string{"a@example.com"}}})

	if slack.Slack == nil || !slack.Slack.WebhookConfigured || slack.Email != nil {
		t.Errorf("Slack 接收人转换不对: %+v", slack)
	}
	if email.Email == nil || len(email.Email.ToAddresses) != 1 || email.Email.ToAddresses[0] != "a@example.com" || email.Slack != nil {
		t.Errorf("邮件接收人转换不对: %+v", email)
	}
	for _, item := range []any{slack, email} {
		encoded, _ := json.Marshal(item)
		for _, secret := range []string{webhook, "hooks.example.com"} {
			if strings.Contains(string(encoded), secret) {
				t.Errorf("响应里出现了凭据 %q: %s", secret, encoded)
			}
		}
	}
}

// 配置解不开（Slack / Email 都是 nil）或凭据为空：如实标成未配置
func TestNotifyReceiverItem_UnconfiguredSecrets(t *testing.T) {
	unreadable := NotifyReceiverItem(&database.NotifyReceiver{ID: 1, Name: "值班频道", Type: "slack"})
	if unreadable.Slack == nil || unreadable.Slack.WebhookConfigured {
		t.Errorf("配置解不开的 Slack 接收人应标成未配置: %+v", unreadable.Slack)
	}
	unreadableEmail := NotifyReceiverItem(&database.NotifyReceiver{ID: 2, Name: "运维邮件", Type: "email"})
	if unreadableEmail.Email == nil || unreadableEmail.Email.ToAddresses == nil || len(unreadableEmail.Email.ToAddresses) != 0 {
		t.Errorf("配置解不开的邮件接收人收件人应为空数组: %+v", unreadableEmail.Email)
	}
}

// 发件设置：密码永不出库，只有「是否已配置」；从没保存过时是空设置、updatedAt 为 null
func TestNotifySMTPView(t *testing.T) {
	const password = "placeholder-not-a-real-password"
	view := NotifySMTPView(&database.NotifySMTP{Host: "smtp.example.com", Port: 587, User: "ops@example.com",
		Password: password, TLS: true, FromAddress: "alerts@example.com", UpdatedAt: notifyAt})
	if !view.Configured || !view.PasswordConfigured || view.Host != "smtp.example.com" || view.Port != 587 ||
		view.User != "ops@example.com" || !view.TLS || view.FromAddress != "alerts@example.com" ||
		view.UpdatedAt == nil || *view.UpdatedAt == "" {
		t.Errorf("发件设置转换不对: %+v", view)
	}
	encoded, _ := json.Marshal(view)
	if strings.Contains(string(encoded), password) {
		t.Errorf("响应里出现了密码: %s", encoded)
	}

	noPassword := NotifySMTPView(&database.NotifySMTP{Host: "smtp.example.com", Port: 587, User: "ops@example.com", UpdatedAt: notifyAt})
	if noPassword.Configured || noPassword.PasswordConfigured {
		t.Errorf("没有密码时应标成未配置: %+v", noPassword)
	}

	empty := NotifySMTPView(nil)
	encoded, _ = json.Marshal(empty)
	if empty.Configured || !strings.Contains(string(encoded), `"updatedAt":null`) {
		t.Errorf("从没保存过应是空设置、updatedAt 为 null: %s", encoded)
	}
}

func TestNotifyRuleItem(t *testing.T) {
	repeat := 3600
	got := NotifyRuleItem(&database.NotifyRule{ID: 1, Name: "生产高严重度", Position: 1, Enabled: true,
		MinSeverity: "high", ReceiverIDs: []int64{1, 2}, RepeatIntervalSeconds: &repeat, CreatedAt: notifyAt})
	if got.ClusterIDs == nil || got.Sources == nil || len(got.ReceiverIDs) != 2 || got.GroupWaitSeconds != nil ||
		got.RepeatIntervalSeconds == nil || *got.RepeatIntervalSeconds != 3600 {
		t.Errorf("规则转换不对: %+v", got)
	}
	encoded, _ := json.Marshal(got)
	if !strings.Contains(string(encoded), `"clusterIds":[]`) || !strings.Contains(string(encoded), `"groupWaitSeconds":null`) {
		t.Errorf("空条件应是空数组，没覆盖的时间参数应是 null: %s", encoded)
	}
}

func TestNotifySilenceItem_State(t *testing.T) {
	src := &database.NotifySilence{ID: 1, ClusterID: "prod", Namespace: "shop", StartsAt: notifyAt, EndsAt: notifyAt.Add(2 * time.Hour),
		Comment: "数据库升级", CreatedBy: "admin", CreatedAt: notifyAt}
	cases := []struct {
		now  time.Time
		want string
	}{
		{notifyAt.Add(-time.Minute), "pending"},
		{notifyAt.Add(time.Hour), "active"},
		{notifyAt.Add(3 * time.Hour), "expired"},
	}
	for _, c := range cases {
		got := NotifySilenceItem(src, c.now)
		if got.State != c.want || got.Comment != "数据库升级" || got.CreatedBy != "admin" || got.StartsAt != "2026-11-15T01:00:00Z" {
			t.Errorf("静默转换不对（期望状态 %s）: %+v", c.want, got)
		}
	}
}

func TestNotifyItems_EmptyIsNotNil(t *testing.T) {
	if NotifyAlertItems(nil) == nil || NotifyDeliveryItems(nil) == nil || NotifyReceiverItems(nil) == nil ||
		NotifyRuleItems(nil) == nil || NotifySilenceItems(nil, notifyAt) == nil {
		t.Error("批量转换对 nil 输入应返回空切片")
	}
}
