// package: external/logger/writer.go
package logger

import (
	"NeuroController/db/repository/eventlog"
	"NeuroController/external/master_store"
	"NeuroController/model"
	"encoding/json"
	"log"
	"strconv"
	"time"
)

type writeRecord struct {
	Message  string
	Severity string
	Category string
}

// 用于缓存上一次已经写入数据库的事件快照
// key: ClusterID + Kind + Namespace + Name + Reason + Message
// val: 上次写入时的 Message / Severity / Category
var lastWriteMap = make(map[string]writeRecord)

// 截断长 JSON，避免错误日志刷屏
func shrinkJSON(b []byte, max int) string {
	s := string(b)
	if len(s) > max {
		return s[:max] + "...(len=" + strconv.Itoa(len(s)) + ")"
	}
	return s
}

// 解析 Envelope.Payload 为 LogEvent（单条 / 切片 / {"events":[...] }）
func decodeEnvelopeEvents(payload json.RawMessage) ([]model.LogEvent, error) {
	// 1) 单条
	var one model.LogEvent
	if err := json.Unmarshal(payload, &one); err == nil {
		if !one.Timestamp.IsZero() || one.Kind != "" || one.Message != "" || one.ReasonCode != "" {
			return []model.LogEvent{one}, nil
		}
	}

	// 2) 切片
	var many []model.LogEvent
	if err := json.Unmarshal(payload, &many); err == nil && len(many) > 0 {
		return many, nil
	}

	// 3) 包裹 {"events":[...]}
	var wrap struct {
		Events []model.LogEvent `json:"events"`
	}
	if err := json.Unmarshal(payload, &wrap); err == nil && len(wrap.Events) > 0 {
		return wrap.Events, nil
	}

	// 4) 都不匹配 → 返回错误
	return nil, json.Unmarshal(payload, &one)
}

// 主流程：从内存快照读取 → 仅处理 k8s_event → 解析 → 去重/变更判断 → 落库（仅错误打日志）
func WriteNewCleanedEventsToFile() {
	// 1) 读快照
	recs := master_store.Snapshot()

	// 2) 过滤 k8s_event 并解析
	eventLogs := make([]model.EventLog, 0)
	for _, r := range recs {
		if r.Source != "k8s_event" {
			continue
		}
		events, err := decodeEnvelopeEvents(r.Payload)
		if err != nil {
			log.Printf("❌ [writer] 解析 k8s_event 失败: cluster=%s err=%v payload=%s",
				r.ClusterID, err, shrinkJSON(r.Payload, 240))
			continue
		}
		if len(events) == 0 {
			continue
		}
		for _, ev := range events {
			eventLogs = append(eventLogs, model.EventLog{
				ClusterID: r.ClusterID,
				Category:  ev.Category,
				EventTime: ev.Timestamp.Format(time.RFC3339),
				Kind:      ev.Kind,
				Message:   ev.Message,
				Name:      ev.Name,
				Namespace: ev.Namespace,
				Node:      ev.Node,
				Reason:    ev.ReasonCode,
				Severity:  ev.Severity,
				Time:      time.Now().Format(time.RFC3339),
			})
		}
	}

	// 3) 无事件：清空缓存后返回（静默）
	if len(eventLogs) == 0 {
		lastWriteMap = make(map[string]writeRecord)
		return
	}

	// 4) 去重/变更判定（与缓存对比，仅变更才落库）
	newEventRows := make([]model.EventLog, 0, len(eventLogs))
	for _, ev := range eventLogs {
		cacheKey := ev.ClusterID + "|" + ev.Kind + "|" + ev.Namespace + "|" +
			ev.Name + "|" + ev.Reason + "|" + ev.Message

		last, exists := lastWriteMap[cacheKey]
		changed := !exists ||
			ev.Message != last.Message ||
			ev.Severity != last.Severity ||
			ev.Category != last.Category

		if !changed {
			continue
		}

		lastWriteMap[cacheKey] = writeRecord{
			Message:  ev.Message,
			Severity: ev.Severity,
			Category: ev.Category,
		}
		newEventRows = append(newEventRows, ev)
	}

	// 5) 无增量：静默返回
	if len(newEventRows) == 0 {
		return
	}

	// 6) 写库（仅错误打日志；保底 recover）
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ [writer] 写入 SQLite 发生 panic：%v", r)
		}
	}()
	for _, row := range newEventRows {
		if err := eventlog.InsertEventLog(row); err != nil {
			log.Printf("❌ [writer] 插入失败：%v | 行=%+v", err, row)
		}
	}
}


































