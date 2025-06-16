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
	"NeuroController/internal/utils"
	"fmt"
	"time"
)

// ✅ 启动诊断系统：包括清理器和日志写入器
func StartCleanSystem() {
	interval := config.GlobalConfig.Diagnosis.CleanInterval
	fmt.Printf("✅ [Startup] 清理器启动（周期: %s）\n", interval)

	go func() {
		for {
			diagnosis.CleanAndStoreEvents()
			time.Sleep(interval)
		}
	}()
}

func StartLogWriter() {
	interval := config.GlobalConfig.Diagnosis.WriteInterval
	fmt.Printf("✅ [Startup] 日志写入器启动（周期: %s）\n", interval)

	go func() {
		for {
			diagnosis.WriteNewCleanedEventsToFile()
			time.Sleep(interval)
		}
	}()
}

func Startclientchecker() {
	fmt.Println("✅ [Startup] 启动集群健康检查器")

	cfg := utils.InitK8sClient()
	utils.StartK8sHealthChecker(cfg)
}
