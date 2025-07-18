package slack

import (
	"NeuroController/internal/types"
	"NeuroController/model"
	"fmt"
	"sync"
	"time"
)

var (
	// 🧠 上次 Slack 告警发送时间的互斥锁与时间戳（确保并发安全）
	lastSlackSentTimeMu sync.Mutex
	lastSlackSentTime   time.Time
)

// ⏱️ Slack 节流时间间隔（每 N 分钟只允许发送一次）
const slackThrottleInterval = 1 * time.Minute

// =======================================================================================
// ✅ SendSlackAlertWithThrottle: Slack 告警发送函数（带节流控制）
//
// 功能：
//   - 每次调用判断距离上次发送是否已超过 throttleInterval
//   - 若未超过：跳过发送；否则构造并发送 BlockKit 消息
//
// 输入：
//   - subject: 告警标题（用于 Slack header）
//   - data: AlertGroupData（包含资源列表、事件列表等）
//
// 返回：
//   - error: 若发送失败则返回错误；若节流跳过则返回 nil
// =======================================================================================
func SendSlackAlertWithThrottle(subject string, data types.AlertGroupData) error {
	lastSlackSentTimeMu.Lock()
	defer lastSlackSentTimeMu.Unlock()

	// ⏳ 若当前仍处于节流窗口内，则跳过发送
	if !lastSlackSentTime.IsZero() && time.Since(lastSlackSentTime) < slackThrottleInterval {
		fmt.Println("⏳ [SlackThrottle] 距离上次发送过短，跳过本轮 Slack 发送。")
		return nil
	}

	// ✅ 通过节流校验：更新发送时间，开始发送
	lastSlackSentTime = time.Now()

	// 构造 BlockKit 消息体并发送
	payload := BuildSlackBlockFromAlert(data, subject)
	return SendSlackAlert(payload)
}

// =======================================================================================
// 🧠 内部状态：已发送事件缓存（防止重复发送同一条告警）
// =======================================================================================

var (
	// sentEvents 保存已发送事件的唯一标识（加锁保证线程安全）
	sentEventsMu sync.Mutex
	sentEvents   = make(map[string]time.Time) // key: 唯一事件标识, value: 首次发送时间
)

// ⏳ 事件缓存有效期（同一事件仅在 10 分钟外才可再次发送）
const sentEventTTL = 10 * time.Minute

// ✅ eventKey: 生成事件的唯一标识字符串
//
// 用于判断事件是否已发送，字段组合应能唯一描述一次异常
func eventKey(ev model.LogEvent) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		ev.Kind,            // 资源类型（Pod, Deployment 等）
		ev.Namespace,       // 命名空间
		ev.Name,            // 名称
		ev.ReasonCode,      // 原因代码（如 CrashLoopBackOff）
		ev.Severity,        // 严重等级
		ev.Message,         // 报错信息
		ev.Category,        // 分类（如资源型 / 系统型）
		ev.Timestamp.Format(time.RFC3339), // 精确时间戳
	)
}

// ✅ cleanExpiredSentEvents: 移除已过期的发送记录
func cleanExpiredSentEvents() {
	now := time.Now()
	for key, t := range sentEvents {
		if now.Sub(t) > sentEventTTL {
			delete(sentEvents, key)
		}
	}
}

// ✅ filterNewEvents: 过滤出尚未发送或已过期的事件列表
//
// 输入：events - 原始事件列表
// 返回：仅包含“未曾发送”或“发送已过期”的新事件列表（供 Slack 发送）
func filterNewEvents(events []model.LogEvent) []model.LogEvent {
	sentEventsMu.Lock()
	defer sentEventsMu.Unlock()

	// 🔄 清理超时缓存
	cleanExpiredSentEvents()

	newEvents := make([]model.LogEvent, 0)
	now := time.Now()

	for _, ev := range events {
		key := eventKey(ev)
		if _, sent := sentEvents[key]; !sent {
			newEvents = append(newEvents, ev)
			sentEvents[key] = now // ✅ 记录该事件发送时间
		}
	}
	return newEvents
}
