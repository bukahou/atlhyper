// metrics_store_reader.go
package metrics_store

import (
	"time"

	model "NeuroController/model/metrics"
)

// SnapshotInMemoryMetrics
// ----------------------------------------------------------------------
// 返回当前内存中所有节点的快照（原始引用，非深拷贝）
// 调用方如果需要修改数据，应先拷贝
// ----------------------------------------------------------------------
func SnapshotInMemoryMetrics() map[string][]*model.NodeMetricsSnapshot {
	memMu.Lock()
	defer memMu.Unlock()

	out := make(map[string][]*model.NodeMetricsSnapshot, len(memBuf))
	for node, arr := range memBuf {
		// 直接引用，如果需要避免外部修改，改成 copy
		out[node] = arr
	}
	return out
}

// GetNodeSnapshots
// ----------------------------------------------------------------------
// 获取指定节点的快照数据（原始引用）
// 如果节点不存在，返回 nil
// ----------------------------------------------------------------------
func GetNodeSnapshots(nodeName string) []*model.NodeMetricsSnapshot {
	memMu.Lock()
	defer memMu.Unlock()

	if arr, ok := memBuf[nodeName]; ok {
		return arr
	}
	return nil
}

// GetNodeSnapshotsFiltered
// ----------------------------------------------------------------------
// 获取指定节点的快照数据，并可按时间/条数过滤
// since: 如果非零，则只返回 >= since 的快照
// limit: 如果 >0，则只返回最新 limit 条
// ----------------------------------------------------------------------
func GetNodeSnapshotsFiltered(nodeName string, since time.Time, limit int) []*model.NodeMetricsSnapshot {
	memMu.Lock()
	defer memMu.Unlock()

	arr, ok := memBuf[nodeName]
	if !ok || len(arr) == 0 {
		return nil
	}

	// 按时间过滤
	if !since.IsZero() {
		filtered := make([]*model.NodeMetricsSnapshot, 0, len(arr))
		for _, s := range arr {
			if t, ok := tsAsTime(s.Timestamp); ok && (t.After(since) || t.Equal(since)) {
				filtered = append(filtered, s)
			}
		}
		arr = filtered
	}

	// 限制条数
	if limit > 0 && len(arr) > limit {
		arr = arr[len(arr)-limit:]
	}

	return arr
}
