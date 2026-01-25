// atlhyper_master_v2/notifier/dedup.go
// 告警去重缓存
package notifier

import (
	"sync"
	"time"
)

// dedupCache 去重缓存
// 相同 Key 的告警在 TTL 内只发送一次
type dedupCache struct {
	cache map[string]time.Time // key -> 首次记录时间
	ttl   time.Duration
	mu    sync.Mutex
}

// newDedupCache 创建去重缓存
func newDedupCache(ttl time.Duration) *dedupCache {
	d := &dedupCache{
		cache: make(map[string]time.Time),
		ttl:   ttl,
	}
	// 启动定期清理
	go d.cleanupLoop()
	return d
}

// IsDuplicate 检查是否重复
// 如果是新 Key 或已过期，返回 false 并记录
// 如果在 TTL 内，返回 true（重复）
func (d *dedupCache) IsDuplicate(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	if t, exists := d.cache[key]; exists {
		if now.Sub(t) < d.ttl {
			// 在 TTL 内，是重复
			return true
		}
		// 已过期，更新时间
	}

	// 记录新时间
	d.cache[key] = now
	return false
}

// cleanup 清理过期条目
func (d *dedupCache) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for key, t := range d.cache {
		if now.Sub(t) >= d.ttl {
			delete(d.cache, key)
		}
	}
}

// cleanupLoop 定期清理循环
func (d *dedupCache) cleanupLoop() {
	ticker := time.NewTicker(d.ttl / 2) // 每半个 TTL 清理一次
	for range ticker.C {
		d.cleanup()
	}
}

// Size 返回缓存大小（用于调试）
func (d *dedupCache) Size() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.cache)
}
