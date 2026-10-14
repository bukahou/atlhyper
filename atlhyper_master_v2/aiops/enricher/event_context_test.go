package enricher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// fakeEventReader 按 kind/namespace/name 返回预置行，并记录调用
type fakeEventReader struct {
	rows  map[string][]*database.ClusterEvent // "Kind|ns|name" → rows
	err   error
	calls []string
}

func (f *fakeEventReader) ListByInvolvedResource(_ context.Context, _, kind, namespace, name string) ([]*database.ClusterEvent, error) {
	key := kind + "|" + namespace + "|" + name
	f.calls = append(f.calls, key)
	if f.err != nil {
		return nil, f.err
	}
	return f.rows[key], nil
}

func evRow(kind, ns, name, reason string, first, last time.Time, count int32) *database.ClusterEvent {
	return &database.ClusterEvent{
		InvolvedKind: kind, InvolvedNamespace: ns, InvolvedName: name,
		Type: "Warning", Reason: reason, Message: reason + " message",
		FirstTimestamp: first, LastTimestamp: last, Count: count,
	}
}

var (
	winFrom = time.Date(2026, 10, 14, 9, 6, 34, 0, time.UTC) // 开始前 15 分钟
	winTo   = time.Date(2026, 10, 14, 9, 44, 14, 0, time.UTC) // 恢复后 5 分钟
)

// 与窗口有交集的行入选（含跨越窗口起点的长期合并行），窗外排除；按最近一次倒序
func TestCollectWindowEvents_WindowOverlap(t *testing.T) {
	r := &fakeEventReader{rows: map[string][]*database.ClusterEvent{"Pod|ns|p": {
		evRow("Pod", "ns", "p", "Inside", winFrom.Add(5*time.Minute), winFrom.Add(10*time.Minute), 2),
		evRow("Pod", "ns", "p", "Spanning", winFrom.Add(-120*time.Hour), winFrom.Add(18*time.Minute), 6),
		evRow("Pod", "ns", "p", "Before", winFrom.Add(-3*time.Hour), winFrom.Add(-time.Minute), 1),
		evRow("Pod", "ns", "p", "After", winTo.Add(time.Minute), winTo.Add(2*time.Minute), 1),
	}}}
	rows, failed := collectWindowEvents(context.Background(), r, "c", []*database.AIOpsIncidentEntity{{EntityKey: "ns/pod/p"}}, winFrom, winTo)
	if failed {
		t.Fatal("不应失败")
	}
	var reasons []string
	for _, e := range rows {
		reasons = append(reasons, e.Reason)
	}
	if strings.Join(reasons, ",") != "Spanning,Inside" {
		t.Errorf("got %v，want [Spanning Inside]", reasons)
	}
}

func TestCollectWindowEvents_CapsAt20(t *testing.T) {
	var many []*database.ClusterEvent
	for i := 0; i < 30; i++ {
		many = append(many, evRow("Pod", "ns", "p", fmt.Sprintf("R%d", i), winFrom, winFrom.Add(time.Duration(i)*time.Second), 1))
	}
	r := &fakeEventReader{rows: map[string][]*database.ClusterEvent{"Pod|ns|p": many}}
	rows, _ := collectWindowEvents(context.Background(), r, "c", []*database.AIOpsIncidentEntity{{EntityKey: "ns/pod/p"}}, winFrom, winTo)
	if len(rows) != MaxEventEntries || rows[0].Reason != "R29" {
		t.Errorf("应保留最近 %d 条，got %d（首条 %s）", MaxEventEntries, len(rows), rows[0].Reason)
	}
}

// pod → Pod，node → Node（集群级资源 namespace 为空）；service / ingress 不查
func TestCollectWindowEvents_EntityKinds(t *testing.T) {
	r := &fakeEventReader{}
	collectWindowEvents(context.Background(), r, "c", []*database.AIOpsIncidentEntity{
		{EntityKey: "ns/pod/p"}, {EntityKey: "_cluster/node/n1"}, {EntityKey: "ns/service/s"}, {EntityKey: "ns/ingress/i"},
	}, winFrom, winTo)
	if strings.Join(r.calls, ",") != "Pod|ns|p,Node||n1" {
		t.Errorf("calls = %v", r.calls)
	}
}

// Count 是自首次以来的累计值：必须写成「累计 / 首次 / 最近」，不能读成窗口内次数
func TestRenderEvents_CountSemanticsAndStates(t *testing.T) {
	jst := time.FixedZone("JST", 9*3600)
	row := evRow("Pod", "ns", "p", "Unhealthy",
		time.Date(2026, 10, 9, 4, 45, 26, 0, time.UTC), time.Date(2026, 10, 14, 9, 24, 10, 0, time.UTC), 6)
	got := renderEvents([]*database.ClusterEvent{row}, 1, false, jst)
	for _, want := range []string{"[Pod ns/p] Unhealthy（累计 6 次，首次 10-09 13:45，最近 10-14 18:24）", "Unhealthy message"} {
		if !strings.Contains(got, want) {
			t.Errorf("缺 %q：\n%s", want, got)
		}
	}

	if got := renderEvents(nil, 0, false, jst); got != "事件窗口内无 Warning 事件" {
		t.Errorf("空：%q", got)
	}
	if got := renderEvents(nil, 0, true, jst); got != "（读取失败）" {
		t.Errorf("失败：%q", got)
	}
	if got := renderEvents(nil, 3, false, jst); !strings.Contains(got, "因长度限制省略") {
		t.Errorf("被截断成 0 条时不得说「无事件」：%q", got)
	}
}

