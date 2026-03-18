package memory

import (
	"testing"
	"time"

	agentmodel "AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
)

// helper: 创建最小快照
func makeSnapshot(clusterID string, fetchedAt time.Time) *cluster.ClusterSnapshot {
	return &cluster.ClusterSnapshot{
		ClusterID: clusterID,
		FetchedAt: fetchedAt,
	}
}

// helper: 创建含 OTel 的快照
func makeSnapshotWithOTel(clusterID string, fetchedAt time.Time, totalServices int) *cluster.ClusterSnapshot {
	return &cluster.ClusterSnapshot{
		ClusterID: clusterID,
		FetchedAt: fetchedAt,
		OTel: &cluster.OTelSnapshot{
			TotalServices: totalServices,
		},
	}
}

// helper: 创建含 Events 的快照
func makeSnapshotWithEvents(clusterID string, events []cluster.Event) *cluster.ClusterSnapshot {
	return &cluster.ClusterSnapshot{
		ClusterID: clusterID,
		FetchedAt: time.Now(),
		Events:    events,
	}
}

// helper: 创建 MemoryStore（短超时便于测试）
func newTestStore() *MemoryStore {
	return NewMemoryStore(5*time.Minute, 30*time.Second, 1*time.Minute)
}

// ==================== 快照存取 ====================

func TestSetGetSnapshot_Basic(t *testing.T) {
	store := newTestStore()
	now := time.Now()
	snap := makeSnapshot("cluster-a", now)

	if err := store.SetSnapshot("cluster-a", snap); err != nil {
		t.Fatalf("SetSnapshot error: %v", err)
	}

	got, err := store.GetSnapshot("cluster-a")
	if err != nil {
		t.Fatalf("GetSnapshot error: %v", err)
	}
	if got == nil {
		t.Fatal("GetSnapshot returned nil")
	}
	if got.ClusterID != "cluster-a" {
		t.Fatalf("ClusterID = %q, want %q", got.ClusterID, "cluster-a")
	}
	if !got.FetchedAt.Equal(now) {
		t.Fatalf("FetchedAt = %v, want %v", got.FetchedAt, now)
	}
}

func TestGetSnapshot_NotFound(t *testing.T) {
	store := newTestStore()

	got, err := store.GetSnapshot("nonexistent")
	if err != nil {
		t.Fatalf("GetSnapshot error: %v", err)
	}
	if got != nil {
		t.Fatalf("GetSnapshot returned %v, want nil", got)
	}
}

func TestSetSnapshot_Overwrite(t *testing.T) {
	store := newTestStore()
	now := time.Now()

	store.SetSnapshot("cluster-a", makeSnapshot("cluster-a", now))
	later := now.Add(time.Minute)
	store.SetSnapshot("cluster-a", makeSnapshot("cluster-a", later))

	got, _ := store.GetSnapshot("cluster-a")
	if got == nil {
		t.Fatal("GetSnapshot returned nil after overwrite")
	}
	if !got.FetchedAt.Equal(later) {
		t.Fatalf("FetchedAt = %v, want %v (latest)", got.FetchedAt, later)
	}
}

func TestSetSnapshot_MultiCluster(t *testing.T) {
	store := newTestStore()
	now := time.Now()

	store.SetSnapshot("cluster-a", makeSnapshot("cluster-a", now))
	store.SetSnapshot("cluster-b", makeSnapshot("cluster-b", now.Add(time.Second)))

	a, _ := store.GetSnapshot("cluster-a")
	b, _ := store.GetSnapshot("cluster-b")

	if a == nil || b == nil {
		t.Fatal("one of the snapshots is nil")
	}
	if a.ClusterID != "cluster-a" || b.ClusterID != "cluster-b" {
		t.Fatalf("clusters mixed up: a=%q, b=%q", a.ClusterID, b.ClusterID)
	}
}

