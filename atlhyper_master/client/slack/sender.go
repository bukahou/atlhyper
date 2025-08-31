package slack

import (
	"log"

	"AtlHyper/model" // ✅ 注意要引入 model 包
)

// 🚧 TODO: 替换回 commonapi.LightweightAlert
type LightweightAlertStub struct {
	Title   string
	Data    model.AlertGroupData
	Display bool
}

func DispatchSlackAlertFromCleanedEvents() {
	// 临时空占位
	alertResponses := []LightweightAlertStub{}

	// ✅ 无告警内容，直接跳过发送
	if len(alertResponses) == 0 {
		log.Println("✅ [SlackDispatch] 暂无轻量告警，跳过 Slack 发送")
		return
	}

	// ✅ 遍历每个 Agent 的告警结果
	for _, resp := range alertResponses {
		if !resp.Display {
			continue
		}

		// ✅ Data 类型与函数签名一致（model.AlertGroupData）
		err := SendSlackAlertWithThrottle(resp.Title, resp.Data)
		if err != nil {
			log.Printf("❌ [SlackDispatch] Slack 发送失败（%s）: %v\n", resp.Title, err)
		} else {
			log.Printf("📬 [SlackDispatch] Slack 告警已发送，标题: \"%s\"\n", resp.Title)
		}
	}
}
