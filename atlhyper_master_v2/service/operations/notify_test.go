package operations

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/model"
)

// 占位凭据：不是任何真实系统的地址或密码
const (
	notifyTestWebhook  = "https://hooks.example.com/services/T000/B000/placeholder-not-real"
	notifyTestPassword = "placeholder-not-a-real-password"
)

var notifyTestNow = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

func newNotifyTestService(t *testing.T) (*NotifyService, *database.DB) {
	t.Helper()
	if err := repo.SetEncryptionSecret("test-secret-not-real"); err != nil {
		t.Fatal(err)
	}
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "t.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())

	svc := NewNotifyService(NotifyRepos{
		Policy: db.NotifyPolicy, Receivers: db.NotifyReceiver, Rules: db.NotifyRule, Silences: db.NotifySilence,
		SMTP: db.NotifySMTP,
	})
	svc.now = func() time.Time { return notifyTestNow }
	return svc, db
}

func notifyBool(v bool) *bool { return &v }
func notifyInt(v int) *int    { return &v }

func slackReceiverRequest(name string) *model.NotifyReceiverRequest {
	return &model.NotifyReceiverRequest{Name: name, Type: "slack", Slack: &model.NotifySlackInput{WebhookURL: notifyTestWebhook}}
}

func emailReceiverRequest(name string) *model.NotifyReceiverRequest {
	return &model.NotifyReceiverRequest{Name: name, Type: "email", Email: &model.NotifyEmailInput{
		ToAddresses: []string{"a@example.com"}}}
}

