// atlhyper_master_v2/notifier/log_limiter.go
// 重复日志限频：同一件事在一段时间内只记一次，并记下期间压下了多少条
package notifier

import (
	"sync"
	"time"
)

// repeatLogLimiterMaxKeys 记录的键数上限。
// 键是告警标题，标题里带资源名、种类不受控；超过上限时先清掉过期的键，仍超过则整表清空
const repeatLogLimiterMaxKeys = 1024

// repeatLogLimiter 同一个键在一个时间窗内只放行一次。
//
// 与单纯的「限频」不同，它记得期间压下了多少条：下一次放行时一并报出，
// 日志变少了，但「这件事发生了多少次」的信息没有丢。
type repeatLogLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	now     func() time.Time // 可替换的时钟，便于测试
	entries map[string]*repeatLogEntry
}

type repeatLogEntry struct {
	lastAllowed time.Time
	dropped     int // 上次放行之后压下的条数
}

func newRepeatLogLimiter(window time.Duration, now func() time.Time) *repeatLogLimiter {
	return &repeatLogLimiter{window: window, now: now, entries: make(map[string]*repeatLogEntry)}
}

// allow 这一条该不该记。
//
// 放行时 droppedSinceLast 是上次放行之后被压下的条数（随即清零）；不放行时恒为 0。
func (l *repeatLogLimiter) allow(key string) (allowed bool, droppedSinceLast int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.now()
	entry, seen := l.entries[key]
	if seen && current.Sub(entry.lastAllowed) < l.window {
		entry.dropped++
		return false, 0
	}
	if seen {
		droppedSinceLast = entry.dropped
		entry.lastAllowed, entry.dropped = current, 0
		return true, droppedSinceLast
	}

	if len(l.entries) >= repeatLogLimiterMaxKeys {
		l.evict(current)
	}
	l.entries[key] = &repeatLogEntry{lastAllowed: current}
	return true, 0
}

// evict 腾出位置：先清掉时间窗已过的键（它们下次出现本来就会被放行）；
// 全都还在时间窗内时整表清空 —— 宁可多记几条日志，也不让表无限增长
func (l *repeatLogLimiter) evict(current time.Time) {
	for key, entry := range l.entries {
		if current.Sub(entry.lastAllowed) >= l.window {
			delete(l.entries, key)
		}
	}
	if len(l.entries) >= repeatLogLimiterMaxKeys {
		l.entries = make(map[string]*repeatLogEntry)
	}
}
