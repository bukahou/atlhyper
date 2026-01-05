// external/master_store/bootstrap.go
package master_store

import (
	"log"

	"AtlHyper/atlhyper_master/config"
)

// Bootstrap 初始化总入口。
// -----------------------------------------------------------------------------
// - 功能：在进程启动时调用，完成 master_store 的初始化
// - 步骤：
//     1) Init() → 创建全局 Hub（内存池，负责存放 EnvelopeRecord）
//     2) StartTTLJanitor() → 启动后台清理协程，周期性调用 Compact()
// - 配置：从全局配置读取 TTL、MaxItems、CleanupInterval、MetricsTTL
// - 日志：启动完成后打印一条说明日志，包含 TTL、容量、间隔
// - 使用场景：建议在 external.StartExternalSystems() 中调用，
//             确保整个进程生命周期内，全局池持续有清理器运行。
// -----------------------------------------------------------------------------
// 注意：
//   * Bootstrap 必须在任何写入 (Append/AppendEnvelope) 之前调用
// -----------------------------------------------------------------------------
func Bootstrap() {
	// 1) 初始化全局 Hub
	Init()

	// 2) 从配置获取参数
	ttl := config.GlobalConfig.Store.TTL
	maxItems := config.GlobalConfig.Store.MaxItems
	interval := config.GlobalConfig.Store.CleanupInterval
	metricsTTL := config.GlobalConfig.Store.MetricsTTL

	// 3) 启动定期清理（TTL + 容量）
	StartTTLJanitor(ttl, maxItems, interval, metricsTTL)

	log.Printf("master_store bootstrap ok (TTL=%s, cap=%d, interval=%s, metricsTTL=%s)",
		ttl, maxItems, interval, metricsTTL)
}



