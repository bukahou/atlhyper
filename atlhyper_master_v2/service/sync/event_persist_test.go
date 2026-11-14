package sync

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub/memory"
	"AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

// fakeEventRepo 只记录 UpsertBatch 收到的批次；其余方法未实现（调用即 panic，测试不应触及）
type fakeEventRepo struct {
	database.ClusterEventRepository
	batches [][]*database.ClusterEvent
}

func (f *fakeEventRepo) UpsertBatch(_ context.Context, events []*database.ClusterEvent) error {
	f.batches = append(f.batches, events)
	return nil
}

func backOffEvent(name string, count int32, last time.Time, message string) cluster.Event {
	return cluster.Event{
		CommonMeta:     model_v3.CommonMeta{Name: name, Namespace: "default"},
		Type:           "Warning",
		Reason:         "BackOff",
		Message:        message,
		InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Namespace: "default", Name: "nginx"},
		Count:          count,
		FirstTimestamp: last.Add(-time.Hour),
		LastTimestamp:  last,
	}
}

func newSyncFixture(t *testing.T) (*EventPersistService, *memory.MemoryStore, *fakeEventRepo) {
	t.Helper()
	store := memory.NewMemoryStore(time.Hour, time.Minute, time.Hour)
	repo := &fakeEventRepo{}
	return NewEventPersistService(store, repo, EventPersistConfig{}), store, repo
}

func setEvents(t *testing.T, store *memory.MemoryStore, events ...cluster.Event) {
	t.Helper()
	if err := store.SetSnapshot("c1", &cluster.ClusterSnapshot{ClusterID: "c1", Events: events}); err != nil {
		t.Fatal(err)
	}
}

// 快照没有变化时不写库：每 10 秒一份的快照不应每次都重写全部事件。
func TestSync_UnchangedSnapshotIsNotRewritten(t *testing.T) {
	svc, store, repo := newSyncFixture(t)
	last := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	setEvents(t, store, backOffEvent("nginx.1", 3, last, "back-off"))

	for i := 0; i < 3; i++ {
		if err := svc.Sync("c1"); err != nil {
			t.Fatal(err)
		}
	}
	if len(repo.batches) != 1 {
		t.Fatalf("同一份快照同步 3 次，写库 %d 次，want 1", len(repo.batches))
	}
}

// 每份快照都是重新解码出来的：同一时刻的时间值带着不同的时区对象，不能因此被当成有变化。
func TestSync_SameInstantInFreshlyDecodedSnapshotIsUnchanged(t *testing.T) {
	svc, store, repo := newSyncFixture(t)
	instant := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		decoded := instant.In(time.FixedZone("", 9*3600)) // 每次一个新的时区对象
		setEvents(t, store, backOffEvent("nginx.1", 3, decoded, "back-off"))
		if err := svc.Sync("c1"); err != nil {
			t.Fatal(err)
		}
	}
	if len(repo.batches) != 1 {
		t.Fatalf("三份内容相同的快照，写库 %d 次，want 1", len(repo.batches))
	}
}

// 事件有变化时只写变化的那几条。
func TestSync_WritesOnlyChangedEvents(t *testing.T) {
	svc, store, repo := newSyncFixture(t)
	last := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	other := backOffEvent("redis.1", 2, last, "back-off")
	other.InvolvedObject.Name = "redis"

	setEvents(t, store, backOffEvent("nginx.1", 3, last, "back-off"), other)
	if err := svc.Sync("c1"); err != nil {
		t.Fatal(err)
	}
	setEvents(t, store, backOffEvent("nginx.1", 4, last.Add(time.Minute), "back-off"), other)
	if err := svc.Sync("c1"); err != nil {
		t.Fatal(err)
	}

	if len(repo.batches) != 2 {
		t.Fatalf("写库 %d 次，want 2", len(repo.batches))
	}
	second := repo.batches[1]
	if len(second) != 1 || second[0].InvolvedName != "nginx" || second[0].Count != 4 {
		t.Errorf("第二次应只写变化的 nginx 事件（count=4），实际 %+v", second)
	}
}

// 同一对象同一原因在 K8s 里可能对应多条事件（消息不同）：合并成一行，次数相加，取最新一条的消息。
func TestSync_MergesEventsSharingDedupKey(t *testing.T) {
	svc, store, repo := newSyncFixture(t)
	last := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	setEvents(t, store,
		backOffEvent("nginx.1", 3, last.Add(-5*time.Minute), "container a"),
		backOffEvent("nginx.2", 4, last, "container b"),
	)
	if err := svc.Sync("c1"); err != nil {
		t.Fatal(err)
	}
	if len(repo.batches) != 1 || len(repo.batches[0]) != 1 {
		t.Fatalf("期望合并成 1 行，实际 %+v", repo.batches)
	}
	merged := repo.batches[0][0]
	if merged.Count != 7 {
		t.Errorf("count = %d, want 7（3 + 4）", merged.Count)
	}
	if merged.Message != "container b" || !merged.LastTimestamp.Equal(last) {
		t.Errorf("应取最新一条的消息与时间，实际 %q %v", merged.Message, merged.LastTimestamp)
	}
	if !merged.FirstTimestamp.Equal(last.Add(-5*time.Minute - time.Hour)) {
		t.Errorf("first_timestamp 应取最早的，实际 %v", merged.FirstTimestamp)
	}
}

// 写库失败时不记为已持久化，下一份快照要重试。
func TestSync_FailedWriteIsRetried(t *testing.T) {
	svc, store, _ := newSyncFixture(t)
	failing := &failingEventRepo{fail: true}
	svc.eventRepo = failing
	last := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	setEvents(t, store, backOffEvent("nginx.1", 3, last, "back-off"))

	if err := svc.Sync("c1"); err == nil {
		t.Fatal("期望返回写库错误")
	}
	failing.fail = false
	if err := svc.Sync("c1"); err != nil {
		t.Fatal(err)
	}
	if failing.succeeded != 1 {
		t.Errorf("失败后应重试写入，成功次数 = %d, want 1", failing.succeeded)
	}
}

type failingEventRepo struct {
	database.ClusterEventRepository
	fail      bool
	succeeded int
}

func (f *failingEventRepo) UpsertBatch(_ context.Context, _ []*database.ClusterEvent) error {
	if f.fail {
		return context.DeadlineExceeded
	}
	f.succeeded++
	return nil
}
