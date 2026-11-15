package repo_test

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

var incidentStartedAt = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

// storeSubjectIncident 建一条以工作负载为主体的事件
func storeSubjectIncident(t *testing.T, db *database.DB, id, clusterID, subject, severity string) {
	t.Helper()
	err := db.AIOpsIncident.CreateIncident(context.Background(), &database.AIOpsIncident{
		ID: id, ClusterID: clusterID, State: "warning", Severity: severity, Subject: subject,
		RootCause: "shop/pod/api-a", PeakRisk: 0.3, StartedAt: incidentStartedAt, CreatedAt: incidentStartedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func loadIncident(t *testing.T, db *database.DB, id string) *database.AIOpsIncident {
	t.Helper()
	inc, err := db.AIOpsIncident.GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if inc == nil {
		t.Fatalf("事件 %s 不存在", id)
	}
	return inc
}

func TestIncidentCreate_StoresSubject(t *testing.T) {
	db := openTestDB(t)
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "low")

	inc := loadIncident(t, db, "inc-1")
	if inc.Subject != "shop/deployment/api" {
		t.Errorf("subject = %q", inc.Subject)
	}
	if inc.EscalatedAt != nil || inc.CloseReason != "" {
		t.Errorf("新建的事件不应有升级时间与关闭原因: %+v", inc)
	}

	// 列表查询同样带出主体
	list, err := db.AIOpsIncident.List(context.Background(), database.AIOpsIncidentQueryOpts{ClusterID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Subject != "shop/deployment/api" {
		t.Errorf("列表应带出主体，实际 %+v", list)
	}
}

// 状态与严重度分开写：改状态不动严重度
func TestIncidentUpdateState_KeepsSeverity(t *testing.T) {
	db := openTestDB(t)
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "high")

	if err := db.AIOpsIncident.UpdateState(context.Background(), "inc-1", "recovery"); err != nil {
		t.Fatal(err)
	}

	inc := loadIncident(t, db, "inc-1")
	if inc.State != "recovery" || inc.Severity != "high" {
		t.Errorf("期望 recovery / high，实际 %s / %s", inc.State, inc.Severity)
	}
}

// 严重度是峰值：只升不降；峰值风险同样只增不减
func TestIncidentRaiseSeverity_OnlyRises(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "low")

	steps := []struct {
		severity     string
		peakRisk     float64
		wantSeverity string
		wantPeak     float64
	}{
		{"high", 0.75, "high", 0.75},
		{"medium", 0.55, "high", 0.75}, // 更低一档：不降
		{"critical", 0.95, "critical", 0.95},
		{"low", 0.2, "critical", 0.95},
	}
	for _, step := range steps {
		if err := db.AIOpsIncident.RaiseSeverity(ctx, "inc-1", step.severity, step.peakRisk); err != nil {
			t.Fatal(err)
		}
		inc := loadIncident(t, db, "inc-1")
		if inc.Severity != step.wantSeverity || inc.PeakRisk != step.wantPeak {
			t.Errorf("RaiseSeverity(%s, %v) 之后期望 %s / %v，实际 %s / %v",
				step.severity, step.peakRisk, step.wantSeverity, step.wantPeak, inc.Severity, inc.PeakRisk)
		}
	}
}

// 升级时间只记第一次
func TestIncidentMarkEscalated_WrittenOnce(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "medium")

	first := incidentStartedAt.Add(10 * time.Minute)
	if err := db.AIOpsIncident.MarkEscalated(ctx, "inc-1", first); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.MarkEscalated(ctx, "inc-1", first.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}

	inc := loadIncident(t, db, "inc-1")
	if inc.EscalatedAt == nil || !inc.EscalatedAt.Equal(first) {
		t.Errorf("升级时间应停在第一次（%v），实际 %v", first, inc.EscalatedAt)
	}
}

// 关闭后严重度仍在，并记下关闭原因
func TestIncidentResolve_KeepsSeverityAndRecordsReason(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "high")
	storeSubjectIncident(t, db, "inc-2", "c1", "shop/deployment/web", "low")

	resolvedAt := incidentStartedAt.Add(2 * time.Hour)
	if err := db.AIOpsIncident.Resolve(ctx, "inc-1", resolvedAt, "recovered"); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.Resolve(ctx, "inc-2", resolvedAt, "stale"); err != nil {
		t.Fatal(err)
	}

	recovered := loadIncident(t, db, "inc-1")
	if recovered.State != "stable" || recovered.Severity != "high" || recovered.CloseReason != "recovered" {
		t.Errorf("期望 stable / high / recovered，实际 %s / %s / %s", recovered.State, recovered.Severity, recovered.CloseReason)
	}
	if recovered.ResolvedAt == nil || !recovered.ResolvedAt.Equal(resolvedAt) || recovered.DurationS != 7200 {
		t.Errorf("关闭时间与时长不对: resolvedAt=%v durationS=%d", recovered.ResolvedAt, recovered.DurationS)
	}
	if stale := loadIncident(t, db, "inc-2"); stale.CloseReason != "stale" || stale.Severity != "low" {
		t.Errorf("期望 stale / low，实际 %s / %s", stale.CloseReason, stale.Severity)
	}

	// 按严重度筛选能查到已关闭的事件
	list, err := db.AIOpsIncident.List(ctx, database.AIOpsIncidentQueryOpts{ClusterID: "c1", Severity: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "inc-1" {
		t.Errorf("按严重度应查到已关闭的 inc-1，实际 %+v", list)
	}
}

// 按实体查历史：主体本身与成员实体都能查到，同一事件只出现一次
func TestIncidentListByEntity_MatchesSubjectAndMembers(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-api", "c1", "shop/deployment/api", "low")
	storeSubjectIncident(t, db, "inc-web", "c1", "shop/deployment/web", "low")
	storeSubjectIncident(t, db, "inc-other-cluster", "c2", "shop/deployment/api", "low")
	for _, member := range []struct{ incident, entity, role string }{
		{"inc-api", "shop/pod/api-a", "root_cause"},
		{"inc-api", "shop/pod/api-b", "affected"},
		{"inc-web", "shop/pod/web-a", "root_cause"},
	} {
		err := db.AIOpsIncident.AddEntity(ctx, &database.AIOpsIncidentEntity{
			IncidentID: member.incident, EntityKey: member.entity, EntityType: "pod", Role: member.role,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	since := incidentStartedAt.Add(-time.Hour)
	tests := []struct {
		name   string
		entity string
		want   []string
	}{
		{"按主体", "shop/deployment/api", []string{"inc-api"}},
		{"按领头实体", "shop/pod/api-a", []string{"inc-api"}},
		{"按其余成员", "shop/pod/api-b", []string{"inc-api"}},
		{"无关实体", "shop/pod/unknown", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incidents, err := db.AIOpsIncident.ListByEntity(ctx, "c1", tt.entity, since)
			if err != nil {
				t.Fatal(err)
			}
			var ids []string
			for _, inc := range incidents {
				ids = append(ids, inc.ID)
			}
			if len(ids) != len(tt.want) || (len(ids) > 0 && ids[0] != tt.want[0]) {
				t.Errorf("ListByEntity(%s) = %v，期望 %v", tt.entity, ids, tt.want)
			}
		})
	}
}

// 时长是「异常持续了多久」：进入恢复观察期时定下，观察期满关闭不再改
func TestIncidentDuration_ExcludesObservationPeriod(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "low")

	recoveredAt := incidentStartedAt.Add(6 * time.Minute)
	if err := db.AIOpsIncident.SetAnomalyDuration(ctx, "inc-1", recoveredAt); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.UpdateState(ctx, "inc-1", "recovery"); err != nil {
		t.Fatal(err)
	}
	if observing := loadIncident(t, db, "inc-1"); observing.DurationS != 360 {
		t.Errorf("恢复观察中的时长应是进入观察期时记下的 360 秒，实际 %d", observing.DurationS)
	}

	if err := db.AIOpsIncident.Resolve(ctx, "inc-1", recoveredAt.Add(6*time.Hour), "recovered"); err != nil {
		t.Fatal(err)
	}
	if closed := loadIncident(t, db, "inc-1"); closed.DurationS != 360 {
		t.Errorf("观察期满关闭后时长应仍是 360 秒，实际 %d", closed.DurationS)
	}
}

// 还在异常中就失联关闭的事件，时长算到关闭时刻
func TestIncidentDuration_StaleCloseWhileActiveCountsToClose(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "low")

	// 之前恢复过一次（时长已记），随后复发回到警告
	if err := db.AIOpsIncident.SetAnomalyDuration(ctx, "inc-1", incidentStartedAt.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.UpdateState(ctx, "inc-1", "warning"); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.Resolve(ctx, "inc-1", incidentStartedAt.Add(time.Hour), "stale"); err != nil {
		t.Fatal(err)
	}
	if closed := loadIncident(t, db, "inc-1"); closed.DurationS != 3600 {
		t.Errorf("时长应算到关闭时刻 3600 秒，实际 %d", closed.DurationS)
	}
}

// 活跃事件只数仍在异常中的；出事件最多的对象按主体归并
func TestIncidentStats_ActiveExcludesRecoveryAndTopGroupsBySubject(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "low")
	storeSubjectIncident(t, db, "inc-2", "c1", "shop/deployment/api", "low")
	storeSubjectIncident(t, db, "inc-3", "c1", "shop/deployment/web", "low")
	if err := db.AIOpsIncident.UpdateState(ctx, "inc-2", "recovery"); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.UpdateState(ctx, "inc-3", "incident"); err != nil {
		t.Fatal(err)
	}

	since := incidentStartedAt.Add(-time.Hour)
	stats, err := db.AIOpsIncident.GetIncidentStats(ctx, "c1", since)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalIncidents != 3 || stats.ActiveIncidents != 2 {
		t.Errorf("期望共 3 个、活跃 2 个（警告 + 事件中），实际 %d / %d", stats.TotalIncidents, stats.ActiveIncidents)
	}

	top, err := db.AIOpsIncident.TopRootCauses(ctx, "c1", since, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 || top[0].EntityKey != "shop/deployment/api" || top[0].Count != 2 {
		t.Errorf("期望按主体归并且 api 居首（2 次），实际 %+v", top)
	}
}
