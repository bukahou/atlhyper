package agent_store

import (
	"log"
	"sync"

	"AtlHyper/atlhyper_agent/config"
)

var bootOnce sync.Once

// Bootstrap 使用配置中的 TTL 与清理间隔启动全局 Store 与清理协程。
func Bootstrap() {
	bootOnce.Do(func() {
		Init()
		maxAge := config.GlobalConfig.Store.TTLMaxAge
		interval := config.GlobalConfig.Store.CleanupInterval
		StartTTLJanitor(maxAge, interval)
		log.Printf("agent_store bootstrap ok (TTL=%s, interval=%s)", maxAge, interval)
	})
}
