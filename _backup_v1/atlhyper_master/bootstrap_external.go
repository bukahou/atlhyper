// atlhyper_master/bootstrap_external.go
package external

import (
	"log"

	"AtlHyper/atlhyper_master/config"
	"AtlHyper/atlhyper_master/gateway/integration"
	"AtlHyper/atlhyper_master/repository/eventwriter"
)

// StartOptionalServices 启动可选功能模块
// -----------------------------------------------------------------------------
// 这些服务是非核心功能，可以根据配置开关启用/禁用
// 即使某个服务启动失败，也不应影响主服务运行
// -----------------------------------------------------------------------------
func StartOptionalServices() {
	log.Println("🔧 启动可选功能模块 ...")

	// 邮件告警调度器（根据配置决定是否真正发送）
	if config.GlobalConfig.Mailer.EnableEmailAlert {
		log.Println("  📧 启动邮件告警调度器")
	}
	integration.StartEmailDispatcher()

	// Slack 告警调度器（根据配置决定是否真正发送）
	if config.GlobalConfig.Slack.EnableSlackAlert {
		log.Println("  💬 启动 Slack 告警调度器")
	}
	integration.StartSlackDispatcher()

	// 事件日志写入调度器（DataHub → SQLite 同步）
	log.Println("  📝 启动事件日志写入调度器")
	eventwriter.StartLogWriterScheduler()

	log.Println("✅ 可选功能模块启动完成")
}
