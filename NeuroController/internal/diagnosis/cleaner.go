package diagnosis

import (
	"fmt"
	"sync"
	"time"
)

// =======================================================================================
// 📄 diagnosis/cleaner.go
//
// ✨ 功能说明：
//     实现日志事件清理逻辑，包括：去重、时间过期移除。
//     支持定时清洗并维护一个独立的 cleanedEventPool，供 matcher 使用。
// =======================================================================================

var (
	mu               sync.Mutex
	cleanedEventPool []LogEvent // 去重后的清理池
)

const (
	retentionRawDuration     = 10 * time.Minute
	retentionCleanedDuration = 5 * time.Minute
)

// ✅ 清理原始池：保留最近 10 分钟
func CleanEventPool() {
	now := time.Now()
	newRaw := make([]LogEvent, 0)
	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) <= retentionRawDuration {
			newRaw = append(newRaw, ev)
		}
	}
	eventPool = newRaw
}

// ✅ 重建清理池：从 eventPool 和旧 cleanedEventPool 合并去重生成新清理池
func RebuildCleanedEventPool() {
	now := time.Now()
	uniqueMap := make(map[string]LogEvent)
	newCleaned := make([]LogEvent, 0)

	// 筛选并添加来自原始池的近5分钟事件
	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) > retentionCleanedDuration {
			continue
		}
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode
		if _, exists := uniqueMap[key]; !exists {
			uniqueMap[key] = ev
			newCleaned = append(newCleaned, ev)
		}
	}

	// 清理，清理池中过期和重复的事件
	for _, ev := range cleanedEventPool {
		if now.Sub(ev.Timestamp) <= retentionCleanedDuration {
			key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode
			if _, exists := uniqueMap[key]; !exists {
				uniqueMap[key] = ev
				newCleaned = append(newCleaned, ev)
			}
		}
	}

	cleanedEventPool = newCleaned
}

// ✅ 周期清理入口
func CleanAndStoreEvents() {
	mu.Lock()
	defer mu.Unlock()
	//清理原始池旧数据
	CleanEventPool()
	//清理，清理池池旧数据
	RebuildCleanedEventPool()
}

// ✅ 外部接口：获取当前清理池中的所有日志事件
func GetCleanedEvents() []LogEvent {
	mu.Lock()
	defer mu.Unlock()

	copy := make([]LogEvent, len(cleanedEventPool))
	copy = append(copy[:0], cleanedEventPool...)
	return copy
}

// ✅ 启动定时清理（建议在 main.go 或 controller 启动入口调用）
func StartCleanerLoop(interval time.Duration) {
	go func() {
		for {
			CleanAndStoreEvents()
			// 🧪 测试用打印，可删除
			printCleanedEvents()
			time.Sleep(interval)
		}
	}()
}

// ✅ 测试用：打印清理池内容
func printCleanedEvents() {
	events := GetCleanedEvents()
	fmt.Println("──────────────────────────────")
	fmt.Println("🧼 当前清理池状态：")
	for _, ev := range events {
		fmt.Printf(" - [%s] %s/%s → %s (%s)\n",
			ev.Kind, ev.Namespace, ev.Name, ev.ReasonCode, ev.Timestamp.Format("15:04:05"))
	}
	fmt.Printf("🧮 清理后日志总数：%d 条\n", len(events))
	fmt.Println("──────────────────────────────")
}
