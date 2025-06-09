package diagnosis

import (
	"NeuroController/internal/utils"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
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
	lastDumpTime     time.Time  // 上次写入时间
)

const (
	retentionRawDuration     = 10 * time.Minute
	retentionCleanedDuration = 5 * time.Minute
)

// ✅ 清理并更新清理池（不负责写入）
func CleanAndStoreEvents() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()

	// === 🧼 清理原始池：保留最近 10 分钟 ===
	newRaw := make([]LogEvent, 0)
	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) <= retentionRawDuration {
			newRaw = append(newRaw, ev)
		}
	}
	eventPool = newRaw

	// === 🧼 构建新清理池（去重）===
	uniqueMap := make(map[string]LogEvent)
	newCleaned := make([]LogEvent, 0)

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

// ✅ 外部接口：获取当前清理池中的所有日志事件
func GetCleanedEvents() []LogEvent {
	mu.Lock()
	defer mu.Unlock()

	// 返回拷贝，避免外部修改原始数据
	copy := make([]LogEvent, len(cleanedEventPool))
	copy = append(copy[:0], cleanedEventPool...)
	return copy
}

// ✅ 启动定时清理（建议在 main.go 或 controller 启动入口调用）
func StartCleanerLoop(interval time.Duration) {
	go func() {
		for {
			CleanAndStoreEvents()
			// 🧪 清理后打印一次内容。测试后删除
			printCleanedEvents()
			time.Sleep(interval)
		}
	}()
}

// 🧪 测试用：打印当前清理池中的日志事件
func printCleanedEvents() {
	events := GetCleanedEvents()
	fmt.Println("🧼 当前清理池状态：")
	for _, ev := range events {
		fmt.Printf(" - [%s] %s/%s → %s (%s)\n",
			ev.Kind, ev.Namespace, ev.Name, ev.ReasonCode, ev.Timestamp.Format("15:04:05"))
	}
	fmt.Printf("🧮 清理后日志总数：%d 条\n", len(events))
	fmt.Println("──────────────────────────────")
}

func DumpEventsToFile(events []LogEvent) {
	logDir := "./logs"
	logPath := logDir + "/cleaned_events.log"

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		utils.Warn(nil, "⚠️ 无法创建日志目录", zap.Error(err))
		return
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		utils.Warn(nil, "⚠️ 无法写入清理日志文件", zap.Error(err))
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString("🕒 Dump at " + timestamp + "\n")
	for _, ev := range events {
		line := fmt.Sprintf(" - [%s] %s/%s → %s (%s)\n",
			ev.Kind, ev.Namespace, ev.Name, ev.ReasonCode, ev.Timestamp.Format("15:04:05"))
		f.WriteString(line)
	}
	f.WriteString("──────────────────────────────\n")
}

// ✅ 仅写入上次 dump 之后新增的日志
func WriteNewCleanedEventsToFile() {
	mu.Lock()
	defer mu.Unlock()

	if len(cleanedEventPool) == 0 {
		return
	}

	newLogs := make([]LogEvent, 0)
	for _, ev := range cleanedEventPool {
		if ev.Timestamp.After(lastDumpTime) {
			newLogs = append(newLogs, ev)
		}
	}

	if len(newLogs) == 0 {
		return
	}

	DumpEventsToFile(newLogs)
	lastDumpTime = time.Now()
}