func TestCollectWindowEvents_ReadError(t *testing.T) {
	r := &fakeEventReader{err: errors.New("db down")}
	_, failed := collectWindowEvents(context.Background(), r, "c", []*database.AIOpsIncidentEntity{{EntityKey: "ns/pod/p"}}, winFrom, winTo)
	if !failed {
		t.Error("读失败应标记 failed")
	}
}

// 功能三②：同时段（开始 ±15min）其他事件；排除自身；多个不同命名空间时附事实提示
func TestBuildConcurrent(t *testing.T) {
	t0 := time.Date(2026, 10, 14, 18, 21, 34, 0, time.Local)
	self := &database.AIOpsIncident{ID: "inc-self", RootCause: "app-a/pod/a-1", StartedAt: t0}
	others := []*database.AIOpsIncident{
		self,
		{ID: "inc-2", RootCause: "app-b/pod/b-1", StartedAt: t0.Add(10 * time.Second), PeakRisk: 0.56, State: "recovery"},
		{ID: "inc-3", RootCause: "app-b/pod/b-2", StartedAt: t0.Add(4 * time.Minute), PeakRisk: 0.47, State: "stable"},
	}
	got := buildConcurrent(self, excludeIncident(others, self.ID), 2)
	for _, want := range []string{"app-b/pod/b-1", "18:21:44", "峰值风险 56", "recovery", "app-b/pod/b-2", "同时段 2 个不同命名空间出现事件"} {
		if !strings.Contains(got, want) {
			t.Errorf("缺 %q：\n%s", want, got)
		}
	}
	if strings.Contains(got, "inc-self") {
		t.Errorf("不得含自身：\n%s", got)
	}
	if got := buildConcurrent(self, nil, 0); got != "同时段（开始前后 15 分钟）无其他事件" {
		t.Errorf("无：%q", got)
	}
	if got := buildConcurrent(self, nil, 4); !strings.Contains(got, "因长度限制省略") {
		t.Errorf("截断：%q", got)
	}
}

// 组装：event 窗口 = 开始前 15min → 恢复后 5min；同时段按开始 ±15min 查
func TestAssembleIncidentContext_EventsAndConcurrent(t *testing.T) {
	t0 := time.Date(2026, 10, 14, 18, 21, 34, 0, time.FixedZone("JST", 9*3600))
	inc := &database.AIOpsIncident{ID: "inc-1", ClusterID: "c", State: "recovery", RootCause: "ns/pod/p", StartedAt: t0}
	repo := &mockIncidentRepo{
		incident: inc,
		entities: []*database.AIOpsIncidentEntity{{IncidentID: "inc-1", EntityKey: "ns/pod/p", Role: "root_cause"}},
		timeline: []*database.AIOpsIncidentTimeline{
			{Timestamp: t0.Add(17*time.Minute + 40*time.Second), EventType: "recovery_started", Detail: "状态变更 → recovery (风险 0)"},
		},
		listed: []*database.AIOpsIncident{inc, {ID: "inc-2", RootCause: "other/pod/q", StartedAt: t0}},
	}
	reader := &fakeEventReader{rows: map[string][]*database.ClusterEvent{"Pod|ns|p": {
		evRow("Pod", "ns", "p", "Unhealthy", t0.Add(-120*time.Hour), t0.Add(2*time.Minute+36*time.Second), 6),
		evRow("Pod", "ns", "p", "Late", t0.Add(30*time.Minute), t0.Add(31*time.Minute), 1), // 恢复 + 5min 之后
	}}}
	e := NewEnricher(repo, nil, newMockAIService("{}"))
	e.SetEventSource(reader)

	ev, err := e.assembleIncidentContext(context.Background(), "inc-1")
	if err != nil {
		t.Fatal(err)
	}
	pc := ev.promptContext(0)
	if !strings.Contains(pc.K8sEvents, "Unhealthy") || strings.Contains(pc.K8sEvents, "Late") {
		t.Errorf("event 窗口不对：\n%s", pc.K8sEvents)
	}
	if !strings.Contains(pc.ConcurrentIncidents, "other/pod/q") {
		t.Errorf("应列出同时段事件：\n%s", pc.ConcurrentIncidents)
	}
	if !repo.listOpts.From.Equal(t0.Add(-15*time.Minute)) || !repo.listOpts.To.Equal(t0.Add(15*time.Minute)) || repo.listOpts.ClusterID != "c" {
		t.Errorf("同时段查询条件 = %+v", repo.listOpts)
	}

	// 未注入 event 源：不出现该节（而不是写「无事件」误导）
	e2 := NewEnricher(repo, nil, newMockAIService("{}"))
	ev2, _ := e2.assembleIncidentContext(context.Background(), "inc-1")
	if pc2 := ev2.promptContext(0); pc2.K8sEvents != "" {
		t.Errorf("无 event 源时应为空：%q", pc2.K8sEvents)
	}
}
