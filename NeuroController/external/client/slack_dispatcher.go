// =======================================================================================
// 📄 external/bootstrap/slack_dispatcher.go
//
// 💬 Description:
//     Slack alert dispatcher module. Periodically evaluates cleaned events and sends
//     lightweight alerts to Slack via webhook. Symmetrical in behavior to the email
//     dispatcher and includes throttling to prevent alert storms.
//
// ⚙️ Responsibilities:
//     - Periodically check cleaned alert events
//     - Determine whether Slack alerts should be triggered
//     - Send formatted `AlertGroupData` via Slack Webhook with rate limiting
//
// 🕒 Recommended to be initialized on controller startup.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package client

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