func TestSetSnapshot_WithOTel(t *testing.T) {
	store := newTestStore()
	now := time.Now()

	store.SetSnapshot("cluster-a", makeSnapshotWithOTel("cluster-a", now, 5))

	entries, err := store.GetOTelTimeline("cluster-a", time.Time{})
	if err != nil {
		t.Fatalf("GetOTelTimeline error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("OTel timeline entries = %d, want 1", len(entries))
	}
	if entries[0].Snapshot.TotalServices != 5 {
		t.Fatalf("TotalServices = %d, want 5", entries[0].Snapshot.TotalServices)
	}
}

func TestSetSnapshot_NilOTel(t *testing.T) {
	store := newTestStore()
	now := time.Now()

	// OTel 为 nil 的快照不应追加到 Ring
	store.SetSnapshot("cluster-a", makeSnapshot("cluster-a", now))

	entries, err := store.GetOTelTimeline("cluster-a", time.Time{})
	if err != nil {
		t.Fatalf("GetOTelTimeline error: %v", err)
	}
	if entries != nil {
		t.Fatalf("OTel timeline = %v, want nil (no OTel was set)", entries)
	}
}

func TestSetSnapshot_UpdatesAgentInfo(t *testing.T) {
	store := newTestStore()
	now := time.Now()

	store.SetSnapshot("cluster-a", makeSnapshot("cluster-a", now))

	status, err := store.GetAgentStatus("cluster-a")
	if err != nil {
		t.Fatalf("GetAgentStatus error: %v", err)
	}
	if status == nil {
		t.Fatal("Agent status is nil after SetSnapshot")
	}
	if status.Status != agentmodel.StatusOnline {
		t.Fatalf("Agent status = %q, want %q", status.Status, agentmodel.StatusOnline)
	}
	if !status.LastSnapshot.Equal(now) {
		t.Fatalf("LastSnapshot = %v, want %v", status.LastSnapshot, now)
	}
}

// ==================== Agent 状态 ====================

func TestUpdateHeartbeat_NewAgent(t *testing.T) {
	store := newTestStore()
	before := time.Now()

	if err := store.UpdateHeartbeat("cluster-x"); err != nil {
		t.Fatalf("UpdateHeartbeat error: %v", err)
	}

	status, err := store.GetAgentStatus("cluster-x")
	if err != nil {
		t.Fatalf("GetAgentStatus error: %v", err)
	}
	if status == nil {
		t.Fatal("Agent status is nil after heartbeat")
	}
	if status.ClusterID != "cluster-x" {
		t.Fatalf("ClusterID = %q, want %q", status.ClusterID, "cluster-x")
	}
	if status.Status != agentmodel.StatusOnline {
		t.Fatalf("Status = %q, want %q", status.Status, agentmodel.StatusOnline)
	}
	if status.LastHeartbeat.Before(before) {
		t.Fatalf("LastHeartbeat %v is before test start %v", status.LastHeartbeat, before)
	}
}

func TestUpdateHeartbeat_ExistingAgent(t *testing.T) {
	store := newTestStore()

	store.UpdateHeartbeat("cluster-x")
	first, _ := store.GetAgentStatus("cluster-x")
	firstHB := first.LastHeartbeat

	// 短暂等待确保时间推进
	time.Sleep(time.Millisecond)

	store.UpdateHeartbeat("cluster-x")
	second, _ := store.GetAgentStatus("cluster-x")

	if !second.LastHeartbeat.After(firstHB) {
		t.Fatalf("second heartbeat %v should be after first %v", second.LastHeartbeat, firstHB)
	}
}

func TestGetAgentStatus_NotFound(t *testing.T) {
	store := newTestStore()

	status, err := store.GetAgentStatus("nonexistent")
	if err != nil {
		t.Fatalf("GetAgentStatus error: %v", err)
	}
	if status != nil {
		t.Fatalf("GetAgentStatus returned %v, want nil", status)
	}
}

func TestGetAgentStatus_ReturnsCorrectFields(t *testing.T) {
	store := newTestStore()
	now := time.Now()

	// 通过 SetSnapshot 创建 Agent（会设置 LastSnapshot）
	store.SetSnapshot("cluster-a", makeSnapshot("cluster-a", now))
	// 再发心跳更新 LastHeartbeat
	store.UpdateHeartbeat("cluster-a")

	status, _ := store.GetAgentStatus("cluster-a")
	if status == nil {
		t.Fatal("status is nil")
	}

	// 验证 4 个字段都有值
	if status.ClusterID != "cluster-a" {
		t.Fatalf("ClusterID = %q", status.ClusterID)
	}
	if status.Status != agentmodel.StatusOnline {
		t.Fatalf("Status = %q", status.Status)
	}
	if status.LastHeartbeat.IsZero() {
		t.Fatal("LastHeartbeat is zero")
	}
	if !status.LastSnapshot.Equal(now) {
		t.Fatalf("LastSnapshot = %v, want %v", status.LastSnapshot, now)
	}
}

func TestListAgents_Empty(t *testing.T) {
	store := newTestStore()

	agents, err := store.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents error: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("ListAgents returned %d agents, want 0", len(agents))
	}
}

func TestListAgents_MultipleAgents(t *testing.T) {
	store := newTestStore()

	store.UpdateHeartbeat("cluster-a")
	store.UpdateHeartbeat("cluster-b")
	store.UpdateHeartbeat("cluster-c")

	agents, err := store.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents error: %v", err)
	}
	if len(agents) != 3 {
		t.Fatalf("ListAgents returned %d agents, want 3", len(agents))
	}

	// 验证所有 clusterID 都存在（map 迭代无序，用 set 检查）
	ids := make(map[string]bool)
	for _, a := range agents {
		ids[a.ClusterID] = true
	}
	for _, expected := range []string{"cluster-a", "cluster-b", "cluster-c"} {
		if !ids[expected] {
			t.Fatalf("missing agent %q in ListAgents result", expected)
		}
	}
}

// ==================== 事件查询 ====================

