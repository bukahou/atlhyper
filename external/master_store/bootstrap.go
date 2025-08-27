// external/master_store/bootstrap.go
package master_store

import (
	"log"
	"time"
)

var stopJanitor func()

// -----------------------------------------------------------------------------
// 默认参数（可按需调整）
// -----------------------------------------------------------------------------
// - defaultTTL      → 每条记录的默认生存时间（超过该时间会被清理）
// - defaultMaxItems → 全局池最多保留的记录数（超过则裁剪）
// - defaultInterval → 清理任务的执行间隔
// -----------------------------------------------------------------------------
// 说明：这些值只作为默认配置使用，若你未来需要灵活调整，
//       可以改为从配置文件或环境变量读取。
const (
	defaultTTL      = 15 * time.Minute
	defaultMaxItems = 50_000
	defaultInterval = 1 * time.Minute
)

// Bootstrap 初始化总入口。
// -----------------------------------------------------------------------------
// - 功能：在进程启动时调用，完成 master_store 的初始化
// - 步骤：
//     1) Init() → 创建全局 Hub（内存池，负责存放 EnvelopeRecord）
//     2) StartTTLJanitor() → 启动后台清理协程，周期性调用 Compact()
// - 配置：使用默认参数 → TTL=15分钟、最大 5 万条、清理周期 1 分钟
// - 日志：启动完成后打印一条说明日志，包含 TTL、容量、间隔
// - 使用场景：建议在 external.StartExternalSystems() 中调用，
//             确保整个进程生命周期内，全局池持续有清理器运行。
// -----------------------------------------------------------------------------
// 注意：
//   * Bootstrap 必须在任何写入 (Append/AppendEnvelope) 之前调用
//   * stopJanitor 仅保留在此模块内部，正常生产环境无需调用 Shutdown
// -----------------------------------------------------------------------------
func Bootstrap() {
	// 1) 初始化全局 Hub
	Init()

	// 2) 启动定期清理（TTL + 容量）
	stopJanitor = StartTTLJanitor(defaultTTL, defaultMaxItems, defaultInterval)

	log.Printf("master_store bootstrap ok (TTL=%s, cap=%d, interval=%s)",
		defaultTTL, defaultMaxItems, defaultInterval)
}



