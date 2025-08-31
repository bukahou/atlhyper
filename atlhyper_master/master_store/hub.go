// external/master_store/hub.go
package master_store

import (
	"AtlHyper/model/envelope"
	"sync"
	"time"
)

// Hub 维护 Master 侧的全局 Envelope 池。
// -----------------------------------------------------------------------------
// - 作用：作为 Master 的唯一存储容器，保存所有上报的 EnvelopeRecord
// - 内部结构：使用 RWMutex + slice 封装，支持并发安全的写入和快照读取
// - 对外能力：
//     * 写入口（Append / AppendBatch / AppendEnvelope / AppendEnvelopeBatch）
//     * 只读拷贝出口（Snapshot / Len）
// - 特点：
//     * 只负责存储和并发保护，不做任何条件过滤/加工逻辑
//     * 外部如果需要按 ClusterID/Source/时间过滤，应在 Snapshot 返回的副本上完成
// -----------------------------------------------------------------------------
type Hub struct {
	mu  sync.RWMutex        // 并发保护锁：写时独占，读时共享
	all []EnvelopeRecord    // 存储池：存放所有 EnvelopeRecord
}

// Global 是全局唯一的 Hub 实例。
// 必须在进程启动时调用 Init() 初始化，否则使用时会 panic。
var Global *Hub

// 初始化全局 Hub 实例
// -----------------------------------------------------------------------------
func Init() {
	Global = &Hub{
		all: make([]EnvelopeRecord, 0, 1024), // 初始容量你可按需调整
	}
}

// 防止未初始化访问
// -----------------------------------------------------------------------------
// ensureInit 用于在每次访问 Hub 前检查是否已初始化。
// 若未初始化，直接 panic 提示调用者必须先 Init()。
func ensureInit() {
	if Global == nil {
		panic("master_store not initialized: call master_store.Init() at process boot")
	}
}

// Append 写入口（单条）。
// -----------------------------------------------------------------------------
// - 功能：将一条 EnvelopeRecord 写入全局池
// - 并发：内部使用写锁，保证 append 操作安全
// - 注意：调用方需保证 rec 已构造完毕（含 EnqueuedAt 等元信息）
func Append(rec EnvelopeRecord) {
	ensureInit()
	Global.mu.Lock()
	Global.all = append(Global.all, rec)
	Global.mu.Unlock()
}

// AppendBatch 写入口（批量）。
// -----------------------------------------------------------------------------
// - 功能：批量写入多条 EnvelopeRecord，一次加锁减少锁竞争
// - 并发：内部使用写锁；整个批次写入过程为原子操作
// - 注意：如果 recs 为空，直接返回
func AppendBatch(recs []EnvelopeRecord) {
	if len(recs) == 0 {
		return
	}
	ensureInit()
	Global.mu.Lock()
	Global.all = append(Global.all, recs...)
	Global.mu.Unlock()
}

// AppendEnvelope 写入口（单条，上报壳）。
// -----------------------------------------------------------------------------
// - 功能：直接接受上报的 Envelope，内部转换成 EnvelopeRecord 再写入
// - 使用场景：接收器 HTTP Handler 中最常用的一行写入
// - 注意：内部调用 Append，锁仍在 Append 内部处理
func AppendEnvelope(env envelope.Envelope) {
	ensureInit()
	Append(NewRecordFromStd(env))
}


// ReplaceLatest 原子“删同源同集群 + 写新”。
// -----------------------------------------------------------------------------
// 适用场景：pod_list_snapshot / node_list_snapshot 等“仅保留每集群最新一条”的来源。
// 并发安全：整段在同一把写锁内完成，避免“先删再写”两步在并发下产生竞态。
// 注意：不能在这里调用 AppendEnvelope/Append（会二次加锁导致死锁）。
// 复杂度：O(N)，N 为当前池中记录数；考虑到这些来源每次只保留 1 条，整体可控。
func ReplaceLatest(env envelope.Envelope) int {
    ensureInit()

    rec := NewRecordFromStd(env) // ✅ 与 AppendEnvelope 同步的转换路径

    Global.mu.Lock()
    defer Global.mu.Unlock()

    // 先删旧
    deleted := 0
    dst := Global.all[:0]
    for _, r := range Global.all {
        if r.Source == rec.Source && r.ClusterID == rec.ClusterID {
            deleted++
            continue
        }
        dst = append(dst, r)
    }
    // 收缩一份新切片，避免保留已删元素的底层数组引用
    Global.all = append([]EnvelopeRecord(nil), dst...)

    // 再写新
    Global.all = append(Global.all, rec)
    return deleted
}


// AppendEnvelopeBatch 写入口（批量，上报壳）。
// -----------------------------------------------------------------------------
// - 功能：直接接受一批 Envelope，上报时自动转换为 EnvelopeRecord 并批量写入
// - 特点：同一批次的记录共用一个 EnqueuedAt（Master 入池时刻）
// - 并发：内部最终调用 AppendBatch，一次加锁
func AppendEnvelopeBatch(envs []envelope.Envelope) {
	if len(envs) == 0 {
		return
	}
	ensureInit()

	recs := make([]EnvelopeRecord, 0, len(envs))
	now := time.Now() // 共用批次的入池时间
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

// Snapshot 读取口（只读副本）。
// -----------------------------------------------------------------------------
// - 功能：返回当前全局池的完整拷贝
// - 并发：使用读锁，可与其他读操作并发，但与写互斥
// - 注意：
//     * 返回的是副本，调用方可以自由修改，不影响真实池子
//     * 任何条件过滤/业务逻辑应基于此副本完成
func Snapshot() []EnvelopeRecord {
	ensureInit()
	Global.mu.RLock()
	defer Global.mu.RUnlock()

	cp := make([]EnvelopeRecord, len(Global.all))
	copy(cp, Global.all)
	return cp
}

// Len 辅助函数：返回当前池内记录数。
// -----------------------------------------------------------------------------
// - 功能：获取当前全局池的大小
// - 并发：使用读锁，开销低
// - 常用于：监控/调试/快速判断是否有数据
func Len() int {
	ensureInit()
	Global.mu.RLock()
	defer Global.mu.RUnlock()
	return len(Global.all)
}
