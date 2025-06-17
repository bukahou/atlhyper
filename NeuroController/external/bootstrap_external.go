// =======================================================================================
// 📄 external/bootstrap/bootstrap_external.go
//
// 🧠 Description:
//     外部模块（如邮件、Slack、Webhook）的统一启动入口。
//     推荐在 controller/main.go 中调用 StartExternalSystems 来初始化外部系统功能。
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
	// StartSlackDispatcher()
	// StartWebhookDispatcher()
	// ...

	log.Println("✅ 所有外部组件启动完成。")
}
