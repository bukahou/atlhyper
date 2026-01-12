package mailer

import (
	"AtlHyper/atlhyper_master/gateway/integration/alert"
	"log"
	"time"
)

// DispatchEmailAlertFromCleanedEvents 从清洗后的事件中获取告警并发送邮件
func DispatchEmailAlertFromCleanedEvents(cfg MailConfig) {
	// 从 alert 包获取增量唯一事件聚合结果
	stub := alert.BuildAlertGroupFromEvents()

	// 无告警内容，直接跳过
	if !stub.Display || stub.Data.AlertCount == 0 {
		return
	}

	// 收件人列表从配置中获取
	if len(cfg.To) == 0 {
		log.Println("⚠️ [EmailDispatch] 收件人列表为空，跳过邮件发送")
		return
	}

	// 发送邮件（带节流机制）
	if err := SendAlertEmailWithThrottle(cfg, stub.Title, stub.Data, time.Now()); err != nil {
		log.Printf("❌ [EmailDispatch] 邮件发送失败（%s）: %v\n", stub.Title, err)
	} else {
		log.Printf("📬 [EmailDispatch] 邮件已发送，标题: %q，收件人: %v\n", stub.Title, cfg.To)
	}
}
