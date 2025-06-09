package diagnosis

import (
	"fmt"
	"time"
)

// 🕒 可配置参数（你也可以放到 config 包）
var (
	CleanInterval = 30 * time.Second // 清理间隔
	WriteInterval = 30 * time.Second // 写入间隔
)

// ✅ 启动诊断模块：日志清理 + 日志写入
func StartDiagnosisSystem() {
	// ✅ 启动日志打印
	fmt.Printf("🧠 正在启动诊断系统...\n")
	fmt.Printf("🧼 日志清理间隔：%v\n", CleanInterval)
	fmt.Printf("📝 日志写入间隔：%v\n", WriteInterval)

	// 启动清理器（保鲜 + 去重）
	StartCleanerLoop(CleanInterval)

	// 启动日志写入器（去重写入日志）
	go func() {
		for {
			WriteNewCleanedEventsToFile()
			time.Sleep(WriteInterval)
		}
	}()

	fmt.Println("✅ 诊断系统已启动完成。")
}
