// internal/ingest/store/store.go
package store

// import (
// 	"context"
// 	"sync"
// 	"time"

// 	model "NeuroController/model/metrics"
// )

// // Store 维护最近一段时间（Retention）的节点指标快照，仅存内存，不做持久化。
// type Store struct {
// 	mu        sync.RWMutex
// 	data      map[string][]*model.NodeMetricsSnapshot // nodeName -> 按时间追加的快照切片
// 	retention time.Duration                            // 仅保留最近 Retention 内的数据
// }

// // NewStore 创建内存存储；retention<=0 时默认 10 分钟。
// func NewStore(retention time.Duration) *Store {
// 	if retention <= 0 {
// 		retention = 10 * time.Minute
// 	}
// 	return &Store{
// 		data:      make(map[string][]*model.NodeMetricsSnapshot, 256),
// 		retention: retention,
// 	}
// }

// // Put 追加一条快照到指定节点。
// // 说明：假设调用方已保证 snap.Timestamp 合理（为空时应在 handler 处补 now）。
// func (s *Store) Put(node string, snap *model.NodeMetricsSnapshot) {
// 	if node == "" || snap == nil {
// 		return
// 	}
// 	s.mu.Lock()
// 	s.data[node] = append(s.data[node], snap)
// 	s.mu.Unlock()
// }

// // GetAllLatestCopy 返回“所有节点 → 最新快照”的值拷贝（不暴露内部指针）。
// func (s *Store) GetAllLatestCopy() map[string]model.NodeMetricsSnapshot {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()

// 	out := make(map[string]model.NodeMetricsSnapshot, len(s.data))
// 	for node, arr := range s.data {
// 		if n := len(arr); n > 0 && arr[n-1] != nil {
// 			out[node] = *arr[n-1] // 值拷贝
// 		}
// 	}
// 	return out
// }

// // StartJanitor 启动后台清理协程，按 interval 频率清理超过 Retention 的数据。
// // ctx 取消后退出；interval<=0 时默认 1 分钟。
// func (s *Store) StartJanitor(ctx context.Context, interval time.Duration) {
// 	if interval <= 0 {
// 		interval = time.Minute
// 	}
// 	t := time.NewTicker(interval)
// 	go func() {
// 		defer t.Stop()
// 		for {
// 			select {
// 			case <-ctx.Done():
// 				return
// 			case <-t.C:
// 				s.cleanupOnce()
// 			}
// 		}
// 	}()
// }

// // cleanupOnce 立即执行一次清理：删除早于 (now - retention) 的快照。
// // 空节点会被移除，以节省内存。
// func (s *Store) cleanupOnce() {
// 	threshold := time.Now().Add(-s.retention)

// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	for node, arr := range s.data {
// 		if len(arr) == 0 {
// 			delete(s.data, node)
// 			continue
// 		}
// 		// 原地过滤：保留 ts>=threshold 的项
// 		j := 0
// 		for _, snap := range arr {
// 			if snap != nil && snap.Timestamp.After(threshold) {
// 				arr[j] = snap
// 				j++
// 			}
// 		}
// 		if j == 0 {
// 			delete(s.data, node)
// 		} else {
// 			s.data[node] = arr[:j]
// 		}
// 	}
// }

// package store

// import (
// 	"context"
// 	"sync"
// 	"time"

// 	model "NeuroController/model/metrics"
// )

// // Store 维护最近一段时间（Retention）的节点指标快照，仅存内存，不做持久化。
// type Store struct {
// 	mu        sync.RWMutex
// 	data      map[string][]*model.NodeMetricsSnapshot // nodeName -> 按时间追加的快照切片
// 	retention time.Duration                             // 仅保留最近 Retention 内的数据
// }

// // NewStore 创建内存存储；retention<=0 时默认 10 分钟。
// func NewStore(retention time.Duration) *Store {
// 	if retention <= 0 {
// 		retention = 10 * time.Minute
// 	}
// 	return &Store{
// 		data:      make(map[string][]*model.NodeMetricsSnapshot, 256),
// 		retention: retention,
// 	}
// }

// // Put 追加一条快照到指定节点。
// // 说明：假设调用方已保证 snap.Timestamp 合理（为空时应在 handler 处补 now）。
// func (s *Store) Put(node string, snap *model.NodeMetricsSnapshot) {
// 	if node == "" || snap == nil {
// 		return
// 	}
// 	s.mu.Lock()
// 	s.data[node] = append(s.data[node], snap)
// 	s.mu.Unlock()
// }

// // StartJanitor 启动后台清理协程，按 interval 频率清理超过 Retention 的数据。
// // ctx 取消后退出；interval<=0 时默认 1 分钟。
// func (s *Store) StartJanitor(ctx context.Context, interval time.Duration) {
// 	if interval <= 0 {
// 		interval = time.Minute
// 	}
// 	t := time.NewTicker(interval)
// 	go func() {
// 		defer t.Stop()
// 		for {
// 			select {
// 			case <-ctx.Done():
// 				return
// 			case <-t.C:
// 				s.cleanupOnce()
// 			}
// 		}
// 	}()
// }

// // cleanupOnce 立即执行一次清理：删除早于 (now - retention) 的快照。
// // 空节点会被移除，以节省内存。
// func (s *Store) cleanupOnce() {
// 	threshold := time.Now().Add(-s.retention)

// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	for node, arr := range s.data {
// 		if len(arr) == 0 {
// 			delete(s.data, node)
// 			continue
// 		}
// 		// 原地过滤：保留 ts>=threshold 的项
// 		j := 0
// 		for _, snap := range arr {
// 			if snap != nil && snap.Timestamp.After(threshold) {
// 				arr[j] = snap
// 				j++
// 			}
// 		}
// 		if j == 0 {
// 			delete(s.data, node)
// 		} else {
// 			s.data[node] = arr[:j]
// 		}
// 	}
// }