// atlhyper_master_v2/datahub/memory/otel_ring.go
// OTel 时间线环形缓冲区
// 每个集群维护一个固定容量的 Ring Buffer，保存 OTel 快照历史
package memory

import (
	"sync"
	"time"

	"AtlHyper/model_v3/cluster"
)

// otelEntry 带时间戳的 OTel 快照（Ring Buffer 内部使用）
type otelEntry struct {
	snapshot  *cluster.OTelSnapshot
	timestamp time.Time
}

// OTelRing 固定容量环形缓冲区
type OTelRing struct {
	entries  []otelEntry
	head     int // 下一个写入位置
	count    int // 当前元素数量
	capacity int
	mu       sync.RWMutex
}

// NewOTelRing 创建 OTelRing
func NewOTelRing(capacity int) *OTelRing {
	if capacity <= 0 {
		capacity = 90
	}
	return &OTelRing{
		entries:  make([]otelEntry, capacity),
		capacity: capacity,
	}
}

// Add 添加一条 OTel 快照到环形缓冲区
func (r *OTelRing) Add(snapshot *cluster.OTelSnapshot, ts time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[r.head] = otelEntry{
		snapshot:  snapshot,
		timestamp: ts,
	}
	r.head = (r.head + 1) % r.capacity
	if r.count < r.capacity {
		r.count++
	}
}

// Latest 返回最新一条（snapshot + timestamp）
func (r *OTelRing) Latest() (*cluster.OTelSnapshot, time.Time) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return nil, time.Time{}
	}
	idx := (r.head - 1 + r.capacity) % r.capacity
	entry := r.entries[idx]
	return entry.snapshot, entry.timestamp
}

// Since 返回 since 之后的所有条目（按时间升序）
// 返回 snapshots 和 timestamps 两个平行切片
func (r *OTelRing) Since(since time.Time) ([]*cluster.OTelSnapshot, []time.Time) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return nil, nil
	}

	snapshots := make([]*cluster.OTelSnapshot, 0, r.count)
	timestamps := make([]time.Time, 0, r.count)

	start := (r.head - r.count + r.capacity) % r.capacity
	for i := 0; i < r.count; i++ {
		idx := (start + i) % r.capacity
		entry := r.entries[idx]
		if !entry.timestamp.Before(since) {
			snapshots = append(snapshots, entry.snapshot)
			timestamps = append(timestamps, entry.timestamp)
		}
	}
	return snapshots, timestamps
}

// Count 返回当前元素数量
func (r *OTelRing) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}
