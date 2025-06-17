// =======================================================================================
// 📄 alerter/email_throttle.go
//
// ✉️ Description:
//     提供带有节流机制的邮件告警功能，防止在短时间内重复发送相似的邮件。
//     对外只暴露 SendAlertEmailWithThrottle 接口，确保统一管理告警邮件的发送频率。
//
// ⚙️ Features:
//     - 节流间隔配置为 1 小时（throttleInterval）
//     - 线程安全地记录和检查上一次发送邮件的时间
//     - 日志记录每次尝试是否成功触发告警
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package mailer

import (
	"NeuroController/config"
	"NeuroController/interfaces"
	"NeuroController/internal/types"
	"log"
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
