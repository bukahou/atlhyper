package repo_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
)

// 占位凭据：不是任何真实系统的地址或密码
const (
	testWebhookURL   = "https://hooks.example.com/services/T000/B000/placeholder-not-real"
	testSMTPPassword = "placeholder-not-a-real-password"
)

var notifyT0 = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

func openNotifyTestDB(t *testing.T) *database.DB {
	t.Helper()
	if err := repo.SetEncryptionSecret("test-secret-not-real"); err != nil {
		t.Fatal(err)
	}
	return openTestDB(t)
}

func timePtr(at time.Time) *time.Time { return &at }

// ==================== 策略 ====================

func TestNotifyPolicy_DefaultsWhenRowMissing(t *testing.T) {
	db := openNotifyTestDB(t)
	policy, err := db.NotifyPolicy.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := database.DefaultNotifyPolicy()
	if policy.MinSeverity != want.MinSeverity || policy.GroupWaitSeconds != 30 || policy.RepeatIntervalSeconds != 14400 ||
		!policy.IncidentAlertsEnabled || policy.K8sEventAlertsEnabled || policy.WarningFlapThreshold != 3 ||
		policy.DefaultReceiverIDs == nil || len(policy.DefaultReceiverIDs) != 0 {
		t.Errorf("没有策略行时应返回出厂值，实际 %+v", policy)
	}
}

func TestNotifyPolicy_SaveThenGet(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	policy := database.DefaultNotifyPolicy()
	policy.MinSeverity = "high"
	policy.NotifyResolved = false
	policy.WarningFlapThreshold = 0
	policy.DefaultReceiverIDs = []int64{2, 5}
	if err := db.NotifyPolicy.Save(ctx, policy); err != nil {
		t.Fatal(err)
	}
	policy.GroupWaitSeconds = 10 // 再存一次：仍是同一行
	if err := db.NotifyPolicy.Save(ctx, policy); err != nil {
		t.Fatal(err)
	}

	got, err := db.NotifyPolicy.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.MinSeverity != "high" || got.NotifyResolved || got.WarningFlapThreshold != 0 || got.GroupWaitSeconds != 10 ||
		len(got.DefaultReceiverIDs) != 2 || got.DefaultReceiverIDs[1] != 5 || got.UpdatedAt.IsZero() {
		t.Errorf("取回的策略不对: %+v", got)
	}
	var rows int
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM notify_policy").Scan(&rows); err != nil || rows != 1 {
		t.Errorf("策略表应恒为一行，实际 %d 行 (%v)", rows, err)
	}
}

// ==================== 告警 ====================

func firingAlert(fingerprint, clusterID string) *database.NotifyAlert {
	return &database.NotifyAlert{
		Fingerprint: fingerprint, Source: "incident", ClusterID: clusterID,
		Severity: "high", Status: "firing", Kind: "firing",
		Title: "标题", Summary: "摘要",
		Labels:   map[string]string{"namespace": "shop", "subject": "shop/deployment/api"},
		Details:  map[string]string{"incidentId": "inc-1"},
		StartsAt: notifyT0, UpdatedAt: notifyT0,
	}
}

