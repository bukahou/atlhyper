package core

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/incident"
	"AtlHyper/atlhyper_master_v2/aiops/statemachine"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

const (
	apiSubject = "shop/deployment/api"
	podA       = "shop/pod/api-7d9f-a"
	podB       = "shop/pod/api-7d9f-b"
)

func openIncidentRepo(t *testing.T) database.AIOpsIncidentRepository {
	t.Helper()
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "t.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())
	return db.AIOpsIncident
}

// newIncidentEngine 只装配与事件有关的部分：事件存储（真实 SQLite）+ 状态机
func newIncidentEngine(incidentRepo database.AIOpsIncidentRepository) *engine {
	e := &engine{incidentStore: incident.NewStore(incidentRepo), bgCtx: context.Background()}
	e.sm = statemachine.NewStateMachine(e, incident.NewIncidentID)
	return e
}

func podRisk(key string, rFinal float64) *aiops.EntityRisk {
	return &aiops.EntityRisk{EntityKey: key, EntityType: "pod", RLocal: rFinal, RFinal: rFinal}
}

func apiRisk(members ...*aiops.EntityRisk) *aiops.SubjectRisk {
	return &aiops.SubjectRisk{SubjectKey: apiSubject, RFinal: members[0].RFinal, Lead: members[0], Members: members}
}

func apiChange(kind aiops.TransitionKind, at time.Time, risk *aiops.SubjectRisk) aiops.IncidentTransition {
	return aiops.IncidentTransition{
		Kind: kind, IncidentID: "inc-1", ClusterID: "c1", SubjectKey: apiSubject,
		RootCause: risk.Lead.EntityKey, Severity: aiops.SeverityFromRisk(risk.RFinal),
		RFinal: risk.RFinal, PeakRisk: risk.RFinal, At: at,
	}
}

// 每次变化先落库、再发给订阅者：订阅者（AI 后台分析、通知）拿到变化时，事件已经查得到
func TestOnTransition_PersistsThenPublishes(t *testing.T) {
	e := newIncidentEngine(openIncidentRepo(t))
	ctx := context.Background()

	var received []aiops.IncidentTransition
	var storedWhenNotified *aiops.IncidentDetail
	e.SubscribeTransitions(func(tr aiops.IncidentTransition) {
		received = append(received, tr)
		storedWhenNotified = e.GetIncidentDetail(ctx, tr.IncidentID)
	})
	var secondSubscriber int
	e.SubscribeTransitions(func(aiops.IncidentTransition) { secondSubscriber++ })

	risk := apiRisk(podRisk(podA, 0.6), podRisk(podB, 0.3))
	created := apiChange(aiops.TransitionCreated, time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC), risk)
	e.OnTransition(ctx, created, risk)

	if len(received) != 1 || !reflect.DeepEqual(received[0], created) {
		t.Fatalf("订阅者应原样收到这条变化，实际 %+v", received)
	}
	if secondSubscriber != 1 {
		t.Errorf("每个订阅者都应收到，第二个收到 %d 次", secondSubscriber)
	}
	if storedWhenNotified == nil || storedWhenNotified.Subject != apiSubject || len(storedWhenNotified.Entities) != 2 {
		t.Errorf("通知订阅者时事件应已落库（含成员），实际 %+v", storedWhenNotified)
	}
}

func rsOwnedPod(namespace, name, replicaSet string) cluster.Pod {
	return cluster.Pod{Summary: cluster.PodSummary{Namespace: namespace, Name: name, OwnerKind: "ReplicaSet", OwnerName: replicaSet}}
}

// 评估前按主体聚合：同一个 Deployment 的两个 Pod 是一个主体、一条变化
func TestEvaluateSubjects_PodsOfOneWorkloadShareOneIncident(t *testing.T) {
	e := newIncidentEngine(openIncidentRepo(t))
	var received []aiops.IncidentTransition
	e.SubscribeTransitions(func(tr aiops.IncidentTransition) { received = append(received, tr) })

	// 升级过、正在观察期的事件：风险一回来立即复发，用例因此不必等待持续时间
	e.sm.RestoreEntry(&aiops.StateMachineEntry{
		ClusterID: "c1", SubjectKey: apiSubject, CurrentState: aiops.StateRecovery, IncidentID: "inc-1",
		ReachedIncident: true, Severity: "medium", PeakRisk: 0.6, RootCause: podA,
	})

	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			rsOwnedPod("shop", "api-7d9f-a", "api-7d9f"),
			rsOwnedPod("shop", "api-7d9f-b", "api-7d9f"),
		},
		ReplicaSets: []cluster.ReplicaSet{
			{CommonMeta: model_v3.CommonMeta{Namespace: "shop", Name: "api-7d9f", OwnerKind: "Deployment", OwnerName: "api"}},
		},
	}
	entityRisks := map[string]*aiops.EntityRisk{
		podA:               podRisk(podA, 0.4),
		podB:               podRisk(podB, 0.7),
		"shop/service/api": {EntityKey: "shop/service/api", EntityType: "service"},
	}

	e.evaluateSubjects("c1", snap, entityRisks, nil)

	if len(received) != 1 {
		t.Fatalf("两个 Pod 同属一个主体，应只有 1 条变化，实际 %d 条: %+v", len(received), received)
	}
	got := received[0]
	if got.Kind != aiops.TransitionRecurred || got.SubjectKey != apiSubject || got.IncidentID != "inc-1" {
		t.Errorf("变化应落在主体 %s 的事件上: %+v", apiSubject, got)
	}
	if got.RootCause != podB || !reflect.DeepEqual(got.Members, []string{podB, podA}) || got.RFinal != 0.7 {
		t.Errorf("领头是风险最高的 Pod、成员含两个 Pod: %+v", got)
	}
	// Pod 自己不再有各自的状态机条目；service 的主体是它自己
	if e.sm.GetEntry("c1", podA) != nil || e.sm.GetEntry("c1", podB) != nil {
		t.Error("归到工作负载的 Pod 不应再有自己的条目")
	}
	if e.sm.GetEntry("c1", "shop/service/api") == nil {
		t.Error("service 应以自身为主体被评估")
	}
}

