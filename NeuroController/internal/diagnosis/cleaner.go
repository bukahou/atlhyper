// =======================================================================================
// 📄 diagnosis/cleaner.go
//
// ✨ Description:
//     Implements log event cleanup logic, including deduplication and time-based expiration.
//     Maintains a periodically refreshed `cleanedEventPool` that can be used by the matcher module.
//
// 🧼 Responsibilities:
//     - Remove outdated events from the raw event pool
//     - Merge and deduplicate events into the cleaned pool (within retention window)
//     - Provide access to the cleaned pool
//     - Run as a scheduled background cleaner
// =======================================================================================

package diagnosis

import (
	"NeuroController/config"
	"NeuroController/internal/alerter"
	"NeuroController/internal/types"
	"fmt"
	"sync"
	"time"
)

var (
	mu               sync.Mutex
	cleanedEventPool []types.LogEvent // 去重后的清理池
)

// 已经转移到配置文件中统一管理
// const (
// 	retentionRawDuration     = 10 * time.Minute // 原始事件保留时间
// 	retentionCleanedDuration = 5 * time.Minute  // 清理池事件保留时间
// )

// ✅ 清理原始事件池：只保留最近 10 分钟内的事件
func CleanEventPool() {
	rawDuration := config.GlobalConfig.Diagnosis.RetentionRawDuration

	now := time.Now()
	newRaw := make([]types.LogEvent, 0)
	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) <= rawDuration {
			newRaw = append(newRaw, ev)
		}
	}
	eventPool = newRaw
}

// ✅ 重建清理池：从原始池中合并新事件并去重
func RebuildCleanedEventPool() {
	cleanedDuration := config.GlobalConfig.Diagnosis.RetentionCleanedDuration
	now := time.Now()
	uniqueMap := make(map[string]types.LogEvent)
	newCleaned := make([]types.LogEvent, 0)

	// 添加来自原始池的近期事件（在清理保留期内）
	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) > cleanedDuration {
			continue
		}
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode
		if _, exists := uniqueMap[key]; !exists {
			uniqueMap[key] = ev
			newCleaned = append(newCleaned, ev)
		}
	}

	// 添加上一轮清理池中尚未过期且不重复的事件
	for _, ev := range cleanedEventPool {
		if now.Sub(ev.Timestamp) <= cleanedDuration {
			key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode
			if _, exists := uniqueMap[key]; !exists {
				uniqueMap[key] = ev
				newCleaned = append(newCleaned, ev)
			}
		}
	}

	cleanedEventPool = newCleaned
	alerter.EvaluateAlertsFromCleanedEvents(cleanedEventPool)
}

// ✅ 公共函数：清理原始池和清理池（线程安全）
func CleanAndStoreEvents() {
	mu.Lock()
	defer mu.Unlock()
	CleanEventPool()
	RebuildCleanedEventPool()
}

// ✅ 获取当前的清理池列表（线程安全）
func GetCleanedEvents() []types.LogEvent {
	mu.Lock()
	defer mu.Unlock()

	copy := make([]types.LogEvent, len(cleanedEventPool))
	copy = append(copy[:0], cleanedEventPool...)
	return copy
}

// ✅ 启动后台定时清理循环
//
// （应由 main.go 或控制器入口调用）
func StartCleanerLoop(interval time.Duration) {
	go func() {
		for {
			CleanAndStoreEvents()
			// 🧪 调试用输出，可在正式部署时移除
			printCleanedEvents()
			time.Sleep(interval)
		}
	}()
}

// ✅ 调试函数：打印当前清理池的状态
func printCleanedEvents() {
	events := GetCleanedEvents()
	fmt.Println("──────────────────────────────")
	fmt.Println("🧼 当前清理事件池:")
	for _, ev := range events {
		fmt.Printf(" - [%s] %s/%s → %s (%s)\n",
			ev.Kind, ev.Namespace, ev.Name, ev.ReasonCode, ev.Timestamp.Format("15:04:05"))
	}
	fmt.Printf("🧮 总清理事件数: %d 条\n", len(events))
	fmt.Println("──────────────────────────────")
}
