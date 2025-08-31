package logger

import (
	"log"
	"time"

	"AtlHyper/config"
)

// =======================================================================================
// ✅ StartLogWriterScheduler - 启动日志写入调度器
//
// 📌 用法：
//     - 在控制器初始化阶段调用，建议在主程序中统一启动所有调度器
//     - 定期调用 WriteNewCleanedEventsToFile 将内存中的新事件写入日志文件
//
// ⚙️ 配置来源：
//     - 写入周期由 config.GlobalConfig.Diagnosis.WriteInterval 控制
//
// 📢 功能说明：
//     - 异步循环执行日志落盘操作，避免阻塞主线程
//     - 支持节流与新事件去重机制配合
// =======================================================================================
func StartLogWriterScheduler() {
	// 🕒 获取日志写入周期配置
	interval := config.GlobalConfig.Diagnosis.WriteInterval

	// 📝 启动提示
	log.Printf("📝 [Logger] 日志写入器启动（周期: %s）", interval)

	// ✅ 启动后台 goroutine 定期执行写入逻辑
	go func() {
		for {
			// 🚀 调用日志写入函数（写入 new 事件至文件）
			WriteNewCleanedEventsToFile()

			// ⏱ 等待下一个周期
			time.Sleep(interval)
		}
	}()
}
