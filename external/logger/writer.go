package logger

import (
	"NeuroController/model"
	"NeuroController/sync/center/http/commonapi"
	"log"
	"sync"
)

type writeRecord struct {
	Message  string
	Severity string
	Category string
}

var (
	writeMu      sync.Mutex
	lastWriteMap = make(map[string]writeRecord)
)

// WriteNewCleanedEventsToFile ✅ 将清理池中“新增或变更”的事件写入 JSON 文件（带写入缓存去重）
//
// ✨ 功能：
//   - 避免重复写入：仅写入与上一次相比内容发生变化的事件
//   - 记录写入缓存（lastWriteMap），用于判断事件是否“真正更新”
//   - 使用互斥锁 writeMu 保证并发安全
//   - 写入时调用 DumpEventsToJSONFile，并用 recover 防止崩溃
//
// 📦 使用场景：
//   - 由定时器周期性触发，将更新过的清理事件持久化
//   - 提供结构化日志供后续分析与查询
func WriteNewCleanedEventsToFile() {
	// 🧵 加锁，避免与其他写入操作并发冲突
	writeMu.Lock()
	defer writeMu.Unlock()

	// 🧪 获取当前清理池快照（已去重 & 时间过滤）
	// cleaned := interfaces.GetCleanedEventLogs()

	var cleaned []model.LogEvent
	for _, group := range commonapi.GetCleanedEventsFromAgents() {
		cleaned = append(cleaned, group...)
	}


	// ✅ 清理池为空时，表示系统健康或已恢复，清空写入缓存以便后续重建差异状态
	if len(cleaned) == 0 {
		lastWriteMap = make(map[string]writeRecord)
		return
	}

	// 📥 存放需要写入的新事件
	newLogs := make([]model.LogEvent, 0)

	// 🔁 遍历清理池，检测是否为“首次写入”或“字段有变化”
	for _, ev := range cleaned {
		// 生成用于比对的唯一键（包含 Kind + Namespace + Name + ReasonCode + Message）
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message

		// 获取上一轮写入的记录
		last, exists := lastWriteMap[key]

		// 判断事件是否有变化：
		//   - 首次出现
		//   - message 字段变更
		//   - severity 级别变更
		//   - category 分类变更
		changed := !exists ||
			ev.Message != last.Message ||
			ev.Severity != last.Severity ||
			ev.Category != last.Category

		// 若存在变化，则添加进待写入列表，并更新写入缓存
		if changed {
			newLogs = append(newLogs, ev)
			lastWriteMap[key] = writeRecord{
				Message:  ev.Message,
				Severity: ev.Severity,
				Category: ev.Category,
			}
		}
	}

	// ✅ 如果存在变更事件，则触发写入
	if len(newLogs) > 0 {
		// ⚠️ 用 defer + recover 保护写入流程，防止 JSON 写入崩溃影响主流程
		defer func() {
			if r := recover(); r != nil {
				log.Printf("❌ 写入 JSON 文件过程中发生 panic: %v", r)
			}
		}()

		// ✍️ 调用写入函数（按 JSON 单行格式追加写入）
		// DumpEventsToJSONFile(newLogs)
		DumpEventsToSQLite(newLogs)

	}
}
