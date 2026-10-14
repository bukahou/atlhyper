package enricher

import (
	"context"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

// 功能三①：异常时长与 48h 观察期分开写（曾把累计到当前时刻的 2.4h 当成持续时间）
func TestBuildSummary_SeparatesActiveAndObservation(t *testing.T) {
	t0 := time.Date(2026, 10, 14, 18, 21, 34, 0, time.Local)
	inc := &database.AIOpsIncident{ID: "inc-1", State: "recovery", Severity: "low", StartedAt: t0, DurationS: 9000}
	timeline := []*database.AIOpsIncidentTimeline{
		{Timestamp: t0, EventType: aiops.TimelineStateChange, Detail: "状态变更: healthy → warning"},
		{Timestamp: t0.Add(17*time.Minute + 40*time.Second), EventType: aiops.TimelineRecoveryStarted, Detail: "状态变更 → recovery (风险 0)"},
	}
	got := buildSummary(inc, timeline)
	for _, want := range []string{"异常持续: 17 分钟（18:21:34 → 18:39:14）", "恢复观察期", "距开始 2.5 小时", "48h"} {
		if !strings.Contains(got, want) {
			t.Errorf("缺 %q：\n%s", want, got)
		}
	}
	if strings.Contains(got, "| 持续:") {
		t.Errorf("不得再输出累计到当前时刻的「持续」：\n%s", got)
	}

	ongoing := &database.AIOpsIncident{ID: "inc-2", State: "incident", StartedAt: t0, DurationS: 720}
	if got := buildSummary(ongoing, timeline[:1]); !strings.Contains(got, "异常持续: 12 分钟（进行中）") {
		t.Errorf("进行中：\n%s", got)
	}

	noRecord := &database.AIOpsIncident{ID: "inc-3", State: "recovery", StartedAt: t0, DurationS: 720}
	if got := buildSummary(noRecord, timeline[:1]); !strings.Contains(got, "异常持续: 未记录") {
		t.Errorf("recovery 但时间线无恢复记录：\n%s", got)
	}
}

// 功能三③：Pod → ReplicaSet → Deployment；解析不到就说未知，不靠去 hash 猜名
func TestPodOwner_Resolution(t *testing.T) {
	rs := cluster.ReplicaSet{CommonMeta: model_v3.CommonMeta{Name: "web-7d9f", Namespace: "ns", OwnerKind: "Deployment", OwnerName: "web"}}
	pod := func(name, ownerKind, ownerName string) cluster.Pod {
		p := podWith("ns", name, cluster.PodContainerDetail{Name: "app", State: "running"})
		p.Summary.OwnerKind, p.Summary.OwnerName = ownerKind, ownerName
		return p
	}
	snap := &cluster.ClusterSnapshot{
		ReplicaSets: []cluster.ReplicaSet{rs},
		Pods: []cluster.Pod{
			pod("web-7d9f-a", "ReplicaSet", "web-7d9f"),
			pod("api-5c4b-b", "ReplicaSet", "api-5c4b"), // RS 不在快照
			pod("db-0", "StatefulSet", "db"),
		},
	}
	entities := []*database.AIOpsIncidentEntity{
		{EntityKey: "ns/pod/web-7d9f-a"}, {EntityKey: "ns/pod/api-5c4b-b"}, {EntityKey: "ns/pod/db-0"},
	}
	got := buildPodStateContext(snap, entities)
	for _, want := range []string{
		"所属: Deployment ns/web（经 ReplicaSet web-7d9f）",
		"所属: ReplicaSet api-5c4b（上级未知）",
		"所属: StatefulSet ns/db",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("缺 %q：\n%s", want, got)
		}
	}
	if strings.Contains(got, "Deployment ns/api") {
		t.Errorf("不得猜 Deployment 名：\n%s", got)
	}
}

// 后台与深度两条路径共用同一份证据：深度分析也要拿到 Pod 状态与历史
func TestAssembleIncidentContext_BothPathsShareEvidence(t *testing.T) {
	inc := makeTestIncident()
	inc.RootCause = "default/pod/api-abc"
	repo := &mockIncidentRepo{
		incident: inc,
		entities: []*database.AIOpsIncidentEntity{{IncidentID: inc.ID, EntityKey: "default/pod/api-abc", Role: "root_cause"}},
		timeline: makeTestTimeline(),
		byEntity: []*database.AIOpsIncident{{ID: "inc-old", RootCause: "default/pod/api-abc", StartedAt: time.Now().Add(-48 * time.Hour), DurationS: 600}},
	}
	e := NewEnricher(repo, nil, newMockAIService("{}"))
	e.SetStore(&snapshotStore{snap: &cluster.ClusterSnapshot{Pods: []cluster.Pod{
		podWith("default", "api-abc", cluster.PodContainerDetail{Name: "app", State: "running", RestartCount: 2}),
	}}})

	ev, err := e.assembleIncidentContext(context.Background(), inc.ID)
	if err != nil {
		t.Fatal(err)
	}
	pc := ev.promptContext(0)
	if pc.PodStates == "" || pc.HistoricalContext == "" || strings.Contains(pc.HistoricalContext, "inc-test-001") {
		t.Errorf("应含 Pod 状态与（排除自身的）历史：%+v", pc)
	}
}

// snapshotStore 只实现 GetSnapshot 的 datahub.Store
type snapshotStore struct {
	datahub.Store
	snap *cluster.ClusterSnapshot
}

func (s *snapshotStore) GetSnapshot(string) (*cluster.ClusterSnapshot, error) { return s.snap, nil }
