package client

import (
	"AtlHyper/atlhyper_master/client/slack"
	"AtlHyper/config"
	"log"
	"time"
)
func StartSlackDispatcher() {
	// 🚫 若配置中未启用 Slack 告警功能，则直接退出
	if !config.GlobalConfig.Slack.EnableSlackAlert {
		log.Println("⚠️ Slack 告警功能已关闭，未启动调度器。")
		return
	}

	// 🕒 获取配置中的告警发送间隔
	interval := config.GlobalConfig.Slack.DispatchInterval

	// ✅ 启动后台 goroutine，周期性处理告警调度任务
	go func() {
		for {
			// 🚀 调用 Slack 模块进行告警发送（从清洗后的事件池中）
			slack.DispatchSlackAlertFromCleanedEvents()

			// ⏱ 间隔等待下一轮告警处理
			time.Sleep(interval)
		}
	}()

	log.Println("✅ Slack 告警调度器启动成功。")
}
