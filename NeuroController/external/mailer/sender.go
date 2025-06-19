// =======================================================================================
// 📄 external/mailer/dispatch.go
//
// 📬 Description:
//     Main dispatcher for email alerts. This function is periodically invoked by the
//     email scheduler to evaluate cleaned events and decide whether an email alert
//     should be sent, with built-in recipient validation and throttling logic.
//
// ⚙️ Responsibilities:
//     - Retrieve cleaned events from the diagnosis interface
//     - Evaluate whether conditions for sending alerts are met
//     - Format alert content and determine recipients
//     - Trigger throttled email delivery via SendAlertEmailWithThrottle
//
// 📣 Recommended to be called on a regular interval by the scheduler.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package mailer

import (
	"NeuroController/config"
	"NeuroController/interfaces"
	"log"
	"time"
)

func DispatchEmailAlertFromCleanedEvents() {

	// ✅ 获取清洗后的事件池
	events := interfaces.GetCleanedEventLogs()
	if len(events) == 0 {
		return
	}

	// ✅ 判断是否触发告警并格式化数据
	shouldAlert, subject, data := interfaces.ComposeAlertGroupIfNecessary(events)
	if !shouldAlert {
		return
	}

	// ✅ 获取收件人
	recipients := config.GlobalConfig.Mailer.To
	if len(recipients) == 0 {
		log.Println("⚠️ [EmailDispatch] 收件人列表为空，跳过邮件发送。")
		return
	}

	// ✅ 执行节流判断并发送
	err := SendAlertEmailWithThrottle(recipients, subject, data, time.Now())
	if err != nil {
		log.Printf("❌ [EmailDispatch] 邮件发送失败: %v", err)
	} else {
		log.Printf("📬 [EmailDispatch] 邮件已发送，标题: \"%s\"，收件人: %v", subject, recipients)
	}
}
