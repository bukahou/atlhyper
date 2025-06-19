// =======================================================================================
// 📄 external/slack/throttle.go
//
// 💬 Description:
//     Provides a throttled Slack alert mechanism to avoid sending duplicate alerts
//     within a short time window. Helps prevent alert storms.
//
// 🔐 Features:
//     - Built-in mutex for thread-safe operation
//     - Exposes a single entry point: SendSlackAlertWithThrottle
//     - Automatically filters repeated alerts with unchanged state
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
