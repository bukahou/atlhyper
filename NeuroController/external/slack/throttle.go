// =======================================================================================
// 📄 external/slack/throttle.go
//
// 💬 Description:
//     提供带节流机制的 Slack 告警发送逻辑，防止重复发送相同类型告警。
//     默认节流间隔为 5 分钟，可调整以适应消息量。
//
// 🔐 Features:
//     - 内置互斥锁，保证并发安全
//     - 外部统一调用 SendSlackAlertWithThrottle
//     - 自动过滤频繁触发但状态未变化的告警
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package slack

import (
	"NeuroController/internal/types"
	"fmt"
	"sync"
	"time"
)

var (
	lastSlackSentTimeMu sync.Mutex
	lastSlackSentTime   time.Time
)

// ⏱️ Slack 节流时间间隔（默认每 5 分钟最多发送一次）
const slackThrottleInterval = 1 * time.Minute

// ✅ Slack 告警发送（节流封装）
// 输入：subject 和 AlertGroupData，构建 BlockKit 并决定是否发送
func SendSlackAlertWithThrottle(subject string, data types.AlertGroupData) error {
	lastSlackSentTimeMu.Lock()
	defer lastSlackSentTimeMu.Unlock()

	// 节流检查
	if !lastSlackSentTime.IsZero() && time.Since(lastSlackSentTime) < slackThrottleInterval {
		fmt.Println("⏳ [SlackThrottle] 距离上次发送过短，跳过本轮 Slack 发送。")
		return nil
	}

	// ✅ 满足节流间隔，更新时间并发送
	lastSlackSentTime = time.Now()

	payload := BuildSlackBlockFromAlert(data, subject)
	return SendSlackAlert(payload)
}

// =======================================================================================
// 🧠 内部状态：已发送事件缓存（防止重复发送）
// =======================================================================================

var (
	sentEventsMu sync.Mutex
	sentEvents   = make(map[string]time.Time) // key: eventKey(LogEvent), value: 首次发送时间
)

const sentEventTTL = 10 * time.Minute // ✅ 缓存保留时长（10分钟内不重复发）

// ✅ 构造事件唯一标识
func eventKey(ev types.LogEvent) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		ev.Kind,
		ev.Namespace,
		ev.Name,
		ev.ReasonCode,
		ev.Severity,
		ev.Message,
		ev.Category,
		ev.Timestamp.Format(time.RFC3339),
	)
}

// ✅ 清除过期缓存
func cleanExpiredSentEvents() {
	now := time.Now()
	for key, t := range sentEvents {
		if now.Sub(t) > sentEventTTL {
			delete(sentEvents, key)
		}
	}
}

// ✅ 过滤“尚未发送”或“已过期”的事件
func filterNewEvents(events []types.LogEvent) []types.LogEvent {
	sentEventsMu.Lock()
	defer sentEventsMu.Unlock()

	// 🧹 清理过期记录
	cleanExpiredSentEvents()

	newEvents := make([]types.LogEvent, 0)
	now := time.Now()

	for _, ev := range events {
		key := eventKey(ev)
		if _, sent := sentEvents[key]; !sent {
			newEvents = append(newEvents, ev)
			sentEvents[key] = now // ✅ 标记发送时间
		}
	}
	return newEvents
}