// package logger

// import (
// 	"NeuroController/db/repository/eventlog"
// 	"NeuroController/model"
// 	"NeuroController/sync/center/http/commonapi"
// 	"log"
// 	"time"
// )


// type writeRecord struct {
// 	Message  string
// 	Severity string
// 	Category string
// }

// // 单协程场景下，无需互斥锁
// var (
// 	lastWriteMap = make(map[string]writeRecord)
// )

// // WriteNewCleanedEventsToFile ✅ 将清理池中“新增或变更”的事件写入（带缓存去重，单协程版）
// func WriteNewCleanedEventsToFile() {
// 	// 1) 获取当前清理池快照（已去重 & 时间过滤）
// 	var cleaned []model.LogEvent
// 	for _, group := range commonapi.GetCleanedEventsFromAgents() {
// 		cleaned = append(cleaned, group...)
// 	}

// 	// 2) 清理池为空：清空写入缓存后返回
// 	if len(cleaned) == 0 {
// 		lastWriteMap = make(map[string]writeRecord)
// 		return
// 	}

// 	// 3) 生成增量写入列表
// 	newLogs := make([]model.LogEvent, 0, len(cleaned))
// 	for _, ev := range cleaned {
// 		// 唯一键：Kind|Namespace|Name|ReasonCode|Message
// 		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message

// 		last, exists := lastWriteMap[key]
// 		changed := !exists ||
// 			ev.Message != last.Message ||
// 			ev.Severity != last.Severity ||
// 			ev.Category != last.Category

// 		if changed {
// 			newLogs = append(newLogs, ev)
// 			lastWriteMap[key] = writeRecord{
// 				Message:  ev.Message,
// 				Severity: ev.Severity,
// 				Category: ev.Category,
// 			}
// 		}
// 	}

// 	// 4) 有变更再落库
// 	if len(newLogs) == 0 {
// 		return
// 	}

// 	// 防御性保护：避免写入崩溃影响主流程
// 	defer func() {
// 		if r := recover(); r != nil {
// 			log.Printf("❌ 写入过程中发生 panic: %v", r)
// 		}
// 	}()

// 	// 你已有的持久化方式（JSON/SQLite）。当前使用 SQLite：
// 	DumpEventsToSQLite(newLogs)
// }

// // =======================================================================================
// // ✅ DumpEventsToSQLite - 批量写入事件日志到 SQLite 数据库
// //
// // 📌 用法：
// //     - 接收处理后的结构化事件列表（LogEvent）
// //     - 转换为 EventLog 数据库模型后，逐条插入 SQLite
// //
// // ⚠️ 注意：
// //     - 采用逐条插入（不批量），如需优化性能可考虑事务批量提交
// //     - 插入失败时会记录日志，但不会中断循环（容错）
// // =======================================================================================

// func DumpEventsToSQLite(events []model.LogEvent) {
// 	for _, ev := range events {
// 		// 构造用于持久化的事件结构（EventLog）
// 		err := eventlog.InsertEventLog(model.EventLog{
// 			Category:  ev.Category,                       // 异常类型分类（如 Pod、Node 等）
// 			EventTime: ev.Timestamp.Format(time.RFC3339), // 原始事件时间
// 			Kind:      ev.Kind,                          // 资源类型
// 			Message:   ev.Message,                       // 事件消息
// 			Name:      ev.Name,                          // 对象名称
// 			Namespace: ev.Namespace,                     // 命名空间
// 			Node:      ev.Node,                          // 所属节点
// 			Reason:    ev.ReasonCode,                    // 事件原因
// 			Severity:  ev.Severity,                      // 严重程度（如 Warning / Critical）
// 			Time:      time.Now().Format(time.RFC3339),  // 写入时间（记录采集时间）
// 		})

// 		// 写入失败时记录日志，但不中断
// 		if err != nil {
// 			log.Printf("❌ 插入事件到数据库失败: %v", err)
// 		}
// 	}
// }
