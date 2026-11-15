package incident

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
)

// 这一组用例走真实的 SQLite：要验证的是「落到库里的结果」（峰值严重度、升级时间只写一次、成员补写）

const (
	apiSubject = "shop/deployment/api"
	podA       = "shop/pod/api-a"
	podB       = "shop/pod/api-b"
	podC       = "shop/pod/api-c"
)

var transitionEpoch = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

func openSQLiteStore(t *testing.T) (*Store, database.AIOpsIncidentRepository) {
	t.Helper()
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "t.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())
	return NewStore(db.AIOpsIncident), db.AIOpsIncident
}

// podRisk 一个成员 Pod 的风险
func podRisk(key string, rFinal float64) *aiops.EntityRisk {
	return &aiops.EntityRisk{EntityKey: key, EntityType: "pod", RLocal: rFinal, RFinal: rFinal}
}

// workloadRisk 主体 shop/deployment/api 在一轮评估中的风险（成员按风险从高到低给出，第一个是领头）
func workloadRisk(members ...*aiops.EntityRisk) *aiops.SubjectRisk {
	return &aiops.SubjectRisk{SubjectKey: apiSubject, RFinal: members[0].RFinal, Lead: members[0], Members: members}
}

// change 拼一条事件变化：严重度、峰值、领头取自这一轮的主体风险，个别字段由 adjust 覆盖
func change(kind aiops.TransitionKind, at time.Time, risk *aiops.SubjectRisk, adjust ...func(*aiops.IncidentTransition)) aiops.IncidentTransition {
	transition := aiops.IncidentTransition{
		Kind: kind, IncidentID: "inc-1", ClusterID: "c1", SubjectKey: apiSubject, At: at,
	}
	if risk != nil {
		transition.RootCause = risk.Lead.EntityKey
		transition.Severity = aiops.SeverityFromRisk(risk.RFinal)
		transition.RFinal, transition.PeakRisk = risk.RFinal, risk.RFinal
	}
	for _, fn := range adjust {
		fn(&transition)
	}
	return transition
}

func loadDetail(t *testing.T, s *Store) *aiops.IncidentDetail {
	t.Helper()
	detail := s.GetIncident(context.Background(), "inc-1")
	if detail == nil {
		t.Fatal("事件 inc-1 不存在")
	}
	return detail
}

// rolesByEntity 事件实体表里每个实体的角色
func rolesByEntity(detail *aiops.IncidentDetail) map[string]string {
	roles := make(map[string]string, len(detail.Entities))
	for _, e := range detail.Entities {
		roles[e.EntityKey] = e.Role
	}
	return roles
}

// 建事件：记下主体、领头实体、严重度；成员写入实体表（领头 root_cause，其余 affected）
func TestApplyTransition_CreatedStoresSubjectAndMembers(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()
	risk := workloadRisk(podRisk(podB, 0.55), podRisk(podA, 0.3))

	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, risk), risk)

	detail := loadDetail(t, s)
	if detail.Subject != apiSubject || detail.SubjectType != "deployment" {
		t.Errorf("主体 = %q / %q", detail.Subject, detail.SubjectType)
	}
	if detail.State != aiops.StateWarning || detail.Severity != "medium" || detail.RootCause != podB || detail.PeakRisk != 0.55 {
		t.Errorf("事件行不对: %+v", detail.Incident)
	}
	if !detail.StartedAt.Equal(transitionEpoch) || detail.EscalatedAt != nil || detail.CloseReason != "" || detail.ResolvedAt != nil {
		t.Errorf("新建的事件不应有升级时间、关闭原因、结束时间: %+v", detail.Incident)
	}
	if got, want := rolesByEntity(detail), map[string]string{podB: "root_cause", podA: "affected"}; !reflect.DeepEqual(got, want) {
		t.Errorf("成员实体 = %v, want %v", got, want)
	}
	for _, e := range detail.Entities {
		if e.EntityType != "pod" || e.RFinal == 0 {
			t.Errorf("成员应带上类型与风险: %+v", e)
		}
	}
}

