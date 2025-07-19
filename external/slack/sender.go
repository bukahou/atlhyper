package slack

import (
	"NeuroController/sync/center/http/commonapi"
	"log"
)

// ✅ 从各 Agent 获取轻量告警，并发送到 Slack（具备节流保护）
//
// 调用流程：
//   1. 从所有 Agent 聚合轻量告警（只用于 Slack 显示）
//   2. 遍历每个 Agent 返回结果（AlertResponse）
//   3. 对于设置了 Display=true 的告警组：构建 Slack BlockKit 并节流发送
func DispatchSlackAlertFromCleanedEvents() {
	// ✅ 获取所有 Agent 的轻量级告警（用于 Slack 展示，不包含严重级别、事件合并等逻辑）
	alertResponses := commonapi.GetLightweightAlertsFromAgents()

	// ✅ 无告警内容，直接跳过发送
	if len(alertResponses) == 0 {
		log.Println("✅ [SlackDispatch] 所有 Agent 均无轻量告警，跳过 Slack 发送")
		return
	}

	// ✅ 遍历每个 Agent 的告警结果
	for _, resp := range alertResponses {
		// ⚠️ Display = false 表示该告警组未达到展示门槛，跳过
		if !resp.Display {
			continue
		}

		// ✅ 发送 Slack 告警（已封装节流逻辑）
		err := SendSlackAlertWithThrottle(resp.Title, resp.Data)
		if err != nil {
			log.Printf("❌ [SlackDispatch] Slack 发送失败（%s）: %v\n", resp.Title, err)
		} else {
			log.Printf("📬 [SlackDispatch] Slack 告警已发送，标题: \"%s\"\n", resp.Title)
		}
	}
}