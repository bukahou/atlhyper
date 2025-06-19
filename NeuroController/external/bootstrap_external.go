// =======================================================================================
// 📄 external/bootstrap/bootstrap_external.go
//
// 🧠 Description:
//     Unified startup entry point for external modules such as Email, Slack, and Webhook.
//     Recommended to be called from controller/main.go via StartExternalSystems.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package external

import (
	"NeuroController/external/bootstrap"
	"log"
)

// ✅ 启动所有 External 功能模块
func StartExternalSystems() {
	log.Println("🚀 启动外部系统组件 ...")

	// ✅ 启动邮件调度器
	bootstrap.StartEmailDispatcher()

	// ✅ 启动 Slack 调度器
	bootstrap.StartSlackDispatcher()
	// ✅ 其他模块预留位
	// ...

	log.Println("✅ 所有外部组件启动完成。")
}
