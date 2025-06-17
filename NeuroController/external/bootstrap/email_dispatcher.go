// =======================================================================================
// 📄 external/bootstrap/email_dispatcher.go
//
// 📬 Description:
//     核心邮件告警调度器。该模块由诊断系统周期性调用，统一处理清洗后的告警事件。
//     若事件满足触发条件，则构造邮件并通过节流控制器发送邮件告警。
//
// ⚙️ Responsibilities:
//     - 从 diagnosis 获取已清洗事件
//     - 调用 alerter 模块判断是否触发告警
//     - 构造 AlertGroupData 并通过 mailer 发送（支持节流）
//
// 📣 推荐由清理器模块周期性调度调用
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package bootstrap

import (
	"NeuroController/config"
	"NeuroController/external/mailer"
	"log"
	"time"
)

// ✅ 启动定时邮件告警调度器（推荐在控制器启动时调用）
//
// 行为：每隔 EmailInterval 周期性调用 DispatchEmailAlertFromCleanedEvents
func StartEmailDispatcher() {

	if !config.GlobalConfig.Mailer.EnableEmailAlert {
		log.Println("⚠️ 邮件告警功能已关闭，未启动调度器。")
		return
	}
	emailInterval := config.GlobalConfig.Diagnosis.AlertDispatchInterval

	// ✅ 启动异步循环
	go func() {
		for {
			mailer.DispatchEmailAlertFromCleanedEvents()
			time.Sleep(emailInterval)
		}
	}()
	log.Println("✅ 邮件调度器启动成功。")
}
