// source/event/datahub/cleaner.go
// 事件清洗与去重
package datahub

import (
	"time"

	"AtlHyper/atlhyper_agent/config"
	model "AtlHyper/model/transport"
)

// CleanAndStoreEvents 清理原始事件池并重建清理池（线程安全）
//
// 该函数由定时清理器周期性调用，主要任务包括：
//  1. 清洗原始事件池，去除过期事件
//  2. 重建清理池，去重并用于告警判断与日志写入
func CleanAndStoreEvents() {
	mu.Lock()
	defer mu.Unlock()

	// 清理原始事件池
	cleanEventPool()

	// 重建清理池
	rebuildCleanedEventPool()
}

// cleanEventPool 清理原始事件池（内部函数，需在锁内调用）
func cleanEventPool() {
	rawDuration := config.GlobalConfig.Diagnosis.RetentionRawDuration
	now := time.Now()
	newRaw := make([]model.LogEvent, 0)

	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) <= rawDuration {
			newRaw = append(newRaw, ev)
		}
	}
	eventPool = newRaw
}

// rebuildCleanedEventPool 重建清理池（内部函数，需在锁内调用）
func rebuildCleanedEventPool() {
	cleanedDuration := config.GlobalConfig.Diagnosis.RetentionCleanedDuration
	now := time.Now()

	uniqueMap := make(map[string]model.LogEvent)
	newCleaned := make([]model.LogEvent, 0)

	// 从原始池提取并去重
	for _, ev := range eventPool {
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode
		if _, exists := uniqueMap[key]; !exists {
			uniqueMap[key] = ev
			newCleaned = append(newCleaned, ev)
		}
	}

	// 合并上一轮清理池中未过期的事件
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
}
