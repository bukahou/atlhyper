package interfaces

import (
	"AtlHyper/atlhyper_master/db/repository/eventlog"
	"AtlHyper/atlhyper_master/master_store"
	modelEvent "AtlHyper/model"
	model "AtlHyper/model/event"
	"context"
	"encoding/json"
	"log"
	"time"
)

// 数据源常量
const sourceK8sEvent = modelEvent.SourceK8sEvent

// GetRecentEventsByCluster
// 从 Store 中获取指定集群最近 within 时间窗口内的事件（LogEvent 全量）
// 注意：返回的是去重后的上报事件（Store 保持 15 分钟活性）
func GetRecentEventsByCluster(_ context.Context, clusterID string, within time.Duration) ([]model.LogEvent, error) {
	if clusterID == "" {
		return []model.LogEvent{}, nil
	}

	var result []model.LogEvent
	cutoff := time.Now().Add(-within)

	// 遍历 Store 快照
	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceK8sEvent || r.ClusterID != clusterID {
			continue
		}

		// 解码 payload: {"events":[...]} 或直接数组
		var wrapper struct {
			Events []model.LogEvent `json:"events"`
		}
		if err := json.Unmarshal(r.Payload, &wrapper); err != nil {
			log.Printf("[event_iface] decode payload fail: cluster=%s err=%v", clusterID, err)
			continue
		}

		// 过滤时间窗
		for _, ev := range wrapper.Events {
			if ev.Timestamp.After(cutoff) {
				result = append(result, ev)
			}
		}
	}

	return result, nil
}


func GetRecentEventLogs(clusterID string, withinDays int) ([]model.EventLog, error) {
	// 构造起始时间戳：当前时间 - N 天
	since := time.Now().
		Add(-time.Duration(withinDays) * 24 * time.Hour).
		Format(time.RFC3339)

	// 调用底层持久层查询函数
	logs, err := eventlog.GetEventLogsSince(clusterID, since)
	if err != nil {
		return nil, err
	}
	
	return logs, nil
}