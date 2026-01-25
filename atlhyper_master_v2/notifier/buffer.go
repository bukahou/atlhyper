// atlhyper_master_v2/notifier/buffer.go
// 告警聚合缓冲
package notifier

import (
	"sync"
	"time"
)

// aggregateBuffer 聚合缓冲区
// 收集告警，在窗口时间到达或达到最大容量时 flush
type aggregateBuffer struct {
	alerts   []*Alert
	window   time.Duration    // 聚合窗口时间
	maxSize  int              // 最大缓冲条数
	timer    *time.Timer      // 窗口定时器
	flushFn  func([]*Alert)   // flush 回调
	mu       sync.Mutex
	stopped  bool
}

// newAggregateBuffer 创建聚合缓冲区
func newAggregateBuffer(window time.Duration, maxSize int, flushFn func([]*Alert)) *aggregateBuffer {
	return &aggregateBuffer{
		alerts:  make([]*Alert, 0),
		window:  window,
		maxSize: maxSize,
		flushFn: flushFn,
	}
}

// Add 添加告警到缓冲区
// 返回 true 表示触发了立即 flush（Critical 或缓冲满）
func (b *aggregateBuffer) Add(alert *Alert) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.stopped {
		return false
	}

	b.alerts = append(b.alerts, alert)

	// 首条告警启动定时器
	if len(b.alerts) == 1 {
		b.timer = time.AfterFunc(b.window, b.timerFlush)
	}

	// Critical 或缓冲满 → 立即 flush
	if alert.Severity == SeverityCritical || len(b.alerts) >= b.maxSize {
		if b.timer != nil {
			b.timer.Stop()
			b.timer = nil
		}
		b.flushLocked()
		return true
	}

	return false
}

// FlushNow 立即 flush（外部调用）
func (b *aggregateBuffer) FlushNow() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.flushLocked()
}

// timerFlush 定时器触发的 flush
func (b *aggregateBuffer) timerFlush() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.timer = nil
	b.flushLocked()
}

// flushLocked 执行 flush（必须持有锁）
func (b *aggregateBuffer) flushLocked() {
	if len(b.alerts) == 0 {
		return
	}

	alerts := b.alerts
	b.alerts = make([]*Alert, 0)

	// 异步回调，避免阻塞
	go b.flushFn(alerts)
}

// Stop 停止缓冲区
func (b *aggregateBuffer) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stopped = true
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	// flush 剩余告警
	if len(b.alerts) > 0 {
		alerts := b.alerts
		b.alerts = nil
		go b.flushFn(alerts)
	}
}

// Size 返回当前缓冲区大小（用于调试）
func (b *aggregateBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.alerts)
}
