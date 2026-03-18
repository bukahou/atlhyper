package memory

import (
	"testing"
	"time"

	"AtlHyper/model_v3/cluster"
)

// helper: 创建带标识的 OTelSnapshot
func makeOTelSnapshot(totalServices int) *cluster.OTelSnapshot {
	return &cluster.OTelSnapshot{
		TotalServices: totalServices,
	}
}

func TestOTelRing_AddAndCount(t *testing.T) {
	ring := NewOTelRing(5)
	now := time.Now()

	if ring.Count() != 0 {
		t.Fatalf("empty ring Count = %d, want 0", ring.Count())
	}

	for i := 1; i <= 3; i++ {
		ring.Add(makeOTelSnapshot(i), now.Add(time.Duration(i)*time.Second))
		if got := ring.Count(); got != i {
			t.Fatalf("after %d Add, Count = %d, want %d", i, got, i)
		}
	}
}

func TestOTelRing_Latest_Empty(t *testing.T) {
	ring := NewOTelRing(5)

	snap, ts := ring.Latest()
	if snap != nil {
		t.Fatalf("empty ring Latest snapshot = %v, want nil", snap)
	}
	if !ts.IsZero() {
		t.Fatalf("empty ring Latest timestamp = %v, want zero", ts)
	}
}

func TestOTelRing_Latest_NonEmpty(t *testing.T) {
	ring := NewOTelRing(5)
	now := time.Now()

	ring.Add(makeOTelSnapshot(10), now)
	ring.Add(makeOTelSnapshot(20), now.Add(time.Second))
	ring.Add(makeOTelSnapshot(30), now.Add(2*time.Second))

	snap, ts := ring.Latest()
	if snap == nil {
		t.Fatal("Latest snapshot is nil")
	}
	if snap.TotalServices != 30 {
		t.Fatalf("Latest TotalServices = %d, want 30", snap.TotalServices)
	}
	if !ts.Equal(now.Add(2 * time.Second)) {
		t.Fatalf("Latest timestamp = %v, want %v", ts, now.Add(2*time.Second))
	}
}

func TestOTelRing_CircularOverwrite(t *testing.T) {
	cap := 3
	ring := NewOTelRing(cap)
	now := time.Now()

	// 写入 5 条，容量只有 3，应覆盖最早的 2 条
	for i := 1; i <= 5; i++ {
		ring.Add(makeOTelSnapshot(i*100), now.Add(time.Duration(i)*time.Second))
	}

	// Count 不应超过 capacity
	if got := ring.Count(); got != cap {
		t.Fatalf("Count = %d, want %d (capacity)", got, cap)
	}

	// Latest 应该是最后写入的（500）
	snap, _ := ring.Latest()
	if snap == nil || snap.TotalServices != 500 {
		t.Fatalf("Latest TotalServices = %v, want 500", snap)
	}

	// Since(zero) 应返回 3 条，且是最近的 300/400/500
	snaps, _ := ring.Since(time.Time{})
	if len(snaps) != 3 {
		t.Fatalf("Since(zero) returned %d entries, want 3", len(snaps))
	}
	expected := []int{300, 400, 500}
	for i, s := range snaps {
		if s.TotalServices != expected[i] {
			t.Fatalf("Since[%d] TotalServices = %d, want %d", i, s.TotalServices, expected[i])
		}
	}
}

func TestOTelRing_Since_FilterByTime(t *testing.T) {
	ring := NewOTelRing(10)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 添加 5 条，间隔 10 秒
	for i := 0; i < 5; i++ {
		ring.Add(makeOTelSnapshot(i+1), base.Add(time.Duration(i)*10*time.Second))
	}

	// Since(base + 25s) 应返回 t=30s 和 t=40s 的条目（即第 4、5 条）
	since := base.Add(25 * time.Second)
	snaps, timestamps := ring.Since(since)

	if len(snaps) != 2 {
		t.Fatalf("Since returned %d entries, want 2", len(snaps))
	}
	if snaps[0].TotalServices != 4 || snaps[1].TotalServices != 5 {
		t.Fatalf("Since values = [%d, %d], want [4, 5]",
			snaps[0].TotalServices, snaps[1].TotalServices)
	}
	if !timestamps[0].Equal(base.Add(30 * time.Second)) {
		t.Fatalf("Since timestamps[0] = %v, want %v", timestamps[0], base.Add(30*time.Second))
	}
}

func TestOTelRing_Since_Empty(t *testing.T) {
	ring := NewOTelRing(5)

	snaps, timestamps := ring.Since(time.Time{})
	if snaps != nil {
		t.Fatalf("empty ring Since snapshots = %v, want nil", snaps)
	}
	if timestamps != nil {
		t.Fatalf("empty ring Since timestamps = %v, want nil", timestamps)
	}
}

func TestOTelRing_Since_AllFiltered(t *testing.T) {
	ring := NewOTelRing(5)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	ring.Add(makeOTelSnapshot(1), base)
	ring.Add(makeOTelSnapshot(2), base.Add(time.Second))

	// since 晚于所有条目
	future := base.Add(time.Hour)
	snaps, timestamps := ring.Since(future)

	if len(snaps) != 0 {
		t.Fatalf("Since(future) returned %d entries, want 0", len(snaps))
	}
	if len(timestamps) != 0 {
		t.Fatalf("Since(future) returned %d timestamps, want 0", len(timestamps))
	}
}

func TestOTelRing_DefaultCapacity(t *testing.T) {
	// capacity <= 0 应使用默认值 90
	tests := []struct {
		name     string
		capacity int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := NewOTelRing(tt.capacity)

			// 验证可以写满 90 条不 panic
			now := time.Now()
			for i := 0; i < 90; i++ {
				ring.Add(makeOTelSnapshot(i), now.Add(time.Duration(i)*time.Second))
			}
			if got := ring.Count(); got != 90 {
				t.Fatalf("Count = %d, want 90 (default capacity)", got)
			}

			// 第 91 条应循环覆盖，Count 仍为 90
			ring.Add(makeOTelSnapshot(999), now.Add(91*time.Second))
			if got := ring.Count(); got != 90 {
				t.Fatalf("after overflow Count = %d, want 90", got)
			}
		})
	}
}
