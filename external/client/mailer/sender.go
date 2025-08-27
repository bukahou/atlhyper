package mailer

import (
	"NeuroController/config"
	"NeuroController/sync/center/http/commonapi"
	"log"
	"time"
)

// ===================================================================================
// ✅ DispatchEmailAlertFromCleanedEvents - 汇总 Agent 告警并发送邮件（支持节流）
//
// 🔁 调用场景：被周期性调度器（client/email_dispatcher.go）调用
//
// 核心逻辑：
//   1. 向所有 Agent 请求告警数据
//   2. 提取需要发送邮件的内容（resp.Alert == true）
//   3. 获取全局收件人列表
//   4. 对每个有效告警，逐条发送邮件（带节流控制）
// ===================================================================================
func DispatchEmailAlertFromCleanedEvents() {
	// ✅ 向所有 Agent 请求告警数据（返回含 Title + Alert + Data）
	alertResponses := commonapi.GetAlertGroupFromAgents()

	// ✅ 筛选出返回值中需要发送邮件的告警项（Alert = true）
	validAlerts := make([]commonapi.AlertResponse, 0)
	for _, resp := range alertResponses {
		if resp.Alert {
			validAlerts = append(validAlerts, resp)
		}
	}

	// ✅ 如果没有任何告警项，打印日志并退出
	if len(validAlerts) == 0 {
		log.Println("✅ [AgentMail] 所有 Agent 均无告警，跳过发送")
		return
	}

	// ✅ 获取收件人列表（从全局配置读取）
	recipients := config.GlobalConfig.Mailer.To
	if len(recipients) == 0 {
		log.Println("⚠️ [AgentMail] 收件人列表为空，跳过邮件发送。")
		return
	}

	// ✅ 遍历每条告警，调用 SendAlertEmailWithThrottle 发送邮件（带节流机制）
	for _, resp := range validAlerts {
		err := SendAlertEmailWithThrottle(recipients, resp.Title, resp.Data, time.Now())
		if err != nil {
			log.Printf("❌ [AgentMail] 邮件发送失败（%s）: %v", resp.Title, err)
		} else {
			log.Printf("📬 [AgentMail] 邮件已发送，标题: \"%s\"，收件人: %v", resp.Title, recipients)
		}
	}
}
