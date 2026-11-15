package query

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/model"
)

var notifyQueryNow = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

// newNotifyQueryService 接在真实 SQLite（临时文件）上的查询服务，只装配通知相关的仓库
func newNotifyQueryService(t *testing.T) (*QueryService, *database.DB) {
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

	q := NewQueryService(QueryServiceDeps{NotifyRepos: NotifyRepos{
		Policy: db.NotifyPolicy, Alerts: db.NotifyAlert, Deliveries: db.NotifyDelivery,
		Receivers: db.NotifyReceiver, Rules: db.NotifyRule, Silences: db.NotifySilence,
	}})
	q.now = func() time.Time { return notifyQueryNow }
	return q, db
}

func seedReceiver(t *testing.T, db *database.DB, name string, enabled bool) int64 {
	t.Helper()
	receiver := &database.NotifyReceiver{Name: name, Type: "slack", Enabled: enabled,
		Slack: &database.SlackConfig{WebhookURL: "https://hooks.example.com/services/T000/B000/placeholder-not-real"}}
	if err := db.NotifyReceiver.Create(context.Background(), receiver); err != nil {
		t.Fatal(err)
	}
	return receiver.ID
}

func TestGetNotifyPolicy_Defaults(t *testing.T) {
	q, _ := newNotifyQueryService(t)
	policy, err := q.GetNotifyPolicy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !policy.IncidentAlertsEnabled || policy.MinSeverity != "low" || policy.WarningFlapThreshold != 3 || policy.K8sEventAlertsEnabled ||
		policy.DefaultReceiverIDs == nil {
		t.Errorf("没保存过策略时应返回出厂值: %+v", policy)
	}
}

func TestListNotifyAlertsAndDeliveries(t *testing.T) {
	q, db := newNotifyQueryService(t)
	ctx := context.Background()
	ended := notifyQueryNow
	for _, row := range []*database.NotifyAlert{
		{Fingerprint: "incident:a", Source: "incident", ClusterID: "prod", Severity: "high", Status: "firing", Kind: "firing", StartsAt: notifyQueryNow, UpdatedAt: notifyQueryNow},
		{Fingerprint: "incident:b", Source: "incident", ClusterID: "prod", Severity: "low", Status: "resolved", Kind: "resolved", StartsAt: notifyQueryNow.Add(-time.Hour), EndsAt: &ended, UpdatedAt: notifyQueryNow},
		{Fingerprint: "heartbeat:test", Source: "heartbeat", ClusterID: "test", Severity: "critical", Status: "firing", Kind: "firing", StartsAt: notifyQueryNow.Add(time.Minute), UpdatedAt: notifyQueryNow},
	} {
		if err := db.NotifyAlert.Upsert(ctx, row); err != nil {
			t.Fatal(err)
		}
	}

	alerts, total, err := q.ListNotifyAlerts(ctx, model.NotifyAlertQuery{Status: "firing", Limit: 1})
	if err != nil || total != 2 || len(alerts) != 1 || alerts[0].Fingerprint != "heartbeat:test" {
		t.Errorf("total 是符合条件的总数，不受分页影响；开始时间新的在前：%+v total=%d (%v)", alerts, total, err)
	}
	all, total, _ := q.ListNotifyAlerts(ctx, model.NotifyAlertQuery{})
	if total != 3 || len(all) != 3 {
		t.Errorf("不带条件应返回全部，实际 %d / %d", len(all), total)
	}
	none, total, err := q.ListNotifyAlerts(ctx, model.NotifyAlertQuery{ClusterID: "nowhere"})
	if err != nil || none == nil || len(none) != 0 || total != 0 {
		t.Errorf("没有匹配时应是空数组与 0，实际 %v / %d (%v)", none, total, err)
	}

	if err := db.NotifyDelivery.Create(ctx, &database.NotifyDelivery{BatchID: "b-1", Fingerprint: "incident:a", AlertTitle: "标题",
		Kind: "firing", Status: database.NotifyDeliveryNoReceiver, CreatedAt: notifyQueryNow, UpdatedAt: notifyQueryNow}); err != nil {
		t.Fatal(err)
	}
	deliveries, total, err := q.ListNotifyDeliveries(ctx, model.NotifyDeliveryQuery{Fingerprint: "incident:a"})
	if err != nil || total != 1 || len(deliveries) != 1 || deliveries[0].Status != "no-receiver" || deliveries[0].ReceiverID != 0 {
		t.Errorf("送达记录不对: %+v total=%d (%v)", deliveries, total, err)
	}
	empty, total, _ := q.ListNotifyDeliveries(ctx, model.NotifyDeliveryQuery{Status: "success"})
	if empty == nil || len(empty) != 0 || total != 0 {
		t.Errorf("没有匹配时应是空数组与 0，实际 %v / %d", empty, total)
	}
}

