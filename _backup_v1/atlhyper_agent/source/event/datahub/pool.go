// source/event/datahub/pool.go
// 事件数据池
package datahub

import (
	"sync"

	model "AtlHyper/model/transport"
)

var (
	mu               sync.Mutex
	eventPool        []model.LogEvent // 原始事件池
	cleanedEventPool []model.LogEvent // 清洗后事件池
)

func init() {
	eventPool = make([]model.LogEvent, 0)
	cleanedEventPool = make([]model.LogEvent, 0)
}

// AppendToEventPool 将事件追加到原始事件池（线程安全）
func AppendToEventPool(event model.LogEvent) {
	mu.Lock()
	defer mu.Unlock()
	eventPool = append(eventPool, event)
}

// GetCleanedEvents 获取清洗后事件列表（线程安全）
func GetCleanedEvents() []model.LogEvent {
	mu.Lock()
	defer mu.Unlock()
	cp := make([]model.LogEvent, len(cleanedEventPool))
	copy(cp, cleanedEventPool)
	return cp
}
