package agent_store

import "time"

// StartTTLJanitor 启动一个常驻清理协程，固定频率清理过期节点。
// maxAge<=0 表示无需清理，直接返回不启动协程。
func StartTTLJanitor(maxAge, interval time.Duration) {
	ensureInit()
	if maxAge <= 0 {
		return // 没有 TTL 需求就不启动
	}
	if interval <= 0 {
		interval = time.Minute
	}

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			compactExpired(maxAge)
		}
	}()
}

// compactExpired 为内部清理实现：删除“最新快照时间 < now-maxAge”的节点。
func compactExpired(maxAge time.Duration) {
	threshold := time.Now().Add(-maxAge)

	Global.mu.Lock()
	for node, snap := range Global.data {
		if snap.Timestamp.Before(threshold) {
			delete(Global.data, node)
		}
	}
	Global.mu.Unlock()
}
