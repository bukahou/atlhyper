// =======================================================================================
// 📄 diagnosis/diagnosis_init.go
//
// ✨ Description:
//     Entry point for starting the diagnosis system.
//     Initializes and launches both the log cleaner and log writer.
//
// 📦 Responsibilities:
//     - Configure intervals for cleaning and writing logs
//     - Start the cleaner loop (deduplication + retention)
//     - Start the file writer loop (deduplicated persistent logs)
// =======================================================================================

package bootstrap

import (
	"NeuroController/config"
	"NeuroController/internal/diagnosis"
	"NeuroController/internal/logging"
	"NeuroController/internal/monitor"
	"NeuroController/internal/utils"
	"log"
	"time"
)

// StartCleanSystem 启动清理器协程，用于定期清理原始事件并存储至清理池。
// 该任务通过 config 中的 CleanInterval 控制清理周期。
func StartCleanSystem() {
	// 读取清理周期配置
	interval := config.GlobalConfig.Diagnosis.CleanInterval

	// 打印启动日志（带周期信息）
	log.Printf("✅ [Startup] 清理器启动（周期: %s）", interval)

	// 启动一个后台协程，定期调用事件清理逻辑
	go func() {
		for {
			// 调用清理函数：去重、聚合、生成告警候选
			diagnosis.CleanAndStoreEvents()

			// 等待下一周期
			time.Sleep(interval)
		}
	}()
}

// StartLogWriter 启动日志写入器协程，定期将清理后的事件写入本地日志文件。
// 写入周期由 config 中的 WriteInterval 控制。
func StartLogWriter() {
	// 读取写入周期配置
	interval := config.GlobalConfig.Diagnosis.WriteInterval

	// 打印启动日志
	log.Printf("✅ [Startup] 日志写入器启动（周期: %s）", interval)

	// 启动后台协程执行写入逻辑
	go func() {
		for {
			// 执行写入操作，将新事件写入日志文件
			logging.WriteNewCleanedEventsToFile()

			// 等待下一个写入周期
			time.Sleep(interval)
		}
	}()
}

// Startclientchecker 启动 Kubernetes 集群健康检查器。
// 内部通过 API Server /healthz 探针检测集群是否可用。
func Startclientchecker() {
	log.Println("✅ [Startup] 启动集群健康检查器")

	cfg := utils.InitK8sClient()
	interval := config.GlobalConfig.Kubernetes.APIHealthCheckInterval

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// 立即执行一次
		monitor.StartK8sHealthChecker(cfg)

		for range ticker.C {
			monitor.StartK8sHealthChecker(cfg)
		}
	}()
}
