package mailer

import (
	"log"
	"time"

	"AtlHyper/atlhyper_master/config"
	"AtlHyper/model/integration" // ✅ 需要这个
)

// ===================================================================================
// ✅ DispatchEmailAlertFromCleanedEvents - 汇总 Agent 告警并发送邮件（支持节流）
// ===================================================================================
func DispatchEmailAlertFromCleanedEvents() {
	// 🚧 临时空占位，不依赖 commonapi
	type EmailAlertStub struct {
		Title string
		Alert bool
		Data  integration.AlertGroupData // ✅ 和 SendAlertEmailWithThrottle 的参数一致
	}
	alertResponses := []EmailAlertStub{} // 空切片

	// ✅ 筛选出需要发送邮件的告警项
	validAlerts := make([]EmailAlertStub, 0)
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
		if err := SendAlertEmailWithThrottle(recipients, resp.Title, resp.Data, time.Now()); err != nil {
			log.Printf("❌ [AgentMail] 邮件发送失败（%s）: %v", resp.Title, err)
		} else {
			log.Printf("📬 [AgentMail] 邮件已发送，标题: \"%s\"，收件人: %v", resp.Title, recipients)
		}
	}
}
