package repo_test

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

func warningEvent(count int32, last time.Time, message string) *database.ClusterEvent {
	return &database.ClusterEvent{
		DedupKey:          "key-1",
		ClusterID:         "c1",
		Namespace:         "default",
		Name:              "nginx.17a",
		Type:              "Warning",
		Reason:            "BackOff",
		Message:           message,
		InvolvedKind:      "Pod",
		InvolvedName:      "nginx",
		InvolvedNamespace: "default",
		FirstTimestamp:    last.Add(-time.Hour),
		LastTimestamp:     last,
		Count:             count,
	}
}

func storedEvent(t *testing.T, db *database.DB) *database.ClusterEvent {
	t.Helper()
	events, err := db.Event.ListByCluster(context.Background(), "c1", database.EventQueryOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("期望 1 行，实际 %d 行", len(events))
	}
	return events[0]
}

// 同一条事件随每份快照反复写入，次数必须保持 K8s 报告的值，不能越写越大。
func TestEventUpsert_RepeatedWriteKeepsCount(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	last := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		if err := db.Event.UpsertBatch(ctx, []*database.ClusterEvent{warningEvent(3, last, "back-off")}); err != nil {
			t.Fatal(err)
		}
	}
	if got := storedEvent(t, db).Count; got != 3 {
		t.Errorf("重复写入 5 次后 count = %d, want 3", got)
	}
}

// 事件再次发生：次数跟随 K8s 增长，最后发生时间与消息更新。
func TestEventUpsert_FollowsGrowth(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	last := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	if err := db.Event.Upsert(ctx, warningEvent(3, last, "back-off 10s")); err != nil {
		t.Fatal(err)
	}
	if err := db.Event.Upsert(ctx, warningEvent(7, last.Add(5*time.Minute), "back-off 40s")); err != nil {
		t.Fatal(err)
	}
	got := storedEvent(t, db)
	if got.Count != 7 {
		t.Errorf("count = %d, want 7", got.Count)
	}
	if !got.LastTimestamp.Equal(last.Add(5 * time.Minute)) {
		t.Errorf("last_timestamp = %v, want %v", got.LastTimestamp, last.Add(5*time.Minute))
	}
	if got.Message != "back-off 40s" {
		t.Errorf("message = %q, want 最新一次的消息", got.Message)
	}
}

// 较旧的记录后到（K8s 事件过期后重新从 1 计数，或乱序）：次数、时间、消息都不回退。
func TestEventUpsert_OlderRecordDoesNotRegress(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	last := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	if err := db.Event.Upsert(ctx, warningEvent(7, last, "back-off 40s")); err != nil {
		t.Fatal(err)
	}
	if err := db.Event.Upsert(ctx, warningEvent(1, last.Add(-10*time.Minute), "back-off 10s")); err != nil {
		t.Fatal(err)
	}
	got := storedEvent(t, db)
	if got.Count != 7 {
		t.Errorf("count = %d, want 7（不回退）", got.Count)
	}
	if !got.LastTimestamp.Equal(last) {
		t.Errorf("last_timestamp = %v, want %v（不回退）", got.LastTimestamp, last)
	}
	if got.Message != "back-off 40s" {
		t.Errorf("message = %q, want 保留较新的消息", got.Message)
	}
}
