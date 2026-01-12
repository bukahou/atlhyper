// gateway/integration/alert/alert_group.go
package alert

import (
	"context"
	"log"
	"time"

	"AtlHyper/atlhyper_master/repository"
	event "AtlHyper/model/transport"
)

// 告警侧的"最后一次写入快照"，与写文件的 lastWriteMap 分离
type alertWriteRecord struct {
	Message  string
	Severity string
	Category string
}

var alertLastWriteMap = map[string]alertWriteRecord{}

// CollectNewEventLogsForAlert 从内存快照收集 k8s_event，做去重/变更判定，返回"增量事件"
// -----------------------------------------------------------------------------
// 重构说明：
//   - 原实现直接访问 datahub.Snapshot()
//   - 新实现通过 Repository 层读取数据，遵循分层架构
// 数据流：DataHub → Repository → Alert Service
// -----------------------------------------------------------------------------
func CollectNewEventLogsForAlert() []event.EventLog {
	ctx := context.Background()

	// 1. 获取所有集群 ID
	clusterIDs, err := repository.Mem.ListClusterIDs(ctx)
	if err != nil {
		log.Printf("[alert-feed] 获取集群列表失败: %v", err)
		return nil
	}

	if len(clusterIDs) == 0 {
		return nil
	}

	// 2. 遍历每个集群，通过 Repository 读取事件
	eventLogs := make([]event.EventLog, 0, 256)
	for _, clusterID := range clusterIDs {
		events, err := repository.Mem.GetK8sEventsRecent(ctx, clusterID, 1000)
		if err != nil {
			log.Printf("[alert-feed] 读取集群 %s 事件失败: %v", clusterID, err)
			continue
		}

		for _, ev := range events {
			eventLogs = append(eventLogs, event.EventLog{
				ClusterID: clusterID,
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

	// 3. 去重/变更判定
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
