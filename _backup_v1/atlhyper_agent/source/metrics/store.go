// source/metrics/store.go
// 指标数据池
package metrics

import (
	"log"
	"sync"
	"time"

	"AtlHyper/model/collect"
)

// ==================== 存储接口定义 ====================

// Store 统一存储接口
type Store interface {
	Put(key string, value any, ttl time.Duration) error
	Get(key string) (any, bool)
	Delete(key string)
	List() []any
	Clear()
	Size() int
	StartCleanup(interval time.Duration)
	Close()
}

// ==================== 内存存储实现 ====================

type item struct {
	value     any
	expiresAt time.Time
}

type memoryStore struct {
	mu         sync.RWMutex
	items      map[string]*item
	defaultTTL time.Duration
	stopCh     chan struct{}
}

// NewMemoryStore 创建内存存储
func NewMemoryStore(defaultTTL time.Duration) Store {
	return &memoryStore{
		items:      make(map[string]*item),
		defaultTTL: defaultTTL,
		stopCh:     make(chan struct{}),
	}
}

func (s *memoryStore) Put(key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = s.defaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	s.mu.Lock()
	s.items[key] = &item{
		value:     value,
		expiresAt: expiresAt,
	}
	s.mu.Unlock()
	return nil
}

func (s *memoryStore) Get(key string) (any, bool) {
	s.mu.RLock()
	it, exists := s.items[key]
	s.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if !it.expiresAt.IsZero() && time.Now().After(it.expiresAt) {
		return nil, false
	}

	return it.value, true
}

func (s *memoryStore) Delete(key string) {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
}

func (s *memoryStore) List() []any {
	now := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]any, 0, len(s.items))
	for _, it := range s.items {
		if !it.expiresAt.IsZero() && now.After(it.expiresAt) {
			continue
		}
		result = append(result, it.value)
	}
	return result
}

func (s *memoryStore) Clear() {
	s.mu.Lock()
	s.items = make(map[string]*item)
	s.mu.Unlock()
}

func (s *memoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

func (s *memoryStore) StartCleanup(interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.cleanup()
			}
		}
	}()

	log.Printf("[metrics/store] 清理协程已启动，间隔: %v", interval)
}

func (s *memoryStore) cleanup() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, it := range s.items {
		if !it.expiresAt.IsZero() && now.After(it.expiresAt) {
			delete(s.items, key)
		}
	}
}

func (s *memoryStore) Close() {
	close(s.stopCh)
	log.Println("[metrics/store] 已关闭")
}

// ==================== 全局实例 ====================

var (
	globalStore Store
	initOnce    sync.Once
)

// Init 初始化全局存储
func Init(defaultTTL time.Duration) Store {
	initOnce.Do(func() {
		globalStore = NewMemoryStore(defaultTTL)
	})
	return globalStore
}

// Get 获取全局存储实例
func GetStore() Store {
	if globalStore == nil {
		panic("[metrics/store] 未初始化，请先调用 Init()")
	}
	return globalStore
}

// ==================== Metrics 专用接口 ====================

// PutMetricsSnapshot 存储指标快照
func PutMetricsSnapshot(snap *collect.NodeMetricsSnapshot) {
	if snap == nil || snap.NodeName == "" {
		return
	}
	GetStore().Put(snap.NodeName, *snap, 0)
}

// GetAllMetricsSnapshots 获取所有指标快照
func GetAllMetricsSnapshots() map[string]collect.NodeMetricsSnapshot {
	items := GetStore().List()
	result := make(map[string]collect.NodeMetricsSnapshot, len(items))

	for _, item := range items {
		if snap, ok := item.(collect.NodeMetricsSnapshot); ok {
			result[snap.NodeName] = snap
		}
	}

	return result
}

// MetricsLen 获取指标数量
func MetricsLen() int {
	return GetStore().Size()
}

// StartMetricsTTLJanitor 启动 TTL 清理
func StartMetricsTTLJanitor(maxAge, interval time.Duration) {
	GetStore().StartCleanup(interval)
}
