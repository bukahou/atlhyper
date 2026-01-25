// atlhyper_master_v2/notifier/manager/buffer.go
// 告警聚合缓冲
package manager

import (
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/notifier"
)

// aggregateBuffer 聚合缓冲区
// 收集告警，在以下情况触发 flush:
//   - 达到时间窗口 (window)
//   - 达到最大容量 (maxSize)
//   - 收到 Critical 级别告警
type aggregateBuffer struct {
	alerts  []*notifier.Alert
	window  time.Duration
	maxSize int
	timer   *time.Timer
	flushFn func([]*notifier.Alert)
	mu      sync.Mutex
	stopped bool
}

// newAggregateBuffer 创建聚合缓冲区
func newAggregateBuffer(window time.Duration, maxSize int, flushFn func([]*notifier.Alert)) *aggregateBuffer {
	return &aggregateBuffer{
		alerts:  make([]*notifier.Alert, 0),
		window:  window,
		maxSize: maxSize,
		flushFn: flushFn,
	}
}

// Add 添加告警到缓冲区
// 返回 true 表示触发了立即 flush
func (b *aggregateBuffer) Add(alert *notifier.Alert) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.stopped {
		return false
	}

	b.alerts = append(b.alerts, alert)

	// Critical 或 满容量: 立即 flush
	if alert.Severity == notifier.SeverityCritical || len(b.alerts) >= b.maxSize {
		b.flushLocked()
		return true
	}

	// 首个告警启动定时器
	if len(b.alerts) == 1 {
		b.timer = time.AfterFunc(b.window, func() {
			b.FlushNow()
		})
	}

	return false
}

// FlushNow 立即 flush
func (b *aggregateBuffer) FlushNow() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushLocked()
}

// flushLocked flush（需要在锁内调用）
func (b *aggregateBuffer) flushLocked() {
	if len(b.alerts) == 0 {
		return
	}

	// 停止定时器
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	// 提取告警
	alerts := b.alerts
	b.alerts = make([]*notifier.Alert, 0)

	// 异步执行 flush 回调（避免死锁）
	go b.flushFn(alerts)
}

// Stop 停止缓冲区，flush 剩余告警
func (b *aggregateBuffer) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stopped = true

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	// flush 剩余
	if len(b.alerts) > 0 {
		alerts := b.alerts
		b.alerts = nil
		go b.flushFn(alerts)
	}
}

// Count 返回当前缓冲数量（用于调试）
func (b *aggregateBuffer) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.alerts)
}
