package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/toolkit"

	"AtlHyper/atlhyper_master_v2/database"
)

type fakeEventLog struct {
	byResource  []*database.ClusterEvent
	byCluster   []*database.ClusterEvent
	resourceArg []string
	clusterOpts database.EventQueryOpts
}

func (f *fakeEventLog) ListByInvolvedResource(_ context.Context, clusterID, kind, ns, name string) ([]*database.ClusterEvent, error) {
	f.resourceArg = []string{clusterID, kind, ns, name}
	return f.byResource, nil
}
func (f *fakeEventLog) ListByCluster(_ context.Context, _ string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error) {
	f.clusterOpts = opts
	return f.byCluster, nil
}

func decodeEvents(t *testing.T, out string) []map[string]any {
	t.Helper()
	var resp struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("输出不是 JSON：%v\n%s", err, out)
	}
	return resp.Events
}

// 指定资源名：按资源查库，并按 since 过滤掉最近一次早于窗口的行
func TestEventHistory_ByResource(t *testing.T) {
	now := time.Now()
	f := &fakeEventLog{byResource: []*database.ClusterEvent{
		{InvolvedKind: "Pod", InvolvedNamespace: "ns", InvolvedName: "p", Reason: "Unhealthy", LastTimestamp: now.Add(-30 * time.Minute), FirstTimestamp: now.Add(-100 * time.Hour), Count: 6},
		{InvolvedKind: "Pod", InvolvedNamespace: "ns", InvolvedName: "p", Reason: "Old", LastTimestamp: now.Add(-10 * time.Hour)},
	}}
	out, err := eventHistory(context.Background(), f, "c1", eventHistoryParams{
		InvolvedKind: "Pod", Namespace: "ns", InvolvedName: "p", SinceMinutes: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.resourceArg; len(got) != 4 || got[0] != "c1" || got[1] != "Pod" || got[2] != "ns" || got[3] != "p" {
		t.Errorf("查询参数 = %v", got)
	}
	evs := decodeEvents(t, out)
	if len(evs) != 1 || evs[0]["reason"] != "Unhealthy" || evs[0]["count"] != float64(6) {
		t.Errorf("events = %v", evs)
	}
}

// 不指定资源名：按集群 + 时间查库（UTC，与库中格式一致），再按 namespace / kind 过滤
func TestEventHistory_ByClusterFilters(t *testing.T) {
	now := time.Now()
	f := &fakeEventLog{byCluster: []*database.ClusterEvent{
		{InvolvedKind: "Pod", InvolvedNamespace: "ns", InvolvedName: "a", Reason: "A", LastTimestamp: now},
		{InvolvedKind: "Pod", InvolvedNamespace: "other", InvolvedName: "b", Reason: "B", LastTimestamp: now},
		{InvolvedKind: "Node", InvolvedName: "n1", Reason: "C", LastTimestamp: now},
	}}
	out, err := eventHistory(context.Background(), f, "c1", eventHistoryParams{Namespace: "ns", SinceMinutes: maxEventHistoryMinutes})
	if err != nil {
		t.Fatal(err)
	}
	evs := decodeEvents(t, out)
	if len(evs) != 1 || evs[0]["reason"] != "A" {
		t.Errorf("events = %v", evs)
	}
	wantSince := now.Add(-maxEventHistoryMinutes * time.Minute)
	if d := f.clusterOpts.Since.Sub(wantSince); d < -time.Minute || d > time.Minute {
		t.Errorf("since = %v, want %v", f.clusterOpts.Since, wantSince)
	}
	if f.clusterOpts.Since.Location() != time.UTC {
		t.Errorf("since 应为 UTC（库中 RFC3339 为 Z 结尾，按字符串比较）")
	}
}

// 回看范围超过上限：在执行前按 schema 拦下（此前悄悄截成上限）
func TestEventHistory_SinceOverLimitRejected(t *testing.T) {
	reg := toolkit.NewRegistry()
	reg.Register(eventHistoryTool(&fakeEventLog{}))
	_, err := reg.Execute(context.Background(), "c1", func(string) bool { return true },
		&llm.ToolCall{Name: "get_event_history", Params: `{"since_minutes":999999}`})
	if !errors.Is(err, toolkit.ErrInvalidParams) {
		t.Errorf("err = %v, want ErrInvalidParams", err)
	}
}
