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
	"fmt"
)

func StartInternalSystems() {
	fmt.Println("🚀 启动内部系统组件 ...")

	// ✅ 启动邮件调度器
	bootstrap.StartCleanSystem()
	bootstrap.StartLogWriter()

	bootstrap.Startclientchecker()

	fmt.Println("✅ 所有内部组件启动完成。")
}
