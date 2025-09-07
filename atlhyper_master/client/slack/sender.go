package slack

import (
	"AtlHyper/atlhyper_master/client/alert"
	"log"
)



func DispatchSlackAlertFromCleanedEvents(webhook string) {
    // 从 alert 包拿“增量唯一事件”聚合后的结果
    stub := alert.BuildAlertGroupFromEvents() // 返回 m.LightweightAlertStub

    // 无告警内容，直接跳过
    if !stub.Display || stub.Data.AlertCount == 0 {
        return
    }

    if err := SendSlackAlertWithThrottle(webhook, stub.Title, stub.Data); err != nil {
        log.Printf("❌ [SlackDispatch] Slack 发送失败（%s）: %v\n", stub.Title, err)
    } else {
        log.Printf("📬 [SlackDispatch] Slack 告警已发送，标题: %q\n", stub.Title)
    }
}