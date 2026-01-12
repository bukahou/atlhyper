// store/memory/cleanup.go
// 内存数据清理
package memory

import (
	"time"
)

// StartTTLJanitor 定时清理器
func StartTTLJanitor(maxAge time.Duration, maxItems int, interval time.Duration, metricsTTL time.Duration) {
	ensureInit()

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			_ = Compact(maxAge, maxItems)
			_ = CompactMetricsSnapshot(metricsTTL)
		}
	}()
}

// Compact 就地清理（TTL + 容量限制）
func Compact(maxAge time.Duration, maxItems int) (removed int) {
	ensureInit()

	if maxAge > 0 {
		removed += pruneOlderThan(time.Now().Add(-maxAge))
	}
	removed += capTo(maxItems)
	return
}

// pruneOlderThan 按时间阈值清理
func pruneOlderThan(cutoff time.Time) (removed int) {
	Global.mu.Lock()
	defer Global.mu.Unlock()

	dst := Global.all[:0]
	for _, r := range Global.all {
		if !r.EnqueuedAt.Before(cutoff) {
			dst = append(dst, r)
		}
	}
	removed = len(Global.all) - len(dst)

	for i := len(dst); i < len(Global.all); i++ {
		Global.all[i] = EnvelopeRecord{}
	}

	Global.all = dst
	return
}

// capTo 按容量限制裁剪
func capTo(max int) (removed int) {
	Global.mu.Lock()
	defer Global.mu.Unlock()

	n := len(Global.all)
	if max <= 0 {
		removed = n
		for i := range Global.all {
			Global.all[i] = EnvelopeRecord{}
		}
		Global.all = Global.all[:0]
		return
	}
	if n <= max {
		return 0
	}

	removed = n - max
	newAll := make([]EnvelopeRecord, max)
	copy(newAll, Global.all[n-max:])

	for i := range Global.all {
		Global.all[i] = EnvelopeRecord{}
	}

	Global.all = newAll
	return
}

// CompactMetricsSnapshot 仅清理 metrics_snapshot
func CompactMetricsSnapshot(maxAge time.Duration) (removed int) {
	if maxAge <= 0 {
		return 0
	}
	ensureInit()

	cutoff := time.Now().Add(-maxAge)

	Global.mu.Lock()
	defer Global.mu.Unlock()

	dst := Global.all[:0]
	for _, r := range Global.all {
		if r.Source == "metrics_snapshot" && r.EnqueuedAt.Before(cutoff) {
			continue
		}
		dst = append(dst, r)
	}
	removed = len(Global.all) - len(dst)

	for i := len(dst); i < len(Global.all); i++ {
		Global.all[i] = EnvelopeRecord{}
	}
	Global.all = dst
	return
}