// 升级时间只写一次：复发后再次升级不覆盖
func TestApplyTransition_EscalatedAtWrittenOnce(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()
	risk := workloadRisk(podRisk(podA, 0.6))
	firstEscalation := transitionEpoch.Add(7 * time.Minute)

	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, risk), risk)
	s.ApplyTransition(ctx, change(aiops.TransitionEscalated, firstEscalation, risk), risk)
	if detail := loadDetail(t, s); detail.State != aiops.StateIncident || detail.EscalatedAt == nil || !detail.EscalatedAt.Equal(firstEscalation) {
		t.Fatalf("首次升级应记下升级时间: %+v", detail.Incident)
	}

	cleared := workloadRisk(podRisk(podA, 0.05))
	s.ApplyTransition(ctx, change(aiops.TransitionRecoveryStarted, transitionEpoch.Add(time.Hour), cleared), cleared)
	s.ApplyTransition(ctx, change(aiops.TransitionRecurred, transitionEpoch.Add(2*time.Hour), risk), risk)
	s.ApplyTransition(ctx, change(aiops.TransitionEscalated, transitionEpoch.Add(3*time.Hour), risk), risk)

	detail := loadDetail(t, s)
	if detail.EscalatedAt == nil || !detail.EscalatedAt.Equal(firstEscalation) {
		t.Errorf("升级时间应停在第一次（%v），实际 %v", firstEscalation, detail.EscalatedAt)
	}
	if detail.State != aiops.StateIncident || detail.Recurrence != 1 {
		t.Errorf("期望 事件中 / 复发 1 次，实际 %s / %d", detail.State, detail.Recurrence)
	}
}

// 关闭后严重度仍在；观察期满与失联两种关闭原因分开
func TestApplyTransition_ClosedKeepsSeverity(t *testing.T) {
	tests := []struct {
		name       string
		kind       aiops.TransitionKind
		wantReason string
	}{
		{"观察期满", aiops.TransitionClosed, aiops.CloseReasonRecovered},
		{"失联", aiops.TransitionClosedStale, aiops.CloseReasonStale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := openSQLiteStore(t)
			ctx := context.Background()
			risk := workloadRisk(podRisk(podA, 0.75))
			closedAt := transitionEpoch.Add(50 * time.Hour)

			s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, risk), risk)
			s.ApplyTransition(ctx, change(tt.kind, closedAt, nil, func(tr *aiops.IncidentTransition) {
				tr.RootCause, tr.Severity, tr.PeakRisk = podA, "high", 0.75
			}), nil)

			detail := loadDetail(t, s)
			if detail.State != aiops.StateStable || detail.Severity != "high" {
				t.Errorf("关闭后应为 stable 且严重度仍是 high，实际 %s / %q", detail.State, detail.Severity)
			}
			if detail.CloseReason != tt.wantReason {
				t.Errorf("关闭原因 = %q, want %q", detail.CloseReason, tt.wantReason)
			}
			if detail.ResolvedAt == nil || !detail.ResolvedAt.Equal(closedAt) || detail.DurationS != int64(50*time.Hour/time.Second) {
				t.Errorf("结束时间与时长不对: %v / %d", detail.ResolvedAt, detail.DurationS)
			}
			last := detail.Timeline[len(detail.Timeline)-1]
			if last.EventType != aiops.TimelineStateChange || last.EntityKey != podA {
				t.Errorf("关闭应在时间线留一条状态变更: %+v", last)
			}
		})
	}
}

// 状态转换时把新出现的成员补进实体表；领头换了，角色与事件的根因跟着换
func TestApplyTransition_AppendsNewMembersAndFollowsLead(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()

	created := workloadRisk(podRisk(podA, 0.6))
	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, created), created)

	// 升级那一轮：B 成了风险最高的，C 是新出现的，A 的风险变了
	escalated := workloadRisk(podRisk(podB, 0.8), podRisk(podA, 0.65), podRisk(podC, 0.3))
	s.ApplyTransition(ctx, change(aiops.TransitionEscalated, transitionEpoch.Add(7*time.Minute), escalated), escalated)

	detail := loadDetail(t, s)
	want := map[string]string{podB: "root_cause", podA: "affected", podC: "affected"}
	if got := rolesByEntity(detail); !reflect.DeepEqual(got, want) {
		t.Errorf("成员实体 = %v, want %v", got, want)
	}
	if detail.RootCause != podB {
		t.Errorf("事件的根因应换成当前领头 %s，实际 %s", podB, detail.RootCause)
	}
	for _, e := range detail.Entities {
		if e.EntityKey == podA && e.RFinal != 0.65 {
			t.Errorf("已有成员的风险应更新为本轮的值，实际 %v", e.RFinal)
		}
	}
}

