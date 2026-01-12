// repository/eventwriter/writer.go
// 事件日志同步写入器（内存 → SQL）
package eventwriter

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/atlhyper_master/store/memory"
	"AtlHyper/model/transport"
)

var (
	lastSync   time.Time
	lastSyncMu sync.Mutex
)

// SyncEventsToSQL 将内存中的事件同步到 SQL 数据库
func SyncEventsToSQL(ctx context.Context) error {
	lastSyncMu.Lock()
	since := lastSync
	lastSyncMu.Unlock()

	// 获取内存快照
	snap := memory.Snapshot()
	if len(snap) == 0 {
		return nil
	}

	// 过滤出事件类型的记录
	var events []transport.EventLog
	latestTime := since

	for _, rec := range snap {
		if rec.Source != transport.SourceK8sEvent {
			continue
		}

		// 只处理 since 之后的记录
		if !since.IsZero() && !rec.EnqueuedAt.After(since) {
			continue
		}

		// 解码事件
		evs, err := decodeEvents(rec.Payload)
		if err != nil {
			continue
		}

		for _, ev := range evs {
			events = append(events, transport.EventLog{
				ClusterID: rec.ClusterID,
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

		if rec.EnqueuedAt.After(latestTime) {
			latestTime = rec.EnqueuedAt
		}
	}

	if len(events) == 0 {
		return nil
	}

	// 批量写入数据库
	if err := repository.Event.InsertBatch(ctx, events); err != nil {
		return err
	}

	// 更新同步时间
	lastSyncMu.Lock()
	lastSync = latestTime
	lastSyncMu.Unlock()

	log.Printf("📝 同步 %d 条事件到数据库", len(events))
	return nil
}

// decodeEvents 解码事件 Payload
func decodeEvents(raw []byte) ([]transport.LogEvent, error) {
	var arr []transport.LogEvent
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	var wrap struct {
		Events []transport.LogEvent `json:"events"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Events) > 0 {
		return wrap.Events, nil
	}

	return nil, nil
}
