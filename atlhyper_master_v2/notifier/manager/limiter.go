// atlhyper_master_v2/notifier/manager/limiter.go
// 发送限流器
package manager

import (
	"sync"
	"time"
)

// rateLimiter 限流器
// 基于滑动窗口的限流，限制每分钟发送次数
type rateLimiter struct {
	maxPerMinute int         // 每分钟最大发送数
	sent         []time.Time // 发送记录（时间戳）
	mu           sync.Mutex
}

// newRateLimiter 创建限流器
func newRateLimiter(maxPerMinute int) *rateLimiter {
	return &rateLimiter{
		maxPerMinute: maxPerMinute,
		sent:         make([]time.Time, 0),
	}
}

// Allow 检查是否允许发送
// 返回 true 表示允许，同时记录本次发送
// 返回 false 表示超限，需要等待
func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// 清理窗口外的记录
	r.cleanup(windowStart)

	// 检查是否超限
	if len(r.sent) >= r.maxPerMinute {
		return false
	}

	// 记录本次发送
	r.sent = append(r.sent, now)
	return true
}

// cleanup 清理过期记录
func (r *rateLimiter) cleanup(windowStart time.Time) {
	// 找到第一个在窗口内的记录
	idx := 0
	for idx < len(r.sent) && r.sent[idx].Before(windowStart) {
		idx++
	}

	if idx > 0 {
		r.sent = r.sent[idx:]
	}
}

// WaitTime 返回需要等待的时间
// 如果不需要等待，返回 0
func (r *rateLimiter) WaitTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sent) == 0 {
		return 0
	}

	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// 清理过期记录
	r.cleanup(windowStart)

	if len(r.sent) < r.maxPerMinute {
		return 0
	}

	// 需要等待最早的记录过期
	oldestExpire := r.sent[0].Add(time.Minute)
	wait := oldestExpire.Sub(now)
	if wait < 0 {
		return 0
	}
	return wait
}

// Count 返回当前窗口内的发送次数（用于调试）
func (r *rateLimiter) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	windowStart := time.Now().Add(-time.Minute)
	r.cleanup(windowStart)
	return len(r.sent)
}
