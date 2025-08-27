package client

import (
	"NeuroController/config"
	"NeuroController/external/client/slack"
	"log"
	"time"
)

// =======================================================================================
// ✅ StartSlackDispatcher - 启动 Slack 告警调度器
//
// 📌 用法：
//     - 在控制器启动初始化完成后调用（如 main.go 中）
//     - 周期性调度 DispatchSlackAlertFromCleanedEvents
//     - 调度间隔由 config.GlobalConfig.Slack.DispatchInterval 决定
//
// 🔐 前提条件：
//     - config.GlobalConfig.Slack.EnableSlackAlert 必须为 true
//     - 配置需提前加载完成
//
// 📢 功能说明：
//     - 实现异步定时任务，轮询“清洗后的事件池”，并发送 Slack 告警
//     - 搭配节流机制避免重复发送，适用于轻量级告警渠道
// =======================================================================================
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
