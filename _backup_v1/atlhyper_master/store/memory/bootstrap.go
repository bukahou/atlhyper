// store/memory/bootstrap.go
// 内存存储引擎初始化
package memory

import (
	"log"

	"AtlHyper/atlhyper_master/config"
)

// Bootstrap 初始化内存存储引擎
func Bootstrap() {
	Init()

	ttl := config.GlobalConfig.Store.TTL
	maxItems := config.GlobalConfig.Store.MaxItems
	interval := config.GlobalConfig.Store.CleanupInterval
	metricsTTL := config.GlobalConfig.Store.MetricsTTL

	StartTTLJanitor(ttl, maxItems, interval, metricsTTL)

	log.Printf("✅ 内存存储引擎初始化完成 (TTL=%s, cap=%d, interval=%s, metricsTTL=%s)",
		ttl, maxItems, interval, metricsTTL)
}
