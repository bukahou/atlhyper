// =======================================================================================
// 📄 external/bootstrap/email_dispatcher.go
//
// 📬 Description:
//     Core email alert dispatcher module. Periodically invoked by the diagnosis system,
//     it processes cleaned events, evaluates alert conditions, and sends email notifications
//     through a throttled mailer mechanism.
//
// ⚙️ Responsibilities:
//     - Fetch cleaned events from the diagnosis system
//     - Evaluate alert triggers via the `alerter` module
//     - Format and send `AlertGroupData` using the `mailer`, with throttling support
//
// 🕒 Recommended to be scheduled periodically by the cleaner or on controller startup.
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
