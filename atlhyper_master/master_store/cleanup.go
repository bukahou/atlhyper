// external/master_store/cleanup.go
package master_store

import (
	"time"
)

// SourcePolicy 定义对某个 Source 的清理策略
type SourcePolicy struct {
	Source   string        // 目标来源，如 "metrics_snapshot"
	MaxAge   time.Duration // 该 Source 的 TTL（<=0 不按时间清理）
	MaxItems int           // 该 Source 的最大条数（<=0 表示清空该 Source）
}



// StartTTLJanitor 定时清理器。
// -----------------------------------------------------------------------------
// - 功能：周期性调用 Compact()，自动清理全局 Store 里的过期或超量数据
// - 参数：
//     * maxAge   → TTL（超过该时长的记录会被清理；<=0 表示不按时间清理）
//     * maxItems → 最大保留条数（超过则裁剪；<=0 表示清空）
//     * interval → 定时清理的间隔
// - 返回：一个 stop() 函数，用于外部在需要时关闭清理协程
// - 并发：内部启动 goroutine，独立运行；调用 stop() 可安全退出
// - 使用场景：Bootstrap() 中初始化后常驻运行

// 外层：确保只启动一次
// var ttlJanitorOnce sync.Once

// StartTTLJanitor 定时清理器（启动后不可停止）。
// - maxAge      → 全局 TTL
// - maxItems    → 全局最大条数
// - interval    → 定时清理间隔
// - metricsTTL  → 专门用于 Source=="metrics_snapshot" 的 TTL
func StartTTLJanitor(maxAge time.Duration, maxItems int, interval time.Duration, metricsTTL time.Duration) {
	ensureInit()

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			// 1) 全局兜底清理
			_ = Compact(maxAge, maxItems)

			// 2) 单独清理 metrics_snapshot
			_ = CompactMetricsSnapshot(metricsTTL)
		}
	}()
}

// Compact 就地清理（TTL + 容量限制）。
// -----------------------------------------------------------------------------
// - 功能：对全局池执行一次清理，返回删除的记录数
// - 规则：
//     * TTL：删除 EnqueuedAt 超过 maxAge 的记录（maxAge <=0 表示不启用）
//     * 容量：只保留最新的 maxItems 条，多余的删除（maxItems <=0 表示清空）
// - 并发：内部使用写锁，保证与 Append/Snapshot 等操作互斥
// - 返回：总共删除的记录数（TTL 删除数 + 容量裁剪数）
// - 使用场景：
//     * 周期性调用（由 StartTTLJanitor 定时触发）
//     * 或者手动在某些关键时刻触发一次
func Compact(maxAge time.Duration, maxItems int) (removed int) {
	ensureInit()

	// 1) TTL 清理
	if maxAge > 0 {
		removed += pruneOlderThan(time.Now().Add(-maxAge))
	}

	// 2) 容量裁剪
	removed += capTo(maxItems)
	return
}

// -----------------------------------------------------------------------------
// 以下为私有实现，仅供 Compact 内部调用
// -----------------------------------------------------------------------------

// pruneOlderThan 按时间阈值清理。
// -----------------------------------------------------------------------------
// - 功能：移除 EnqueuedAt 早于 cutoff 的所有记录
// - 参数：cutoff → 时间阈值（所有 EnqueuedAt < cutoff 的记录会被删除）
// - 返回：删除的条目数
// - 实现：
//     * 申请 dst 切片（复用原始底层数组，避免重新分配）
//     * 遍历 Global.all，把未过期的 append 回去
//     * 将多余位置置空（EnvelopeRecord{}），帮助 GC 回收 Payload
//     * 最终把 Global.all 截断到新长度
func pruneOlderThan(cutoff time.Time) (removed int) {
	Global.mu.Lock()
	defer Global.mu.Unlock()

	dst := Global.all[:0] // 复用底层数组，容量不变
	for _, r := range Global.all {
		if !r.EnqueuedAt.Before(cutoff) {
			dst = append(dst, r)
		}
	}
	removed = len(Global.all) - len(dst)

	// 清理尾部残留，切断引用，帮助 GC 回收内存
	for i := len(dst); i < len(Global.all); i++ {
		Global.all[i] = EnvelopeRecord{}
	}

	Global.all = dst
	return
}

// capTo 按容量限制裁剪。
// -----------------------------------------------------------------------------
// - 功能：保证全局池最多只保留 max 条记录，多余的丢弃
// - 参数：max → 最大允许保留的条目数
// - 返回：删除的条目数
// - 规则：
//     * max <= 0 → 直接清空
//     * n <= max → 不做操作
//     * n > max  → 仅保留最新的 max 条（队尾部分），丢弃最早的
// - 实现：
//     * 通过 copy() 保留后 max 条
//     * 将原有 slice 全部置空，帮助 GC
//     * 替换 Global.all
func capTo(max int) (removed int) {
	Global.mu.Lock()
	defer Global.mu.Unlock()

	n := len(Global.all)
	if max <= 0 {
		// 清空所有
		removed = n
		for i := range Global.all {
			Global.all[i] = EnvelopeRecord{}
		}
		Global.all = Global.all[:0]
		return
	}
	if n <= max {
		// 没超过，不处理
		return 0
	}

	// 超过 max，保留最新的 max 条
	removed = n - max
	newAll := make([]EnvelopeRecord, max)
	copy(newAll, Global.all[n-max:]) // 复制末尾部分

	// 原有数组置空，释放引用
	for i := range Global.all {
		Global.all[i] = EnvelopeRecord{}
	}

	Global.all = newAll
	return
}



// CompactMetricsSnapshot 仅清理 Source=="metrics_snapshot" 且早于 maxAge 的记录。
// - maxAge <= 0 时不做任何事（与 Compact 的 TTL 语义一致）
// - 只做 TTL，不做容量裁剪
func CompactMetricsSnapshot(maxAge time.Duration) (removed int) {
	if maxAge <= 0 {
		return 0
	}
	ensureInit()

	cutoff := time.Now().Add(-maxAge)

	Global.mu.Lock()
	defer Global.mu.Unlock()

	// 复用底层数组，GC 友好
	dst := Global.all[:0]
	for _, r := range Global.all {
		// 丢弃：metrics_snapshot 且过期
		if r.Source == "metrics_snapshot" && r.EnqueuedAt.Before(cutoff) {
			continue
		}
		dst = append(dst, r)
	}
	removed = len(Global.all) - len(dst)

	// 断尾清零，帮助 GC
	for i := len(dst); i < len(Global.all); i++ {
		Global.all[i] = EnvelopeRecord{}
	}
	Global.all = dst
	return
}






// func StartTTLJanitor(maxAge time.Duration, maxItems int, interval time.Duration) (stop func()) {
// 	ensureInit()

// 	stopCh := make(chan struct{})
// 	go func() {
// 		t := time.NewTicker(interval)
// 		defer t.Stop()
// 		for {
// 			select {
// 			case <-t.C:
// 				// 周期性执行清理
// 				_ = Compact(maxAge, maxItems) // 删除数可忽略或打印日志
// 			case <-stopCh:
// 				// 收到停止信号，结束 goroutine
// 				return
// 			}
// 		}
// 	}()
// 	return func() { close(stopCh) }
// }

