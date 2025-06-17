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
	"NeuroController/interfaces"
	"NeuroController/internal/types"
	"fmt"
	"log"
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

func DispatchSlackAlertFromCleanedEvents() {

	// ✅ 获取清洗后的事件池
	events := interfaces.GetCleanedEventLogs()
	if len(events) == 0 {
		return
	}

	// ✅ 格式化为轻量级告警数据
	shouldAlert, subject, data := interfaces.GetLightweightAlertGroup(events)
	if !shouldAlert {
		log.Println("✅ [SlackDispatch] 当前无异常事件，未触发 Slack 告警。")
		return
	}

	// ✅ 构建 BlockKit 并节流发送
	err := SendSlackAlertWithThrottle(subject, data)
	if err != nil {
		log.Printf("❌ [SlackDispatch] Slack 发送失败: %v\n", err)
	} else {
		log.Printf("📬 [SlackDispatch] Slack 告警已发送，标题: \"%s\"\n", subject)
	}
}
