// get_event_history：查 master 持久化的 K8s Warning 事件（master 侧读库，不下发 Agent）
//
// query_cluster get_events 走 Agent 实时查 API Server，只能看到约 1 小时；
// 事件分析常在事后进行，更早的证据只在 cluster_events 表里。
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
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

type eventHistoryParams struct {
	Namespace    string `json:"namespace,omitempty" jsonschema:"按关联资源的命名空间过滤"`
	InvolvedKind string `json:"involved_kind,omitempty" jsonschema:"关联资源类型，如 Pod、Node（指定 involved_name 时不填默认 Pod）"`
	InvolvedName string `json:"involved_name,omitempty" jsonschema:"关联资源名称（填写时按该资源精确查询）"`
	SinceMinutes int    `json:"since_minutes,omitempty" jsonschema:"回看多少分钟（按最近一次发生的时间），不填为 180"`
	Limit        int    `json:"limit,omitempty" jsonschema:"最多返回条数，不填为 30"`
}

func eventHistoryTool(events EventLogReader) *toolkit.Tool {
	return toolkit.MustDefine(toolkit.Spec[eventHistoryParams]{
		Name: "get_event_history",
		Description: "查询 atlhyper 持久化的 K8s Warning 事件（不受 K8s 约 1 小时保留期限制）。" +
			"query_cluster 的 get_events 只能看到最近约 1 小时；排查更早的问题（如已恢复的事件）用本工具。" +
			"同资源同原因合并为一行，count 为自首次以来的累计次数。",
		Permission: toolkit.PermEventsRead,
		Refine: func(s *jsonschema.Schema) {
			s.Properties["since_minutes"].Minimum = ptr(1.0)
			s.Properties["since_minutes"].Maximum = ptr(float64(maxEventHistoryMinutes))
			s.Properties["limit"].Minimum = ptr(1.0)
			s.Properties["limit"].Maximum = ptr(float64(maxEventHistoryLimit))
		},
		Handler: func(ctx context.Context, clusterID string, p eventHistoryParams) (string, error) {
			return eventHistory(ctx, events, clusterID, p)
		},
	})
}

func eventHistory(ctx context.Context, events EventLogReader, clusterID string, p eventHistoryParams) (string, error) {
	kind, namespace, name := p.InvolvedKind, p.Namespace, p.InvolvedName
	minutes := orDefault(p.SinceMinutes, defaultEventHistoryMinutes)
	limit := orDefault(p.Limit, defaultEventHistoryLimit)
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
		return "", fmt.Errorf("查询持久化事件失败: %w", err)
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
	return marshal(map[string]any{
		"source":       "atlhyper 持久化的 K8s Warning 事件（同资源同原因合并为一行，count 为自首次以来的累计次数）",
		"sinceMinutes": minutes,
		"events":       items,
		"count":        len(items),
	})
}
