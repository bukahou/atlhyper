package client

import (
	"AtlHyper/atlhyper_master/client/mailer"
	"AtlHyper/config"
	"log"
	"time"
)

// =======================================================================================
// ✅ StartEmailDispatcher - 启动定时邮件告警调度器
//
// 📌 用法：
//     - 在主控制器初始化完成后调用（如 main.go 中）
//     - 将周期性调用 mailer.DispatchEmailAlertFromCleanedEvents
//     - 以 config 中设定的时间间隔进行告警发送（异步 goroutine）
//
// 🔐 前提条件：
//     - config.GlobalConfig.Mailer.EnableEmailAlert 为 true
//     - 需要在初始化前加载好 config 全局配置
//
// 📬 功能说明：
//     - 实现后台定时任务，轮询“清洗后的事件池”并尝试发送邮件
//     - 配合节流机制避免重复发送
// =======================================================================================
func StartEmailDispatcher() {

	// 🚫 若邮件功能未启用，则直接退出（不启动调度器）
	if !config.GlobalConfig.Mailer.EnableEmailAlert {
		log.Println("⚠️ 邮件告警功能已关闭，未启动调度器。")
		return
	}

	// 🕒 从全局配置中读取邮件发送间隔
	emailInterval := config.GlobalConfig.Diagnosis.AlertDispatchInterval

	// ✅ 启动 goroutine，后台定时执行告警调度逻辑
	go func() {
		for {
			// 🚀 从已清洗事件中触发邮件发送（由 mailer 模块处理逻辑）
			mailer.DispatchEmailAlertFromCleanedEvents()

			// ⏱ 等待指定间隔再执行下一轮（确保间隔一致）
			time.Sleep(emailInterval)
		}
	}()

	log.Println("✅ 邮件调度器启动成功。")
}
