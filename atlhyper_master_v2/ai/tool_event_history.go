// atlhyper_master_v2/ai/tool_event_history.go
// get_event_history：查 master 持久化的 K8s Warning 事件（master 侧读库，不下发 Agent）
//
// query_cluster get_events 走 Agent 实时查 API Server，只能看到约 1 小时；
// 事件分析常在事后进行，更早的证据只在 cluster_events 表里。
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

const (
	defaultEventHistoryMinutes = 180
	maxEventHistoryMinutes     = 7 * 24 * 60
	defaultEventHistoryLimit   = 30
	maxEventHistoryLimit       = 100
	eventHistoryScanLimit      = 500 // 按集群查时先取这么多，再在内存里按 namespace / kind 过滤
)

// EventLogReader 持久化 event 的最小读接口（database.EventRepository 满足）
type EventLogReader interface {
	ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*database.ClusterEvent, error)
	ListByCluster(ctx context.Context, clusterID string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error)
}

// eventHistoryItem 返回给模型的单行（count 为自首次以来的累计次数）
type eventHistoryItem struct {
	Kind      string    `json:"kind"`
	Namespace string    `json:"namespace,omitempty"`
	Name      string    `json:"name"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int32     `json:"count"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

// NewEventHistoryHandler 创建持久化事件查询 Tool Handler
func NewEventHistoryHandler(events EventLogReader) ToolHandler {
	return func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		kind := getString(params, "involved_kind")
		namespace := getString(params, "namespace")
		name := getString(params, "involved_name")
		minutes := min(max(getInt(params, "since_minutes", defaultEventHistoryMinutes), 1), maxEventHistoryMinutes)
		limit := min(max(getInt(params, "limit", defaultEventHistoryLimit), 1), maxEventHistoryLimit)
		// 库中时间为 RFC3339（UTC，Z 结尾）且按字符串比较：查询下界也必须是 UTC
		since := time.Now().Add(-time.Duration(minutes) * time.Minute).UTC()

		var rows []*database.ClusterEvent
		var err error
		if name != "" {
			if kind == "" {
				kind = "Pod"
			}
			rows, err = events.ListByInvolvedResource(ctx, clusterID, kind, namespace, name)
		} else {
			rows, err = events.ListByCluster(ctx, clusterID, database.EventQueryOpts{Since: since, Limit: eventHistoryScanLimit})
		}
		if err != nil {
			return fmt.Sprintf("查询持久化事件失败: %v", err), nil
		}

		items := make([]eventHistoryItem, 0, limit)
		for _, ev := range rows {
			if len(items) == limit {
				break
			}
			if ev.LastTimestamp.Before(since) ||
				(namespace != "" && ev.InvolvedNamespace != namespace) ||
				(kind != "" && ev.InvolvedKind != kind) {
				continue
			}
			items = append(items, eventHistoryItem{
				Kind: ev.InvolvedKind, Namespace: ev.InvolvedNamespace, Name: ev.InvolvedName,
				Reason: ev.Reason, Message: ev.Message, Count: ev.Count,
				FirstSeen: ev.FirstTimestamp, LastSeen: ev.LastTimestamp,
			})
		}

		out, _ := json.Marshal(map[string]interface{}{
			"source":       "atlhyper 持久化的 K8s Warning 事件（同资源同原因合并为一行，count 为自首次以来的累计次数）",
			"sinceMinutes": minutes,
			"events":       items,
			"count":        len(items),
		})
		return string(out), nil
	}
}