// 重启后按主体把未关闭的事件恢复到状态机，条目的各项都从库里取回
func TestReloadActiveIncidents_RestoresBySubject(t *testing.T) {
	incidentRepo := openIncidentRepo(t)
	ctx := context.Background()
	startedAt := time.Now().Add(-10 * time.Hour).Truncate(time.Second)

	// 重启前：建事件 → 升级 → 恢复 → 复发 → 恢复（停在恢复观察期）
	before := newIncidentEngine(incidentRepo)
	high := apiRisk(podRisk(podA, 0.75))
	cleared := apiRisk(podRisk(podA, 0.05))
	steps := []struct {
		kind   aiops.TransitionKind
		offset time.Duration
		risk   *aiops.SubjectRisk
	}{
		{aiops.TransitionCreated, 0, high},
		{aiops.TransitionEscalated, 5 * time.Minute, high},
		{aiops.TransitionRecoveryStarted, time.Hour, cleared},
		{aiops.TransitionRecurred, 2 * time.Hour, high},
		{aiops.TransitionRecoveryStarted, 3 * time.Hour, cleared},
	}
	for _, step := range steps {
		change := apiChange(step.kind, startedAt.Add(step.offset), step.risk)
		change.Severity, change.PeakRisk = "high", 0.75
		before.OnTransition(ctx, change, step.risk)
	}
	// 另一个主体的事件已经关闭：不恢复
	closedRisk := &aiops.SubjectRisk{SubjectKey: "shop/service/web", RFinal: 0.3, Lead: podRisk("shop/service/web", 0.3)}
	closedRisk.Members = []*aiops.EntityRisk{closedRisk.Lead}
	closed := aiops.IncidentTransition{
		Kind: aiops.TransitionCreated, IncidentID: "inc-closed", ClusterID: "c1", SubjectKey: "shop/service/web",
		RootCause: "shop/service/web", Severity: "low", PeakRisk: 0.3, At: startedAt,
	}
	before.OnTransition(ctx, closed, closedRisk)
	closed.Kind, closed.At = aiops.TransitionClosed, startedAt.Add(7*time.Hour)
	before.OnTransition(ctx, closed, nil)

	// 重启：新的引擎、新的状态机，同一个库
	after := newIncidentEngine(incidentRepo)
	after.reloadActiveIncidents(ctx)

	entry := after.sm.GetEntry("c1", apiSubject)
	if entry == nil {
		t.Fatalf("未关闭的事件应按主体 %s 恢复", apiSubject)
	}
	if entry.IncidentID != "inc-1" || entry.CurrentState != aiops.StateRecovery || entry.SubjectKey != apiSubject {
		t.Errorf("条目的事件号 / 状态 / 主体不对: %+v", entry)
	}
	if !entry.ReachedIncident || entry.PeakRisk != 0.75 || entry.Severity != "high" || entry.Recurrence != 1 || entry.RootCause != podA {
		t.Errorf("升级过 / 峰值风险 / 峰值严重度 / 复发次数 / 领头应从库里取回: %+v", entry)
	}
	wantOccurrences := []int64{startedAt.Unix(), startedAt.Add(2 * time.Hour).Unix()}
	if !reflect.DeepEqual(entry.OccurrenceTimes, wantOccurrences) {
		t.Errorf("发生时刻 = %v, want %v（建事件 + 复发）", entry.OccurrenceTimes, wantOccurrences)
	}
	if want := startedAt.Add(3 * time.Hour).Unix(); entry.RecoveryStartedAt != want {
		t.Errorf("观察期起点 = %d, want %d（最后一次恢复）", entry.RecoveryStartedAt, want)
	}
	if entry.LastEvaluatedAt == 0 {
		t.Error("恢复的条目应记下最近评估时刻，失联检查据此计时")
	}

	if after.sm.GetEntry("c1", podA) != nil {
		t.Error("不应再按领头实体建条目")
	}
	if after.sm.GetEntry("c1", "shop/service/web") != nil {
		t.Error("已关闭的事件不应恢复到状态机")
	}
}
