// store/memory/hub.go
// 内存数据中心
package memory

import (
	"AtlHyper/model/transport"
	"sync"
	"time"
)

// Hub 维护 Master 侧的全局 Envelope 池
type Hub struct {
	mu  sync.RWMutex
	all []EnvelopeRecord
}

// Global 全局唯一的 Hub 实例
var Global *Hub

// Init 初始化全局 Hub 实例
func Init() {
	Global = &Hub{
		all: make([]EnvelopeRecord, 0, 1024),
	}
}

// ensureInit 检查是否已初始化
func ensureInit() {
	if Global == nil {
		panic("memory hub not initialized: call memory.Init() at process boot")
	}
}

// Append 写入单条记录
func Append(rec EnvelopeRecord) {
	ensureInit()
	Global.mu.Lock()
	Global.all = append(Global.all, rec)
	Global.mu.Unlock()
}

// AppendBatch 批量写入
func AppendBatch(recs []EnvelopeRecord) {
	if len(recs) == 0 {
		return
	}
	ensureInit()
	Global.mu.Lock()
	Global.all = append(Global.all, recs...)
	Global.mu.Unlock()
}

// AppendEnvelope 写入单条 Envelope
func AppendEnvelope(env transport.Envelope) {
	ensureInit()
	Append(NewRecordFromStd(env))
}

// ReplaceLatest 原子替换同源同集群的数据
func ReplaceLatest(env transport.Envelope) int {
	ensureInit()

	rec := NewRecordFromStd(env)

	Global.mu.Lock()
	defer Global.mu.Unlock()

	deleted := 0
	dst := Global.all[:0]
	for _, r := range Global.all {
		if r.Source == rec.Source && r.ClusterID == rec.ClusterID {
			deleted++
			continue
		}
		dst = append(dst, r)
	}
	Global.all = append([]EnvelopeRecord(nil), dst...)
	Global.all = append(Global.all, rec)
	return deleted
}

// AppendEnvelopeBatch 批量写入 Envelope
func AppendEnvelopeBatch(envs []transport.Envelope) {
	if len(envs) == 0 {
		return
	}
	ensureInit()

	recs := make([]EnvelopeRecord, 0, len(envs))
	now := time.Now()
	for i := range envs {
		e := envs[i]
		recs = append(recs, EnvelopeRecord{
			Version:    e.Version,
			ClusterID:  e.ClusterID,
			Source:     e.Source,
			SentAtMs:   e.TimestampMs,
			EnqueuedAt: now,
			Payload:    e.Payload,
		})
	}
	AppendBatch(recs)
}

// Snapshot 返回当前池的只读副本
func Snapshot() []EnvelopeRecord {
	ensureInit()
	Global.mu.RLock()
	defer Global.mu.RUnlock()

	cp := make([]EnvelopeRecord, len(Global.all))
	copy(cp, Global.all)
	return cp
}

// Len 返回当前池内记录数
func Len() int {
	ensureInit()
	Global.mu.RLock()
	defer Global.mu.RUnlock()
	return len(Global.all)
}