func TestNotifyAlert_UpsertGetAndListFiring(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	if err := db.NotifyAlert.Upsert(ctx, firingAlert("incident:inc-1", "prod")); err != nil {
		t.Fatal(err)
	}

	// 同一指纹再写：更新而不是新增
	updated := firingAlert("incident:inc-1", "prod")
	updated.Severity, updated.Kind = "critical", "escalated"
	updated.LastNotifiedAt = timePtr(notifyT0.Add(time.Minute))
	updated.LastNotifiedKind, updated.LastNotifiedSeverity, updated.NotifyCount = "firing", "high", 1
	updated.SuppressedBy, updated.SilencedBy = "cluster-offline", 7
	if err := db.NotifyAlert.Upsert(ctx, updated); err != nil {
		t.Fatal(err)
	}
	resolved := firingAlert("incident:inc-2", "prod")
	resolved.Status, resolved.EndsAt = "resolved", timePtr(notifyT0.Add(time.Hour))
	if err := db.NotifyAlert.Upsert(ctx, resolved); err != nil {
		t.Fatal(err)
	}

	got, err := db.NotifyAlert.Get(ctx, "incident:inc-1")
	if err != nil || got == nil {
		t.Fatalf("取告警失败: %v %v", got, err)
	}
	if got.Severity != "critical" || got.Kind != "escalated" || got.NotifyCount != 1 || got.LastNotifiedKind != "firing" ||
		got.LastNotifiedSeverity != "high" || got.SuppressedBy != "cluster-offline" || got.SilencedBy != 7 ||
		got.LastNotifiedAt == nil || !got.LastNotifiedAt.Equal(notifyT0.Add(time.Minute)) ||
		got.Labels["subject"] != "shop/deployment/api" || got.Details["incidentId"] != "inc-1" ||
		!got.StartsAt.Equal(notifyT0) || got.EndsAt != nil {
		t.Errorf("取回的告警不对: %+v", got)
	}

	firing, err := db.NotifyAlert.ListFiring(ctx)
	if err != nil || len(firing) != 1 || firing[0].Fingerprint != "incident:inc-1" {
		t.Errorf("持续中的告警应只有 inc-1，实际 %v (%v)", firing, err)
	}

	missing, err := db.NotifyAlert.Get(ctx, "incident:none")
	if err != nil || missing != nil {
		t.Errorf("不存在的告警应返回 (nil, nil)，实际 (%v, %v)", missing, err)
	}
}

func TestNotifyAlert_ListFiltersCountAndCleanup(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	for i, clusterID := range []string{"prod", "prod", "test"} {
		a := firingAlert("incident:inc-"+string(rune('a'+i)), clusterID)
		a.StartsAt = notifyT0.Add(time.Duration(i) * time.Minute)
		if i == 1 {
			a.Status, a.EndsAt = "resolved", timePtr(notifyT0.Add(2*time.Hour))
		}
		if err := db.NotifyAlert.Upsert(ctx, a); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		opts database.NotifyAlertQueryOpts
		want int
	}{
		{database.NotifyAlertQueryOpts{}, 3},
		{database.NotifyAlertQueryOpts{Status: "firing"}, 2},
		{database.NotifyAlertQueryOpts{ClusterID: "prod"}, 2},
		{database.NotifyAlertQueryOpts{Status: "resolved", ClusterID: "prod"}, 1},
		{database.NotifyAlertQueryOpts{Status: "resolved", ClusterID: "test"}, 0},
	}
	for _, c := range cases {
		total, err := db.NotifyAlert.Count(ctx, c.opts)
		if err != nil || total != c.want {
			t.Errorf("Count(%+v) = %d (%v)，期望 %d", c.opts, total, err, c.want)
		}
		list, err := db.NotifyAlert.List(ctx, c.opts)
		if err != nil || len(list) != c.want {
			t.Errorf("List(%+v) 返回 %d 条 (%v)，期望 %d", c.opts, len(list), err, c.want)
		}
	}

	page, err := db.NotifyAlert.List(ctx, database.NotifyAlertQueryOpts{Limit: 1, Offset: 1})
	if err != nil || len(page) != 1 || page[0].Fingerprint != "incident:inc-b" {
		t.Errorf("分页应按开始时间新的在前取第二条，实际 %v (%v)", page, err)
	}

	// 只清理已结束且结束得足够久的；持续中的不动
	deleted, err := db.NotifyAlert.DeleteResolvedBefore(ctx, notifyT0.Add(3*time.Hour))
	if err != nil || deleted != 1 {
		t.Errorf("应清理 1 条已结束的告警，实际 %d (%v)", deleted, err)
	}
	if total, _ := db.NotifyAlert.Count(ctx, database.NotifyAlertQueryOpts{}); total != 2 {
		t.Errorf("清理后应剩 2 条，实际 %d", total)
	}
}

// ==================== 送达记录 ====================

