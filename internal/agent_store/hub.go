package agent_store

import (
	nmetrics "NeuroController/model/metrics"
)

// Global 是全局唯一实例；通过 Init() 初始化。
var Global *Store

// Init 初始化全局 Store（幂等由上层 Bootstrap 保证）
func Init() {
	Global = &Store{
		data: make(map[string]nmetrics.NodeMetricsSnapshot, 256),
	}
}

func ensureInit() {
	if Global == nil {
		panic("agent_store not initialized: call agent_store.Bootstrap(...) first")
	}
}

// Put 追加/覆盖：把该节点的“最新快照”写入（值拷贝，避免外部修改影响内部）
func Put(node string, snap *nmetrics.NodeMetricsSnapshot) {
	if node == "" || snap == nil {
		return
	}
	ensureInit()
	Global.mu.Lock()
	Global.data[node] = *snap // 值存储：无额外堆分配，简单可靠
	Global.mu.Unlock()
}

// PutSnapshot 便捷写入：直接从快照里的 NodeName 入库
func PutSnapshot(snap *nmetrics.NodeMetricsSnapshot) {
	if snap == nil || snap.NodeName == "" {
		return
	}
	Put(snap.NodeName, snap)
}

// GetAllLatestCopy 返回“所有节点 -> 最新快照”的值拷贝
func GetAllLatestCopy() map[string]nmetrics.NodeMetricsSnapshot {
	ensureInit()
	Global.mu.RLock()
	defer Global.mu.RUnlock()
	out := make(map[string]nmetrics.NodeMetricsSnapshot, len(Global.data))
	for k, v := range Global.data {
		out[k] = v
	}
	return out
}

// Len 返回当前节点数量（即有最新快照的节点数）
func Len() int {
	ensureInit()
	Global.mu.RLock()
	defer Global.mu.RUnlock()
	return len(Global.data)
}