func TestListNotifySilences_FilterByState(t *testing.T) {
	q, db := newNotifyQueryService(t)
	ctx := context.Background()
	windows := map[string][2]time.Duration{
		"未开始":      {time.Hour, 2 * time.Hour},
		"生效中":      {-time.Hour, time.Hour},
		"已过期":      {-3 * time.Hour, -2 * time.Hour},
		"没开始就被撤销的": {time.Hour, -time.Minute},
	}
	for comment, window := range windows {
		silence := &database.NotifySilence{StartsAt: notifyQueryNow.Add(window[0]), EndsAt: notifyQueryNow.Add(window[1]), Comment: comment, CreatedBy: "admin"}
		if err := db.NotifySilence.Create(ctx, silence); err != nil {
			t.Fatal(err)
		}
	}

	wantByState := map[string][]string{
		"pending": {"未开始"},
		"active":  {"生效中"},
		"expired": {"已过期", "没开始就被撤销的"},
	}
	for state, wantComments := range wantByState {
		got, err := q.ListNotifySilences(ctx, state)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, item := range got {
			seen[item.Comment] = true
			if item.State != state {
				t.Errorf("state=%s 的结果里混进了 %s", state, item.State)
			}
		}
		if len(got) != len(wantComments) {
			t.Errorf("state=%s 应有 %d 条，实际 %d 条", state, len(wantComments), len(got))
		}
		for _, comment := range wantComments {
			if !seen[comment] {
				t.Errorf("state=%s 应包含「%s」", state, comment)
			}
		}
	}
	if all, _ := q.ListNotifySilences(ctx, ""); len(all) != 4 {
		t.Errorf("不带 state 应返回全部 4 条，实际 %d", len(all))
	}
}

func TestMatchNotifyRule(t *testing.T) {
	q, db := newNotifyQueryService(t)
	ctx := context.Background()
	oncall := seedReceiver(t, db, "值班频道", true)
	disabled := seedReceiver(t, db, "停用的频道", false)
	if err := db.NotifyRule.Create(ctx, &database.NotifyRule{Name: "生产高严重度", Enabled: true, ClusterIDs: []string{"prod"},
		Sources: []string{"incident"}, MinSeverity: "high", ReceiverIDs: []int64{oncall, disabled}}); err != nil {
		t.Fatal(err)
	}
	policy := database.DefaultNotifyPolicy()
	policy.MinSeverity = "medium"
	if err := db.NotifyPolicy.Save(ctx, policy); err != nil {
		t.Fatal(err)
	}

	// 命中规则：列出规则指向的全部接收人（含停用的，由 enabled 标明）
	hit, err := q.MatchNotifyRule(ctx, &model.NotifyRuleMatchRequest{ClusterID: "prod", Source: "incident", Severity: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if hit.MatchedRule == nil || hit.MatchedRule.Name != "生产高严重度" || len(hit.Receivers) != 2 || hit.BelowMinSeverity ||
		!hit.Receivers[0].Enabled || hit.Receivers[1].Enabled {
		t.Errorf("应命中规则: %+v", hit)
	}

	// 没命中：默认策略 = 全部已启用的接收人
	fallback, _ := q.MatchNotifyRule(ctx, &model.NotifyRuleMatchRequest{ClusterID: "test", Source: "incident", Severity: "high"})
	if fallback.MatchedRule != nil || len(fallback.Receivers) != 1 || fallback.Receivers[0].ID != oncall || fallback.BelowMinSeverity {
		t.Errorf("没命中规则应走默认策略: %+v", fallback)
	}

	// 默认策略下严重度不到门槛
	below, _ := q.MatchNotifyRule(ctx, &model.NotifyRuleMatchRequest{ClusterID: "test", Source: "incident", Severity: "low"})
	if below.MatchedRule != nil || !below.BelowMinSeverity || below.Receivers == nil {
		t.Errorf("严重度不到门槛应标明不会发送: %+v", below)
	}

	if _, err := q.MatchNotifyRule(ctx, &model.NotifyRuleMatchRequest{ClusterID: "prod", Source: "incident", Severity: "urgent"}); !errors.Is(err, model.ErrInvalidNotifyInput) {
		t.Errorf("严重度不合法应返回 ErrInvalidNotifyInput，实际 %v", err)
	}
}

func TestGetNotifyItems_NotFoundIsNil(t *testing.T) {
	q, _ := newNotifyQueryService(t)
	ctx := context.Background()
	if item, err := q.GetNotifyReceiver(ctx, 999); item != nil || err != nil {
		t.Errorf("不存在的接收人应返回 (nil, nil)，实际 (%v, %v)", item, err)
	}
	if item, err := q.GetNotifyRule(ctx, 999); item != nil || err != nil {
		t.Errorf("不存在的规则应返回 (nil, nil)，实际 (%v, %v)", item, err)
	}
	if item, err := q.GetNotifySilence(ctx, 999); item != nil || err != nil {
		t.Errorf("不存在的静默应返回 (nil, nil)，实际 (%v, %v)", item, err)
	}
}
