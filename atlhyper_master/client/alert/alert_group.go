// atlhyper_master/client/alert/alert_group.go
package alert

import (
	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model/transport"
	"strconv"

	event "AtlHyper/model/transport"
	"encoding/json"
	"log"
	"time"
)

// 告警侧的“最后一次写入快照”，与写文件的 lastWriteMap 分离
type alertWriteRecord struct {
	Message  string
	Severity string
	Category string
}

var alertLastWriteMap = map[string]alertWriteRecord{}

// CollectNewEventLogsForAlert 从内存快照收集 k8s_event，做去重/变更判定，返回“增量事件”
// - 不限定集群、不限定时间（你如果要加时间窗，可在这里加）
// - 与 writer 的 lastWriteMap 不同缓存，避免互相影响
func CollectNewEventLogsForAlert() []event.EventLog {
	recs := master_store.Snapshot()

	eventLogs := make([]event.EventLog, 0, 256)
	for _, r := range recs {
		if r.Source != transport.SourceK8sEvent {
			continue
		}
		events, err := decodeEnvelopeEvents(r.Payload) // 你现有的解析函数
		if err != nil {
			log.Printf("❌ [alert-feed] 解析 k8s_event 失败: cluster=%s err=%v payload=%s",
				r.ClusterID, err, shrinkJSON(r.Payload, 240))
			continue
		}
		for _, ev := range events {
			eventLogs = append(eventLogs, event.EventLog{
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

	// 去重/变更判定（与 writer 一致的 key 规则）
	newRows := make([]event.EventLog, 0, len(eventLogs))
	for _, ev := range eventLogs {
		cacheKey := ev.ClusterID + "|" + ev.Kind + "|" + ev.Namespace + "|" +
			ev.Name + "|" + ev.Reason + "|" + ev.Message

		last, exists := alertLastWriteMap[cacheKey]
		changed := !exists ||
			ev.Message != last.Message ||
			ev.Severity != last.Severity ||
			ev.Category != last.Category

		if !changed {
			continue
		}

		alertLastWriteMap[cacheKey] = alertWriteRecord{
			Message:  ev.Message,
			Severity: ev.Severity,
			Category: ev.Category,
		}
		newRows = append(newRows, ev)
	}

	return newRows
}


func decodeEnvelopeEvents(payload json.RawMessage) ([]event.LogEvent, error) {
	// 1) 单条
	var one event.LogEvent
	if err := json.Unmarshal(payload, &one); err == nil {
		if !one.Timestamp.IsZero() || one.Kind != "" || one.Message != "" || one.ReasonCode != "" {
			return []event.LogEvent{one}, nil
		}
	}

	// 2) 切片
	var many []event.LogEvent
	if err := json.Unmarshal(payload, &many); err == nil && len(many) > 0 {
		return many, nil
	}

	// 3) 包裹 {"events":[...]}
	var wrap struct {
		Events []event.LogEvent `json:"events"`
	}
	if err := json.Unmarshal(payload, &wrap); err == nil && len(wrap.Events) > 0 {
		return wrap.Events, nil
	}

	// 4) 都不匹配 → 返回错误
	return nil, json.Unmarshal(payload, &one)
}

// 截断长 JSON，避免错误日志刷屏
func shrinkJSON(b []byte, max int) string {
	s := string(b)
	if len(s) > max {
		return s[:max] + "...(len=" + strconv.Itoa(len(s)) + ")"
	}
	return s
}