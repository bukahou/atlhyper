package diagnosis

import (
	"AtlHyper/atlhyper_agent/config"
	model "AtlHyper/model/transport"
	"sync"
	"time"
)

var (
	mu               sync.Mutex
	cleanedEventPool []model.LogEvent // 去重后的清理池
)

// CleanEventPool ✅ 清理原始事件池：只保留最近 N 分钟内的事件（时间窗口由配置项控制）
//
// 该函数负责定期清理 eventPool 中过期的事件，避免内存无限增长。
// 配置项 RetentionRawDuration 决定保留的时间窗口（如：10 分钟）
// 被清理的事件将不会参与后续告警分析和写入操作。
func CleanEventPool() {
	// 获取配置中定义的“原始事件保留时长”，例如：10 分钟
	rawDuration := config.GlobalConfig.Diagnosis.RetentionRawDuration

	// 获取当前时间用于计算每条事件的过期性
	now := time.Now()

	// 创建一个新的事件池，用于保存仍在时间窗口内的事件
	newRaw := make([]model.LogEvent, 0)

	// 遍历原始事件池
	for _, ev := range eventPool {
		// 如果事件发生时间在保留时间窗口内，则保留
		if now.Sub(ev.Timestamp) <= rawDuration {
			newRaw = append(newRaw, ev)
		}
	}

	// 替换旧事件池，仅保留未过期事件
	eventPool = newRaw
}

// RebuildCleanedEventPool ✅ 重建清理池：从原始事件池中提取近期有效事件，并进行去重
//
// 功能说明：
//   - 合并原始事件池与上一轮清理池中的“未过期”事件
//   - 避免重复事件（通过 Kind + Namespace + Name + ReasonCode 生成唯一键）
//   - 清理窗口由 config.Diagnosis.RetentionCleanedDuration 控制
//
// 重建后的 cleanedEventPool 将用于告警判定与日志写入。
func RebuildCleanedEventPool() {
	// 获取清理池保留时间（例如 30 分钟内的事件将被保留）
	cleanedDuration := config.GlobalConfig.Diagnosis.RetentionCleanedDuration

	now := time.Now()

	// 唯一增量池
	uniqueMap := make(map[string]model.LogEvent)

	// newCleaned 清理池临时容器
	newCleaned := make([]model.LogEvent, 0)

	// 第一步：从 eventPool（原始池）中提取未过期的事件，并去重
	for _, ev := range eventPool {
		// 跳过已超出保留窗口的事件
		// if now.Sub(ev.Timestamp) > cleanedDuration {
		// 	continue
		// }

		// 构造唯一键：Kind|Namespace|Name|ReasonCode
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode

		// 如果该事件未出现过，则添加到新清理池
		if _, exists := uniqueMap[key]; !exists {
			uniqueMap[key] = ev
			newCleaned = append(newCleaned, ev)
		}
	}

	// 第二步：合并上一轮 cleanedEventPool 中仍未过期、且不重复的事件
	for _, ev := range cleanedEventPool {
		// 保留尚未超时的事件
		if now.Sub(ev.Timestamp) <= cleanedDuration {
			// 构造相同的唯一键
			key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode

			// 若该事件在当前轮中未出现，则保留
			if _, exists := uniqueMap[key]; !exists {
				uniqueMap[key] = ev
				newCleaned = append(newCleaned, ev)
			}
		}
	}

	// 替换旧的清理池，完成清理池重建
	cleanedEventPool = newCleaned
}

// GetCleanedEvents ✅ 获取当前的清理池事件列表（线程安全）
//
// 该函数用于外部读取当前清理池中的结构化事件列表，常用于告警判断或日志写入。
// 为保证并发安全，函数内部使用全局互斥锁（mu）防止读写冲突。
//
// 注意：返回的是 cleanedEventPool 的浅拷贝，确保调用者获取的数据不会影响原始池内容。
func GetCleanedEvents() []model.LogEvent {
	// 加锁，防止在读取期间其他 goroutine 修改 cleanedEventPool
	mu.Lock()
	defer mu.Unlock()

	// 创建一个与 cleanedEventPool 等长的切片
	copy := make([]model.LogEvent, len(cleanedEventPool))

	// 使用 append 构造新切片，避免直接引用原始底层数组
	copy = append(copy[:0], cleanedEventPool...)

	// 返回复制后的结果
	return copy
}

// CleanAndStoreEvents ✅ 公共函数：清理原始事件池并重建清理池（线程安全）
//
// 该函数由定时清理器周期性调用，主要任务包括：
//  1. CleanEventPool: 清洗原始事件池，去重、合并、筛选无效事件
//  2. RebuildCleanedEventPool: 重建结构化的清理池，用于告警判断与日志写入
//
// 为确保并发安全，整个过程使用全局互斥锁 mu 包裹，避免并发读写造成数据竞争。
func CleanAndStoreEvents() {

	// 加锁，确保清理过程中不会有其他线程读写事件池
	mu.Lock()
	defer mu.Unlock()

	// 第一步：处理原始事件池，清除过期或冗余事件
	CleanEventPool()

	// 第二步：从处理结果重建清理池，准备写入磁盘或用于告警判断
	RebuildCleanedEventPool()
}
