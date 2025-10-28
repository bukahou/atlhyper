//atlhyper_master/bootstrap_external.go

package external

import (
	"AtlHyper/atlhyper_master/client"
	"AtlHyper/atlhyper_master/logger"
	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/atlhyper_master/server"
	"log"
)

// ✅ 启动所有 External 功能模块
func StartExternalSystems() {
	log.Println("🚀 启动Master系统组件 ...")

	//    必须在任何 Append/读取/调度器启动之前
	master_store.Bootstrap()

	// ✅ 启动邮件调度器
	client.StartEmailDispatcher()

	// ✅ 启动 Slack 调度器
	client.StartSlackDispatcher()

		// ✅ 启动日志写入调度器（新增）
	logger.StartLogWriterScheduler()

	// go metrics_store.StartMetricsSync()

	log.Println("🌐 启动统一 HTTP Server（UI API + Webhook）")
	server.StartHTTPServer()

	log.Println("✅ 所有Master组件启动完成。")
}
