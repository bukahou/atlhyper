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

	// 🛑 节流检查
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
	fmt.Println("📨 [SlackDispatch] 开始 Slack 告警调度流程...")

	// ✅ 聚合评估是否触发告警
	shouldAlert, subject, data := interfaces.GetAlertGroupIfNecessary()
	if !shouldAlert {
		fmt.Println("ℹ️ [SlackDispatch] 当前无需发送 Slack 告警，调度流程结束。")
		return
	}

	// ✅ 发送 Slack 消息（带节流控制）
	err := SendSlackAlertWithThrottle(subject, data)
	if err != nil {
		fmt.Printf("❌ [SlackDispatch] Slack 发送失败: %v\n", err)
	} else {
		fmt.Printf("📬 [SlackDispatch] Slack 告警已发送，标题: \"%s\"\n", subject)
	}
}
