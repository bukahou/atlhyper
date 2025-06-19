// =======================================================================================
// 📄 alerter/email_throttle.go
//
// ✉️ Description:
//     Provides a throttled email alerting mechanism to prevent duplicate notifications
//     within short time intervals. Only exposes the unified interface
//     SendAlertEmailWithThrottle for controlled email delivery.
//
// ⚙️ Features:
//     - Throttle interval set to 1 hour (throttleInterval)
//     - Thread-safe tracking of last email send time
//     - Logs each invocation to indicate whether alert was triggered or skipped
//
// 📣 Use this as the only entry point for sending email alerts from external modules.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package mailer

import (
	"NeuroController/internal/types"
	"sync"
	"time"
)

// 🧠 记录上次发送邮件时间的全局状态和互斥锁，确保并发安全
var (
	lastEmailSentTimeMu sync.Mutex // 锁定访问 lastEmailSentTime
	lastEmailSentTime   time.Time  // 上次成功发送告警邮件的时间
)

// ⏲️ 节流时间间隔（每小时最多发送一次告警邮件）
const throttleInterval = 1 * time.Hour

// ✅ 外部统一调用的邮件发送函数，自动判断节流条件
//
// 如果距离上一次邮件发送时间小于 throttleInterval，邮件将不会发送；
// 否则会记录本次发送时间并调用实际邮件发送逻辑。
//
// 参数：
//   - to: 收件人地址列表
//   - subject: 邮件标题
//   - data: 告警数据（将用于填充邮件模板）
//   - eventTime: 触发告警的事件时间
//
// 返回：
//   - error: 若邮件发送失败则返回错误，否则为 nil
func SendAlertEmailWithThrottle(to []string, subject string, data types.AlertGroupData, eventTime time.Time) error {
	lastEmailSentTimeMu.Lock()
	defer lastEmailSentTimeMu.Unlock()

	//  若处于节流时间范围内，跳过邮件发送
	if !lastEmailSentTime.IsZero() && time.Since(lastEmailSentTime) < throttleInterval {

		return nil
	}

	// ✅ 满足发送条件：更新发送时间并实际发送邮件
	lastEmailSentTime = time.Now()

	return SendAlertEmail(to, subject, data)
}
