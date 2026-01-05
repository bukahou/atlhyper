// package: external/logger/writer.go
package logger

import (
	"AtlHyper/atlhyper_master/db/repository/eventlog"
	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model/transport" // 含 Source 常量定义
	model "AtlHyper/model/transport"
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
		if r.Source != transport.SourceK8sEvent {
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