func TestGetEvents_Basic(t *testing.T) {
	store := newTestStore()
	events := []cluster.Event{
		{Reason: "Pulled", Message: "image pulled"},
		{Reason: "Started", Message: "container started"},
	}

	store.SetSnapshot("cluster-a", makeSnapshotWithEvents("cluster-a", events))

	got, err := store.GetEvents("cluster-a")
	if err != nil {
		t.Fatalf("GetEvents error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetEvents returned %d events, want 2", len(got))
	}
	if got[0].Reason != "Pulled" || got[1].Reason != "Started" {
		t.Fatalf("events = [%q, %q], want [Pulled, Started]", got[0].Reason, got[1].Reason)
	}
}

func TestGetEvents_NotFound(t *testing.T) {
	store := newTestStore()

	got, err := store.GetEvents("nonexistent")
	if err != nil {
		t.Fatalf("GetEvents error: %v", err)
	}
	if got != nil {
		t.Fatalf("GetEvents returned %v, want nil", got)
	}
}

// ==================== OTel 时间线 ====================

func TestGetOTelTimeline_Basic(t *testing.T) {
	store := newTestStore()
	base := time.Now()

	for i := 0; i < 3; i++ {
		ts := base.Add(time.Duration(i) * 10 * time.Second)
		store.SetSnapshot("cluster-a", makeSnapshotWithOTel("cluster-a", ts, (i+1)*10))
	}

	entries, err := store.GetOTelTimeline("cluster-a", time.Time{})
	if err != nil {
		t.Fatalf("GetOTelTimeline error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("timeline entries = %d, want 3", len(entries))
	}

	// 验证按时间升序
	for i := 1; i < len(entries); i++ {
		if entries[i].Timestamp.Before(entries[i-1].Timestamp) {
			t.Fatalf("entry[%d] timestamp %v before entry[%d] %v", i, entries[i].Timestamp, i-1, entries[i-1].Timestamp)
		}
	}
}

func TestGetOTelTimeline_NotFound(t *testing.T) {
	store := newTestStore()

	entries, err := store.GetOTelTimeline("nonexistent", time.Time{})
	if err != nil {
		t.Fatalf("GetOTelTimeline error: %v", err)
	}
	if entries != nil {
		t.Fatalf("GetOTelTimeline returned %v, want nil", entries)
	}
}

func TestGetOTelTimeline_SinceFilter(t *testing.T) {
	store := newTestStore()
	base := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		ts := base.Add(time.Duration(i) * 10 * time.Second)
		store.SetSnapshot("cluster-a", makeSnapshotWithOTel("cluster-a", ts, i+1))
	}

	// since = base+25s → 应返回 t=30s 和 t=40s 的条目
	since := base.Add(25 * time.Second)
	entries, _ := store.GetOTelTimeline("cluster-a", since)

	if len(entries) != 2 {
		t.Fatalf("filtered entries = %d, want 2", len(entries))
	}
	if entries[0].Snapshot.TotalServices != 4 {
		t.Fatalf("first filtered TotalServices = %d, want 4", entries[0].Snapshot.TotalServices)
	}
}

func TestGetOTelTimeline_LightweightCopy(t *testing.T) {
	store := newTestStore()
	now := time.Now()

	// 创建含大字段的 OTel 快照
	snap := &cluster.ClusterSnapshot{
		ClusterID: "cluster-a",
		FetchedAt: now,
		OTel: &cluster.OTelSnapshot{
			TotalServices: 42,
			// 这些大字段不应出现在时间线中
			MetricsSummary: &metrics.Summary{TotalNodes: 10},
			APMTopology:    &apm.Topology{Nodes: []apm.TopologyNode{{Name: "svc-a"}}},
		},
	}
	store.SetSnapshot("cluster-a", snap)

	entries, _ := store.GetOTelTimeline("cluster-a", time.Time{})
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}

	entry := entries[0]
	// 标量字段应保留
	if entry.Snapshot.TotalServices != 42 {
		t.Fatalf("TotalServices = %d, want 42", entry.Snapshot.TotalServices)
	}
	// 大字段应被剥离（lightweightOTelCopy 不复制）
	if entry.Snapshot.MetricsSummary != nil {
		t.Fatal("MetricsSummary should be nil in lightweight copy")
	}
	if entry.Snapshot.APMTopology != nil {
		t.Fatal("APMTopology should be nil in lightweight copy")
	}
}

// ==================== 生命周期 ====================

func TestStartStop_Basic(t *testing.T) {
	store := newTestStore()

	if err := store.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := store.Stop(); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
}

func TestStop_Idempotent(t *testing.T) {
	store := newTestStore()
	store.Start()

	if err := store.Stop(); err != nil {
		t.Fatalf("first Stop error: %v", err)
	}

	// 第二次 Stop 不应 panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second Stop panicked: %v", r)
		}
	}()

	if err := store.Stop(); err != nil {
		t.Fatalf("second Stop error: %v", err)
	}
}
