// =======================================================================================
// 📄 external/slack/sender.go
//
// 📦 Description:
//     Responsible for posting formatted Slack BlockKit messages to the configured
//     Slack Webhook URL using standard HTTP POST with JSON encoding.
//
// 🔌 Responsibilities:
//     - Encode alert payload into JSON
//     - Read Webhook URL (recommended from config or environment variables)
//     - Send HTTP POST request to Slack
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package slack

import (
	"NeuroController/interfaces"
	"log"
)

func DispatchSlackAlertFromCleanedEvents() {

	// ✅ 获取清洗后的事件池
	events := interfaces.GetCleanedEventLogs()
	if len(events) == 0 {
		return
	}

	// ✅ 过滤出新增事件（未发送过的）
	newEvents := filterNewEvents(events)
	if len(newEvents) == 0 {
		// log.Println("🔁 [SlackDispatch] 当前无新增事件，跳过 Slack 发送")
		return
	}

	// ✅ 格式化为轻量级告警数据
	shouldAlert, subject, data := interfaces.GetLightweightAlertGroup(newEvents)
	if !shouldAlert {
		log.Println("✅ [SlackDispatch] 当前无异常事件，未触发 Slack 告警。")
		return
	}

	// ✅ 构建 BlockKit 并节流发送
	err := SendSlackAlertWithThrottle(subject, data)
	if err != nil {
		log.Printf("❌ [SlackDispatch] Slack 发送失败: %v\n", err)
	} else {
		log.Printf("📬 [SlackDispatch] Slack 告警已发送，标题: \"%s\"\n", subject)
	}
}
