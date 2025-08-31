package mailer

import (
	"AtlHyper/model"
	"sync"
	"time"
)

// ===================================================================================
// 🧠 节流控制机制 - 防止邮件频繁发送
//
// 使用互斥锁和时间记录，确保任意时间段内只发送一次邮件。
// 可避免因短时间内重复触发告警而导致邮件轰炸。
// ===================================================================================

// 🧠 全局互斥锁和记录变量（必须并发安全）
var (
	lastEmailSentTimeMu sync.Mutex // ✅ 用于保护 lastEmailSentTime 的并发访问
	lastEmailSentTime   time.Time  // ✅ 上一次发送邮件的时间戳
)

// ⏱ 节流时间间隔（设置为 1 小时）
//     - 作用：若距离上次发送不足 1 小时，将跳过邮件发送
const throttleInterval = 1 * time.Hour

// ===================================================================================
// ✅ SendAlertEmailWithThrottle - 节流判断后发送告警邮件
//
// 外部统一调用此函数发送邮件，会自动判断是否满足节流条件。
//     - 若处于冷却期内：直接跳过，不发送
//     - 若超出冷却期：调用 SendAlertEmail 真正发送邮件，并记录时间
//
// 参数：
//     - to         收件人列表（如 ["admin@example.com"]）
//     - subject    邮件标题（如 "节点异常告警"）
//     - data       告警内容结构体，将用于渲染 HTML 模板
//     - eventTime  告警触发事件的时间（暂未用于逻辑判断）
//
// 返回：
//     - error      若邮件发送失败则返回错误，否则为 nil
// ===================================================================================
func SendAlertEmailWithThrottle(to []string, subject string, data model.AlertGroupData, eventTime time.Time) error {
	// ✅ 加锁，确保多协程下不会重复发送
	lastEmailSentTimeMu.Lock()
	defer lastEmailSentTimeMu.Unlock()

	// ❌ 若上次发送时间非零，且距离当前不足 throttleInterval，则跳过
	if !lastEmailSentTime.IsZero() && time.Since(lastEmailSentTime) < throttleInterval {
		return nil
	}

	// ✅ 满足节流条件：更新记录时间，并发送邮件
	lastEmailSentTime = time.Now()
	return SendAlertEmail(to, subject, data)
}