// 之前的领头已经不在本轮成员里（Pod 重建换了名字）：降为 affected，不留下两个 root_cause
func TestApplyTransition_PreviousLeadGoneIsDemoted(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()

	created := workloadRisk(podRisk(podA, 0.6))
	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, created), created)
	recurred := workloadRisk(podRisk(podB, 0.4))
	s.ApplyTransition(ctx, change(aiops.TransitionRecurred, transitionEpoch.Add(time.Hour), recurred), recurred)

	detail := loadDetail(t, s)
	want := map[string]string{podB: "root_cause", podA: "affected"}
	if got := rolesByEntity(detail); !reflect.DeepEqual(got, want) {
		t.Errorf("成员实体 = %v, want %v", got, want)
	}
	for _, e := range detail.Entities {
		if e.EntityKey == podA && e.RFinal != 0.6 {
			t.Errorf("不在本轮的成员保留最后一次的风险值，实际 %v", e.RFinal)
		}
	}
}

// 严重度升高：更新严重度与峰值风险，时间线加一条；状态不变
func TestApplyTransition_SeverityRaised(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()

	created := workloadRisk(podRisk(podA, 0.3))
	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, created), created)
	raised := workloadRisk(podRisk(podA, 0.75))
	s.ApplyTransition(ctx, change(aiops.TransitionSeverityRaised, transitionEpoch.Add(time.Minute), raised), raised)

	detail := loadDetail(t, s)
	if detail.State != aiops.StateWarning || detail.Severity != "high" || detail.PeakRisk != 0.75 {
		t.Errorf("期望 warning / high / 0.75，实际 %s / %s / %v", detail.State, detail.Severity, detail.PeakRisk)
	}
	var found bool
	for _, entry := range detail.Timeline {
		if entry.EventType == aiops.TimelineSeverityRaised {
			found = true
			if entry.Detail != "严重度升高 → high (风险 75)" {
				t.Errorf("时间线文案 = %q", entry.Detail)
			}
		}
	}
	if !found {
		t.Error("时间线应有一条严重度升高")
	}
}

// 进入恢复观察时把至今的峰值风险写进库（两次变化之间涨上去的那部分此前只在内存里）
func TestApplyTransition_RecoveryPersistsPeakRisk(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()

	created := workloadRisk(podRisk(podA, 0.3))
	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, created), created)
	cleared := workloadRisk(podRisk(podA, 0.05))
	s.ApplyTransition(ctx, change(aiops.TransitionRecoveryStarted, transitionEpoch.Add(time.Hour), cleared, func(tr *aiops.IncidentTransition) {
		tr.Severity, tr.PeakRisk = "low", 0.48
	}), cleared)

	detail := loadDetail(t, s)
	if detail.State != aiops.StateRecovery || detail.PeakRisk != 0.48 || detail.Severity != "low" {
		t.Errorf("期望 recovery / 0.48 / low，实际 %s / %v / %s", detail.State, detail.PeakRisk, detail.Severity)
	}
	if got := rolesByEntity(detail); len(got) != 1 || got[podA] != "root_cause" {
		t.Errorf("恢复不改动成员实体，实际 %v", got)
	}
}

// 重启后取回「近 24 小时发生时刻」的原料：建事件的时刻 + 每次复发的时刻
func TestOccurrenceTimes_CreationAndRecurrences(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()
	risk := workloadRisk(podRisk(podA, 0.3))
	cleared := workloadRisk(podRisk(podA, 0.05))

	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, risk), risk)
	s.ApplyTransition(ctx, change(aiops.TransitionRecoveryStarted, transitionEpoch.Add(1*time.Hour), cleared), cleared)
	s.ApplyTransition(ctx, change(aiops.TransitionRecurred, transitionEpoch.Add(2*time.Hour), risk), risk)
	s.ApplyTransition(ctx, change(aiops.TransitionRecoveryStarted, transitionEpoch.Add(3*time.Hour), cleared), cleared)
	s.ApplyTransition(ctx, change(aiops.TransitionRecurred, transitionEpoch.Add(4*time.Hour), risk), risk)

	got := s.OccurrenceTimes(ctx, "inc-1", transitionEpoch)

	want := []int64{transitionEpoch.Unix(), transitionEpoch.Add(2 * time.Hour).Unix(), transitionEpoch.Add(4 * time.Hour).Unix()}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OccurrenceTimes() = %v, want %v", got, want)
	}
}

// 事件号各不相同
func TestNewIncidentID_Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := NewIncidentID(transitionEpoch)
		if id == "" || seen[id] {
			t.Fatalf("事件号重复或为空: %q", id)
		}
		seen[id] = true
	}
}
