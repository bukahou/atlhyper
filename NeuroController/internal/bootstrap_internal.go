// =======================================================================================
// ✨ bootstrap_internal.go
//
// ✨ Description:
//     Unified internal startup sequence for NeuroController.
//     Includes logger initialization, K8s client setup, health checks, and
//     diagnosis subsystem (cleaner + writer).
//
// 🔧 Components Initialized:
//     - Zap structured logger
//     - Kubernetes controller-runtime client
//     - API server health checker
//     - Diagnosis cleaner and writer loop
//
// 📌 Usage:
//     - Call bootstrap.InitInternalSystems() early in main.go
//     - Keeps main.go concise and consistent
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓️ Created: June 2025
// =======================================================================================

package internal

import (
	"NeuroController/internal/bootstrap"
	"log"
)

// StartInternalSystems 启动 NeuroController 内部运行所需的所有基础子系统。
// 包括：
//   - 事件清理器（用于周期性处理原始 Kubernetes 事件）
//   - 日志写入器（将清理后的事件写入持久化日志文件）
//   - 集群健康检查器（周期性探测 API Server 健康状态）
//
// 该函数应在主程序启动时调用，以确保所有后台服务正常运行。
func StartInternalSystems() {
	// 打印启动日志，标记内部系统组件初始化流程开始
	log.Println("🚀 启动内部系统组件 ...")

	// ✅ 启动清理器：周期性清洗并压缩事件日志，形成可判定异常的结构化事件池
	bootstrap.StartCleanSystem()

	// ✅ 启动日志写入器：将处理后的事件写入文件系统，供后续分析或持久化记录
	bootstrap.StartLogWriter()

	// ✅ 启动集群健康检查器：持续检查 Kubernetes API Server 的可用性
	bootstrap.Startclientchecker()

	// 所有子系统完成启动
	log.Println("✅ 所有内部组件启动完成。")
}