func TestNotifyDelivery_LifecycleAndQueries(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	newDelivery := func(batchID, fingerprint, status string, createdAt time.Time) *database.NotifyDelivery {
		return &database.NotifyDelivery{
			BatchID: batchID, Fingerprint: fingerprint, AlertTitle: "标题",
			ReceiverID: 1, ReceiverName: "值班频道", ReceiverType: "slack",
			Kind: "firing", Status: status, CreatedAt: createdAt, UpdatedAt: createdAt,
		}
	}
	first := newDelivery("b-1", "incident:inc-1", database.NotifyDeliveryPending, notifyT0)
	second := newDelivery("b-1", "incident:inc-2", database.NotifyDeliveryPending, notifyT0)
	other := newDelivery("b-2", "incident:inc-1", database.NotifyDeliveryPending, notifyT0.Add(time.Hour))
	for _, d := range []*database.NotifyDelivery{first, second, other} {
		if err := db.NotifyDelivery.Create(ctx, d); err != nil {
			t.Fatal(err)
		}
		if d.ID == 0 {
			t.Fatal("写入后应回填 ID")
		}
	}

	// 一次失败：整个批次一起更新
	retryAt := notifyT0.Add(30 * time.Second)
	err := db.NotifyDelivery.UpdateBatch(ctx, "b-1", database.NotifyDeliveryResult{
		Status: database.NotifyDeliveryFailed, Attempts: 1, LastError: "对端返回 500",
		NextAttemptAt: &retryAt, UpdatedAt: notifyT0.Add(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.NotifyDelivery.UpdateBatch(ctx, "b-2", database.NotifyDeliveryResult{
		Status: database.NotifyDeliverySuccess, Attempts: 1, UpdatedAt: notifyT0.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	unfinished, err := db.NotifyDelivery.ListUnfinished(ctx)
	if err != nil || len(unfinished) != 2 {
		t.Fatalf("未完成的记录应有 2 条，实际 %d (%v)", len(unfinished), err)
	}
	for _, d := range unfinished {
		if d.BatchID != "b-1" || d.Status != database.NotifyDeliveryFailed || d.Attempts != 1 ||
			d.LastError != "对端返回 500" || d.NextAttemptAt == nil || !d.NextAttemptAt.Equal(retryAt) {
			t.Errorf("未完成的记录不对: %+v", d)
		}
	}

	byFingerprint, _ := db.NotifyDelivery.List(ctx, database.NotifyDeliveryQueryOpts{Fingerprint: "incident:inc-1"})
	if len(byFingerprint) != 2 || byFingerprint[0].BatchID != "b-2" {
		t.Errorf("按指纹应有 2 条且新的在前，实际 %+v", byFingerprint)
	}
	if total, _ := db.NotifyDelivery.Count(ctx, database.NotifyDeliveryQueryOpts{Status: database.NotifyDeliverySuccess}); total != 1 {
		t.Errorf("成功的记录应有 1 条，实际 %d", total)
	}

	deleted, err := db.NotifyDelivery.DeleteBefore(ctx, notifyT0.Add(time.Minute))
	if err != nil || deleted != 2 {
		t.Errorf("应清理 2 条旧记录，实际 %d (%v)", deleted, err)
	}
}

// ==================== 接收人 ====================

func slackReceiver(name string) *database.NotifyReceiver {
	return &database.NotifyReceiver{
		Name: name, Type: database.NotifyReceiverSlack, Enabled: true,
		Slack: &database.SlackConfig{WebhookURL: testWebhookURL},
	}
}

func emailReceiver(name string) *database.NotifyReceiver {
	return &database.NotifyReceiver{
		Name: name, Type: database.NotifyReceiverEmail, Enabled: true,
		Email: &database.EmailConfig{ToAddresses: []string{"a@example.com"}},
	}
}

// 凭据不得以明文落库：直接读 config 列，里面不能有 Webhook 地址，也不能有收件人这类明文字段
func TestNotifyReceiver_ConfigEncryptedAtRest(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	for _, r := range []*database.NotifyReceiver{slackReceiver("值班频道"), emailReceiver("运维邮件")} {
		if err := db.NotifyReceiver.Create(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := db.Conn.Query("SELECT config FROM notify_receivers")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var stored string
		if err := rows.Scan(&stored); err != nil {
			t.Fatal(err)
		}
		seen++
		for _, secret := range []string{testWebhookURL, "hooks.example.com", "a@example.com", "webhookUrl"} {
			if strings.Contains(stored, secret) {
				t.Errorf("config 列里出现了明文 %q", secret)
			}
		}
		if !strings.HasPrefix(stored, "enc:") {
			t.Errorf("config 列应是密文，实际 %q", stored)
		}
	}
	if seen != 2 {
		t.Fatalf("应读到 2 行，实际 %d", seen)
	}

	// 经仓库读出时已解密
	list, err := db.NotifyReceiver.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("列表应有 2 个接收人，实际 %d (%v)", len(list), err)
	}
	if list[0].Slack == nil || list[0].Slack.WebhookURL != testWebhookURL || list[0].Email != nil {
		t.Errorf("Slack 接收人解密后不对: %+v", list[0])
	}
	if list[1].Email == nil || len(list[1].Email.ToAddresses) != 1 || list[1].Email.ToAddresses[0] != "a@example.com" {
		t.Errorf("邮件接收人解密后不对: %+v", list[1])
	}
}

func TestNotifyReceiver_NameUniqueUpdateDelete(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	first, second := slackReceiver("值班频道"), slackReceiver("开发频道")
	if err := db.NotifyReceiver.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := db.NotifyReceiver.Create(ctx, second); err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 || first.CreatedAt.IsZero() {
		t.Errorf("创建后应回填 ID 与时间: %+v", first)
	}

	if err := db.NotifyReceiver.Create(ctx, slackReceiver("值班频道")); !errors.Is(err, database.ErrNotifyReceiverNameTaken) {
		t.Errorf("重名创建应返回 ErrNotifyReceiverNameTaken，实际 %v", err)
	}
	second.Name = "值班频道"
	if err := db.NotifyReceiver.Update(ctx, second); !errors.Is(err, database.ErrNotifyReceiverNameTaken) {
		t.Errorf("改成重名应返回 ErrNotifyReceiverNameTaken，实际 %v", err)
	}

	first.Enabled = false
	first.Slack.WebhookURL = testWebhookURL + "-rotated"
	if err := db.NotifyReceiver.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	got, err := db.NotifyReceiver.GetByID(ctx, first.ID)
	if err != nil || got == nil || got.Enabled || got.Slack.WebhookURL != testWebhookURL+"-rotated" {
		t.Errorf("更新后取回的不对: %+v (%v)", got, err)
	}

	if err := db.NotifyReceiver.Update(ctx, &database.NotifyReceiver{ID: 999, Name: "无", Type: "slack", Slack: &database.SlackConfig{}}); !errors.Is(err, database.ErrNotifyReceiverNotFound) {
		t.Errorf("更新不存在的接收人应返回 ErrNotifyReceiverNotFound，实际 %v", err)
	}
	if err := db.NotifyReceiver.Delete(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.NotifyReceiver.Delete(ctx, first.ID); !errors.Is(err, database.ErrNotifyReceiverNotFound) {
		t.Errorf("重复删除应返回 ErrNotifyReceiverNotFound，实际 %v", err)
	}
	if gone, err := db.NotifyReceiver.GetByID(ctx, first.ID); err != nil || gone != nil {
		t.Errorf("删除后应返回 (nil, nil)，实际 (%v, %v)", gone, err)
	}
}

// ==================== 路由规则 ====================

func TestNotifyRule_PositionAndReorder(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	repeat := 3600
	names := []string{"生产高严重度", "测试集群", "兜底"}
	ids := make([]int64, len(names))
	for i, name := range names {
		rule := &database.NotifyRule{
			Name: name, Enabled: true, ClusterIDs: []string{"prod"}, Sources: []string{"incident"},
			MinSeverity: "high", ReceiverIDs: []int64{1, 2}, RepeatIntervalSeconds: &repeat,
		}
		if err := db.NotifyRule.Create(ctx, rule); err != nil {
			t.Fatal(err)
		}
		if rule.Position != i+1 {
			t.Errorf("新规则应排在最后：%q 的位置是 %d", name, rule.Position)
		}
		ids[i] = rule.ID
	}

	got, err := db.NotifyRule.GetByID(ctx, ids[0])
	if err != nil || got == nil {
		t.Fatal(err)
	}
	if got.ClusterIDs[0] != "prod" || got.Sources[0] != "incident" || got.ReceiverIDs[1] != 2 || got.MinSeverity != "high" ||
		got.RepeatIntervalSeconds == nil || *got.RepeatIntervalSeconds != 3600 || got.GroupWaitSeconds != nil || got.GroupIntervalSeconds != nil {
		t.Errorf("取回的规则不对: %+v", got)
	}

	if err := db.NotifyRule.Reorder(ctx, []int64{ids[2], ids[0], ids[1]}); err != nil {
		t.Fatal(err)
	}
	list, err := db.NotifyRule.List(ctx)
	if err != nil || len(list) != 3 || list[0].ID != ids[2] || list[1].ID != ids[0] || list[2].ID != ids[1] || list[2].Position != 3 {
		t.Errorf("重排后的顺序不对: %+v (%v)", list, err)
	}

	for _, bad := range [][]int64{{ids[0], ids[1]}, {ids[0], ids[1], ids[1]}, {ids[0], ids[1], 999}} {
		if err := db.NotifyRule.Reorder(ctx, bad); !errors.Is(err, database.ErrNotifyRuleOrderMismatch) {
			t.Errorf("Reorder(%v) 应返回 ErrNotifyRuleOrderMismatch，实际 %v", bad, err)
		}
	}

	got.Name, got.Enabled, got.RepeatIntervalSeconds = "改名", false, nil
	if err := db.NotifyRule.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	again, _ := db.NotifyRule.GetByID(ctx, ids[0])
	if again.Name != "改名" || again.Enabled || again.RepeatIntervalSeconds != nil || again.Position != 2 {
		t.Errorf("更新不应改变位置，实际 %+v", again)
	}

	if err := db.NotifyRule.Delete(ctx, ids[0]); err != nil {
		t.Fatal(err)
	}
	if err := db.NotifyRule.Delete(ctx, ids[0]); !errors.Is(err, database.ErrNotifyRuleNotFound) {
		t.Errorf("重复删除应返回 ErrNotifyRuleNotFound，实际 %v", err)
	}
	if err := db.NotifyRule.Update(ctx, got); !errors.Is(err, database.ErrNotifyRuleNotFound) {
		t.Errorf("更新已删除的规则应返回 ErrNotifyRuleNotFound，实际 %v", err)
	}
}

// ==================== 静默 ====================

func TestNotifySilence_CreateExpireAndCleanup(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	silence := &database.NotifySilence{
		ClusterID: "prod", Namespace: "shop", StartsAt: notifyT0, EndsAt: notifyT0.Add(2 * time.Hour),
		Comment: "数据库升级", CreatedBy: "admin",
	}
	old := &database.NotifySilence{StartsAt: notifyT0.Add(-48 * time.Hour), EndsAt: notifyT0.Add(-47 * time.Hour), Comment: "旧的", CreatedBy: "admin"}
	for _, s := range []*database.NotifySilence{silence, old} {
		if err := db.NotifySilence.Create(ctx, s); err != nil {
			t.Fatal(err)
		}
	}

	list, err := db.NotifySilence.List(ctx)
	if err != nil || len(list) != 2 || list[0].ID != silence.ID {
		t.Fatalf("列表应有 2 条且开始时间新的在前，实际 %+v (%v)", list, err)
	}
	got := list[0]
	if got.ClusterID != "prod" || got.Namespace != "shop" || got.Subject != "" || got.Comment != "数据库升级" ||
		got.CreatedBy != "admin" || !got.StartsAt.Equal(notifyT0) || !got.EndsAt.Equal(notifyT0.Add(2*time.Hour)) || got.CreatedAt.IsZero() {
		t.Errorf("取回的静默不对: %+v", got)
	}

	// 提前结束
	if err := db.NotifySilence.SetEndsAt(ctx, silence.ID, notifyT0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	expired, _ := db.NotifySilence.GetByID(ctx, silence.ID)
	if !expired.EndsAt.Equal(notifyT0.Add(time.Hour)) {
		t.Errorf("结束时间没有改: %v", expired.EndsAt)
	}
	if err := db.NotifySilence.SetEndsAt(ctx, 999, notifyT0); !errors.Is(err, database.ErrNotifySilenceNotFound) {
		t.Errorf("不存在的静默应返回 ErrNotifySilenceNotFound，实际 %v", err)
	}

	deleted, err := db.NotifySilence.DeleteEndedBefore(ctx, notifyT0.Add(-24*time.Hour))
	if err != nil || deleted != 1 {
		t.Errorf("应清理 1 条过期已久的静默，实际 %d (%v)", deleted, err)
	}
}
