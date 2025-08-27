package store

import (
	model "NeuroController/model/metrics"
	"context"
	"time"
)

/* ======================= 便捷调用：包级默认实例 ======================= */

// defaultStore 为包级默认实例；在引导期设置后，可像“查 DB”一样直接调用。
var defaultStore *Store

// InitDefault 在引导期设置默认实例（建议在 bootstrap 调用）。
func InitDefault(s *Store) { defaultStore = s }

// Default 返回当前默认实例（可能为 nil）。
func Default() *Store { return defaultStore }

// GetAllLatestCopy 返回默认实例的“所有节点最新快照”（值拷贝）。
// 若默认实例未初始化，返回空 map。
func GetAllLatestCopy() map[string]model.NodeMetricsSnapshot {
	if defaultStore == nil {
		return map[string]model.NodeMetricsSnapshot{}
	}
	return defaultStore.GetAllLatestCopy()
}

// Put 向默认实例写入一条快照（若未初始化则忽略）。
func Put(node string, snap *model.NodeMetricsSnapshot) {
	if defaultStore != nil {
		defaultStore.Put(node, snap)
	}
}

// StartDefaultJanitor 为默认实例启动清理器（若未初始化则忽略）。
func StartDefaultJanitor(ctx context.Context, interval time.Duration) {
	if defaultStore != nil {
		defaultStore.StartJanitor(ctx, interval)
	}
}


// // GetLatest 返回指定节点的最新一条快照；不存在则返回 nil。
// func (s *Store) GetLatest(node string) *model.NodeMetricsSnapshot {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()
// 	arr := s.data[node]
// 	if len(arr) == 0 {
// 		return nil
// 	}
// 	return arr[len(arr)-1]
// }

// // Range 返回指定时间窗内的快照（闭区间）；若不存在或窗口无数据，返回空切片。
// // 注意：这里做的是线性过滤，数量大时可根据需要改为二分或环形缓冲。
// func (s *Store) Range(node string, since, until time.Time) []*model.NodeMetricsSnapshot {
// 	if !since.Before(until) {
// 		return nil
// 	}
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()

// 	src := s.data[node]
// 	if len(src) == 0 {
// 		return nil
// 	}
// 	out := make([]*model.NodeMetricsSnapshot, 0, len(src))
// 	for _, it := range src {
// 		if it == nil {
// 			continue
// 		}
// 		ts := it.Timestamp
// 		if (ts.Equal(since) || ts.After(since)) && (ts.Equal(until) || ts.Before(until)) {
// 			out = append(out, it)
// 		}
// 	}
// 	return out
// }

// // DumpAll 深拷贝返回所有节点的所有快照（调试/测试用；生产慎用）。
// func (s *Store) DumpAll() map[string][]*model.NodeMetricsSnapshot {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()

// 	out := make(map[string][]*model.NodeMetricsSnapshot, len(s.data))
// 	for node, arr := range s.data {
// 		cp := make([]*model.NodeMetricsSnapshot, len(arr))
// 		copy(cp, arr)
// 		out[node] = cp
// 	}
// 	return out
// }
