// atlhyper_master_v2/notifier/manager/dedup.go
// 告警去重缓存
package manager

import (
	"sync"
	"time"
)

// dedupCache 去重缓存
// 基于 TTL 的去重，相同 key 在 TTL 内只处理一次
type dedupCache struct {
	cache map[string]time.Time
	ttl   time.Duration
	mu    sync.Mutex
	stopC chan struct{}
}

// newDedupCache 创建去重缓存
func newDedupCache(ttl time.Duration) *dedupCache {
	d := &dedupCache{
		cache: make(map[string]time.Time),
		ttl:   ttl,
		stopC: make(chan struct{}),
	}
	go d.cleanupLoop()
	return d
}

// IsDuplicate 检查是否重复
// 返回 true 表示重复（已存在且未过期），返回 false 表示首次或已过期
// 非重复时会自动记录 key
func (d *dedupCache) IsDuplicate(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// 检查是否存在且未过期
	if expireAt, ok := d.cache[key]; ok {
		if now.Before(expireAt) {
			return true // 重复
		}
	}

	// 记录新 key
	d.cache[key] = now.Add(d.ttl)
	return false
}

// cleanup 清理过期记录
func (d *dedupCache) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for key, expireAt := range d.cache {
		if now.After(expireAt) {
			delete(d.cache, key)
		}
	}
}

// cleanupLoop 定期清理
func (d *dedupCache) cleanupLoop() {
	ticker := time.NewTicker(d.ttl / 2) // 清理间隔为 TTL 的一半
	defer ticker.Stop()

	for {
		select {
		case <-d.stopC:
			return
		case <-ticker.C:
			d.cleanup()
		}
	}
}

// Stop 停止清理循环
func (d *dedupCache) Stop() {
	close(d.stopC)
}

// Count 返回当前缓存数量（用于调试）
func (d *dedupCache) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.cache)
}