func mustCreateReceiver(t *testing.T, svc *NotifyService, req *model.NotifyReceiverRequest) int64 {
	t.Helper()
	id, err := svc.CreateNotifyReceiver(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func fullPolicyRequest(change func(*model.UpdateNotifyPolicyRequest)) *model.UpdateNotifyPolicyRequest {
	minSeverity := "low"
	req := &model.UpdateNotifyPolicyRequest{
		IncidentAlertsEnabled: notifyBool(true), MinSeverity: &minSeverity, NotifyResolved: notifyBool(true),
		WarningFlapThreshold: notifyInt(3), GroupWaitSeconds: notifyInt(30), GroupIntervalSeconds: notifyInt(300),
		RepeatIntervalSeconds: notifyInt(14400), K8sEventAlertsEnabled: notifyBool(false), DefaultReceiverIDs: &[]int64{},
	}
	if change != nil {
		change(req)
	}
	return req
}

// ==================== 策略 ====================

func TestUpdateNotifyPolicy(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	receiverID := mustCreateReceiver(t, svc, slackReceiverRequest("值班频道"))

	err := svc.UpdateNotifyPolicy(ctx, fullPolicyRequest(func(r *model.UpdateNotifyPolicyRequest) {
		high := "high"
		r.MinSeverity, r.NotifyResolved, r.DefaultReceiverIDs = &high, notifyBool(false), &[]int64{receiverID}
	}))
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := db.NotifyPolicy.Get(ctx)
	if stored.MinSeverity != "high" || stored.NotifyResolved || len(stored.DefaultReceiverIDs) != 1 || stored.DefaultReceiverIDs[0] != receiverID {
		t.Errorf("策略没有按请求保存: %+v", stored)
	}

	// 校验不过、或默认接收人不存在：拒绝，库里不变
	for name, bad := range map[string]*model.UpdateNotifyPolicyRequest{
		"取值越界":     fullPolicyRequest(func(r *model.UpdateNotifyPolicyRequest) { r.GroupWaitSeconds = notifyInt(9999) }),
		"默认接收人不存在": fullPolicyRequest(func(r *model.UpdateNotifyPolicyRequest) { r.DefaultReceiverIDs = &[]int64{receiverID, 999} }),
	} {
		if err := svc.UpdateNotifyPolicy(ctx, bad); !errors.Is(err, model.ErrInvalidNotifyInput) {
			t.Errorf("%s：应返回 ErrInvalidNotifyInput，实际 %v", name, err)
		}
	}
	if after, _ := db.NotifyPolicy.Get(ctx); after.MinSeverity != "high" || after.GroupWaitSeconds != 30 {
		t.Errorf("被拒绝的请求不应改动策略: %+v", after)
	}
}

// ==================== 接收人 ====================

func TestCreateNotifyReceiver(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()

	slackID := mustCreateReceiver(t, svc, slackReceiverRequest("  值班频道  "))
	stored, _ := db.NotifyReceiver.GetByID(ctx, slackID)
	if stored.Name != "值班频道" || !stored.Enabled || stored.Slack.WebhookURL != notifyTestWebhook {
		t.Errorf("新建的接收人不对（名称去空白、默认启用）: %+v", stored)
	}

	disabled := emailReceiverRequest("运维邮件")
	disabled.Enabled = notifyBool(false)
	emailID := mustCreateReceiver(t, svc, disabled)
	storedEmail, _ := db.NotifyReceiver.GetByID(ctx, emailID)
	if storedEmail.Enabled || storedEmail.Email == nil || storedEmail.Email.ToAddresses[0] != "a@example.com" {
		t.Errorf("新建的邮件接收人不对: %+v", storedEmail)
	}

	if _, err := svc.CreateNotifyReceiver(ctx, slackReceiverRequest("值班频道")); !errors.Is(err, database.ErrNotifyReceiverNameTaken) {
		t.Errorf("重名应返回 ErrNotifyReceiverNameTaken，实际 %v", err)
	}
	insecure := slackReceiverRequest("不安全")
	insecure.Slack.WebhookURL = "http://hooks.example.com/services/x"
	if _, err := svc.CreateNotifyReceiver(ctx, insecure); !errors.Is(err, model.ErrInvalidNotifyInput) {
		t.Errorf("非 https 的 Webhook 应被拒绝，实际 %v", err)
	}
}

// 修改时凭据留空 = 保持原值；其余字段以请求为准
func TestUpdateNotifyReceiver_BlankSecretKeepsStoredValue(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	slackID := mustCreateReceiver(t, svc, slackReceiverRequest("值班频道"))
	emailID := mustCreateReceiver(t, svc, emailReceiverRequest("运维邮件"))

	// 停用 Slack 接收人：凭据传空串（前端从来拿不到凭据，无从回传）
	err := svc.UpdateNotifyReceiver(ctx, slackID, &model.NotifyReceiverRequest{
		Name: "值班频道（已停用）", Type: "slack", Enabled: notifyBool(false), Slack: &model.NotifySlackInput{WebhookURL: ""}})
	if err != nil {
		t.Fatal(err)
	}
	slack, _ := db.NotifyReceiver.GetByID(ctx, slackID)
	if slack.Name != "值班频道（已停用）" || slack.Enabled || slack.Slack.WebhookURL != notifyTestWebhook {
		t.Errorf("凭据留空应保持原值: %+v", slack)
	}

	// 邮件：改收件人
	update := emailReceiverRequest("运维邮件")
	update.Email.ToAddresses = []string{"b@example.com", "c@example.com"}
	if err := svc.UpdateNotifyReceiver(ctx, emailID, update); err != nil {
		t.Fatal(err)
	}
	email, _ := db.NotifyReceiver.GetByID(ctx, emailID)
	if len(email.Email.ToAddresses) != 2 || !email.Enabled {
		t.Errorf("邮件接收人更新不对（收件人应替换、没传 enabled 应保持）: %+v", email)
	}

	// 不带 email 段：整段配置保持原值
	if err := svc.UpdateNotifyReceiver(ctx, emailID, &model.NotifyReceiverRequest{Name: "运维邮件", Type: "email", Enabled: notifyBool(false)}); err != nil {
		t.Fatal(err)
	}
	email, _ = db.NotifyReceiver.GetByID(ctx, emailID)
	if email.Enabled || len(email.Email.ToAddresses) != 2 {
		t.Errorf("不带配置段时配置应整段保持: %+v", email)
	}

	// 给了新凭据：替换
	rotated := slackReceiverRequest("值班频道")
	rotated.Slack.WebhookURL = notifyTestWebhook + "-rotated"
	if err := svc.UpdateNotifyReceiver(ctx, slackID, rotated); err != nil {
		t.Fatal(err)
	}
	slack, _ = db.NotifyReceiver.GetByID(ctx, slackID)
	if slack.Slack.WebhookURL != notifyTestWebhook+"-rotated" {
		t.Errorf("给了新凭据应替换: %+v", slack.Slack)
	}
}

func TestUpdateNotifyReceiver_Errors(t *testing.T) {
	svc, _ := newNotifyTestService(t)
	ctx := context.Background()
	slackID := mustCreateReceiver(t, svc, slackReceiverRequest("值班频道"))
	mustCreateReceiver(t, svc, emailReceiverRequest("运维邮件"))

	if err := svc.UpdateNotifyReceiver(ctx, 999, slackReceiverRequest("无")); !errors.Is(err, database.ErrNotifyReceiverNotFound) {
		t.Errorf("不存在的接收人应返回 ErrNotifyReceiverNotFound，实际 %v", err)
	}
	if err := svc.UpdateNotifyReceiver(ctx, slackID, slackReceiverRequest("运维邮件")); !errors.Is(err, database.ErrNotifyReceiverNameTaken) {
		t.Errorf("改成重名应返回 ErrNotifyReceiverNameTaken，实际 %v", err)
	}
	// 换类型：旧凭据对新类型没有意义，必须给出新类型的完整配置
	toEmail := &model.NotifyReceiverRequest{Name: "值班频道", Type: "email"}
	if err := svc.UpdateNotifyReceiver(ctx, slackID, toEmail); !errors.Is(err, model.ErrInvalidNotifyInput) {
		t.Errorf("换类型而不给新配置应被拒绝，实际 %v", err)
	}
	if err := svc.UpdateNotifyReceiver(ctx, slackID, emailReceiverRequest("值班频道")); err != nil {
		t.Errorf("换类型并给出完整配置应成功，实际 %v", err)
	}
}

// 仍被规则或默认策略引用的接收人不能删；错误里写明被谁引用
func TestDeleteNotifyReceiver_InUse(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	usedByRule := mustCreateReceiver(t, svc, slackReceiverRequest("值班频道"))
	usedByPolicy := mustCreateReceiver(t, svc, slackReceiverRequest("开发频道"))
	unused := mustCreateReceiver(t, svc, slackReceiverRequest("没人用"))

	ruleID, err := svc.CreateNotifyRule(ctx, &model.NotifyRuleRequest{Name: "生产高严重度", MinSeverity: "high", ReceiverIDs: []int64{usedByRule}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateNotifyPolicy(ctx, fullPolicyRequest(func(r *model.UpdateNotifyPolicyRequest) { r.DefaultReceiverIDs = &[]int64{usedByPolicy} })); err != nil {
		t.Fatal(err)
	}

	var inUse *model.NotifyReceiverInUseError
	if err := svc.DeleteNotifyReceiver(ctx, usedByRule); !errors.As(err, &inUse) || !strings.Contains(err.Error(), "生产高严重度") {
		t.Errorf("被规则引用时应拒绝并写明规则名，实际 %v", err)
	}
	if err := svc.DeleteNotifyReceiver(ctx, usedByPolicy); !errors.As(err, &inUse) || !strings.Contains(err.Error(), "默认策略") {
		t.Errorf("被默认策略引用时应拒绝并写明，实际 %v", err)
	}
	if err := svc.DeleteNotifyReceiver(ctx, unused); err != nil {
		t.Errorf("没被引用的应能删除，实际 %v", err)
	}
	if err := svc.DeleteNotifyReceiver(ctx, unused); !errors.Is(err, database.ErrNotifyReceiverNotFound) {
		t.Errorf("重复删除应返回 ErrNotifyReceiverNotFound，实际 %v", err)
	}

	// 解除引用后可以删
	if err := svc.DeleteNotifyRule(ctx, ruleID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteNotifyReceiver(ctx, usedByRule); err != nil {
		t.Errorf("规则删除后接收人应能删除，实际 %v", err)
	}
	if remaining, _ := db.NotifyReceiver.List(ctx); len(remaining) != 1 {
		t.Errorf("应只剩被默认策略引用的那一个，实际 %d 个", len(remaining))
	}
}

// ==================== 路由规则 ====================

func TestNotifyRuleLifecycle(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	receiverID := mustCreateReceiver(t, svc, slackReceiverRequest("值班频道"))

	firstID, err := svc.CreateNotifyRule(ctx, &model.NotifyRuleRequest{Name: "生产高严重度", ClusterIDs: []string{"prod"},
		Sources: []string{"incident"}, MinSeverity: "high", ReceiverIDs: []int64{receiverID}, RepeatIntervalSeconds: notifyInt(3600)})
	if err != nil {
		t.Fatal(err)
	}
	// 接收人为空是允许的：命中后不发给任何人。严重度没填 = low
	secondID, err := svc.CreateNotifyRule(ctx, &model.NotifyRuleRequest{Name: "测试集群不通知", ClusterIDs: []string{"test"}})
	if err != nil {
		t.Fatal(err)
	}
	second, _ := db.NotifyRule.GetByID(ctx, secondID)
	if second.Position != 2 || !second.Enabled || second.MinSeverity != "low" || len(second.ReceiverIDs) != 0 {
		t.Errorf("新规则应排在最后、默认启用、严重度默认 low: %+v", second)
	}

	if _, err := svc.CreateNotifyRule(ctx, &model.NotifyRuleRequest{Name: "引用不存在的接收人", ReceiverIDs: []int64{999}}); !errors.Is(err, model.ErrInvalidNotifyInput) {
		t.Errorf("接收人不存在应被拒绝，实际 %v", err)
	}

	// 修改：不改位置；没传 enabled 保持原值
	err = svc.UpdateNotifyRule(ctx, firstID, &model.NotifyRuleRequest{Name: "生产全部", ClusterIDs: []string{"prod"}, MinSeverity: "low", ReceiverIDs: []int64{receiverID}})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := db.NotifyRule.GetByID(ctx, firstID)
	if first.Name != "生产全部" || first.Position != 1 || !first.Enabled || first.RepeatIntervalSeconds != nil || len(first.Sources) != 0 {
		t.Errorf("修改后的规则不对: %+v", first)
	}
	if err := svc.UpdateNotifyRule(ctx, 999, &model.NotifyRuleRequest{Name: "无"}); !errors.Is(err, database.ErrNotifyRuleNotFound) {
		t.Errorf("不存在的规则应返回 ErrNotifyRuleNotFound，实际 %v", err)
	}

	if err := svc.ReorderNotifyRules(ctx, []int64{secondID, firstID}); err != nil {
		t.Fatal(err)
	}
	if list, _ := db.NotifyRule.List(ctx); list[0].ID != secondID || list[1].ID != firstID {
		t.Errorf("重排没有生效: %+v", list)
	}
	if err := svc.ReorderNotifyRules(ctx, []int64{firstID}); !errors.Is(err, database.ErrNotifyRuleOrderMismatch) {
		t.Errorf("不是全部规则的排列应被拒绝，实际 %v", err)
	}
}

// ==================== 静默 ====================

func TestNotifySilenceLifecycle(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	stamp := func(offset time.Duration) string { return notifyTestNow.Add(offset).Format(time.RFC3339) }

	id, err := svc.CreateNotifySilence(ctx, &model.CreateNotifySilenceRequest{ClusterID: "prod", Namespace: "shop",
		StartsAt: stamp(0), EndsAt: stamp(2 * time.Hour), Comment: "  数据库升级  "}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := db.NotifySilence.GetByID(ctx, id)
	if stored.ClusterID != "prod" || stored.Namespace != "shop" || stored.Comment != "数据库升级" || stored.CreatedBy != "admin" ||
		!stored.EndsAt.Equal(notifyTestNow.Add(2*time.Hour)) {
		t.Errorf("新建的静默不对: %+v", stored)
	}

	if _, err := svc.CreateNotifySilence(ctx, &model.CreateNotifySilenceRequest{StartsAt: stamp(0), EndsAt: stamp(-time.Hour), Comment: "x"}, "admin"); !errors.Is(err, model.ErrInvalidNotifyInput) {
		t.Errorf("结束时间已过应被拒绝，实际 %v", err)
	}

	// 提前结束：结束时间置为现在
	if err := svc.ExpireNotifySilence(ctx, id); err != nil {
		t.Fatal(err)
	}
	if expired, _ := db.NotifySilence.GetByID(ctx, id); !expired.EndsAt.Equal(notifyTestNow) {
		t.Errorf("提前结束后结束时间应为现在: %v", expired.EndsAt)
	}
	// 已经结束的再结束一次：不把结束时间往后改
	svc.now = func() time.Time { return notifyTestNow.Add(time.Hour) }
	if err := svc.ExpireNotifySilence(ctx, id); err != nil {
		t.Fatal(err)
	}
	if again, _ := db.NotifySilence.GetByID(ctx, id); !again.EndsAt.Equal(notifyTestNow) {
		t.Errorf("已结束的静默不应被改动: %v", again.EndsAt)
	}
	if err := svc.ExpireNotifySilence(ctx, 999); !errors.Is(err, database.ErrNotifySilenceNotFound) {
		t.Errorf("不存在的静默应返回 ErrNotifySilenceNotFound，实际 %v", err)
	}
}

// 还没开始的静默也能提前结束（建错了撤销）：之后它不再生效
func TestExpireNotifySilence_BeforeItStarts(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	id, err := svc.CreateNotifySilence(ctx, &model.CreateNotifySilenceRequest{
		StartsAt: notifyTestNow.Add(time.Hour).Format(time.RFC3339), EndsAt: notifyTestNow.Add(2 * time.Hour).Format(time.RFC3339), Comment: "建错了"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ExpireNotifySilence(ctx, id); err != nil {
		t.Fatal(err)
	}
	stored, _ := db.NotifySilence.GetByID(ctx, id)
	if stored.EndsAt.After(notifyTestNow) {
		t.Errorf("撤销后结束时间不应晚于现在: %+v", stored)
	}
}
