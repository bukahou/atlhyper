// =======================================================================================
// 📄 external/bootstrap/slack_dispatcher.go
//
// 💬 Description:
//     启动 Slack 告警调度器。周期性检查是否需要告警并通过 Slack Webhook 发送。
//     行为与 Email 告警完全对称，支持节流机制，避免告警风暴。
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package bootstrap

import (
	"NeuroController/config"
	"NeuroController/external/slack"
	"log"
	"time"
)

// ✅ 启动 Slack 告警调度器（建议在控制器启动时调用）
//
// 行为：每隔 AlertDispatchInterval 周期性调用 DispatchSlackAlertFromCleanedEvents
func StartSlackDispatcher() {
	if !config.GlobalConfig.Slack.EnableSlackAlert {
		log.Println("⚠️ Slack 告警功能已关闭，未启动调度器。")
		return
	}

	interval := config.GlobalConfig.Slack.DispatchInterval

	go func() {
		for {
			slack.DispatchSlackAlertFromCleanedEvents()
			time.Sleep(interval)
		}
	}()
	log.Println("✅ Slack 告警调度器启动成功。")
}
