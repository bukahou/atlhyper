// atlhyper_master_v2/model/convert/event.go
// cluster.Event / database.ClusterEvent → model.EventLog 转换函数
package convert

import (
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/cluster"
)

// EventLog 转换单个 cluster.Event 为 Web API 响应
func EventLog(src *cluster.Event, clusterID string) model.EventLog {
	kind := src.InvolvedObject.Kind
	if kind == "" {
		kind = "Event"
	}

	name := src.InvolvedObject.Name
	if name == "" {
		name = src.Name
	}

	return model.EventLog{
		ClusterID: clusterID,
		Category:  src.Reason,
		EventTime: formatTime(src.LastTimestamp),
		Kind:      kind,
		Message:   src.Message,
		Name:      name,
		Namespace: src.Namespace,
		Node:      src.Source,
		Reason:    src.Reason,
		Severity:  eventSeverity(src.Type, src.Reason),
		Time:      formatTime(src.FirstTimestamp),
	}
}

// EventLogs 转换多个 cluster.Event
func EventLogs(src []cluster.Event, clusterID string) []model.EventLog {
	if src == nil {
		return []model.EventLog{}
	}
	result := make([]model.EventLog, len(src))
	for i := range src {
		result[i] = EventLog(&src[i], clusterID)
	}
	return result
}

// EventLogFromDB 转换数据库事件记录为 Web API 响应
func EventLogFromDB(src *database.ClusterEvent, clusterID string) model.EventLog {
	return model.EventLog{
		ClusterID: clusterID,
		Category:  src.Reason,
		EventTime: formatTime(src.LastTimestamp),
		Kind:      src.InvolvedKind,
		Message:   src.Message,
		Name:      src.InvolvedName,
		Namespace: src.Namespace,
		Node:      src.SourceComponent,
		Reason:    src.Reason,
		Severity:  eventSeverity(src.Type, src.Reason),
		Time:      formatTime(src.FirstTimestamp),
	}
}

// EventLogsFromDB 转换多个数据库事件记录
func EventLogsFromDB(src []*database.ClusterEvent, clusterID string) []model.EventLog {
	if src == nil {
		return []model.EventLog{}
	}
	result := make([]model.EventLog, len(src))
	for i, e := range src {
		result[i] = EventLogFromDB(e, clusterID)
	}
	return result
}

// EventCardsOf 统计一批事件（按严重程度、资源类型、原因）
func EventCardsOf(rows []model.EventLog) model.EventCards {
	kinds := map[string]struct{}{}
	categories := map[string]struct{}{}
	cards := model.EventCards{TotalEvents: len(rows)}
	for _, r := range rows {
		if r.Kind != "" {
			kinds[r.Kind] = struct{}{}
		}
		if r.Reason != "" {
			categories[r.Reason] = struct{}{}
		}
		switch r.Severity {
		case model.EventSeverityCritical:
			cards.Critical++
		case model.EventSeverityWarning:
			cards.Warning++
		case model.EventSeverityError:
			cards.Error++
		default:
			cards.Info++
		}
	}
	cards.TotalAlerts = cards.Critical + cards.Warning + cards.Error
	cards.KindsCount = len(kinds)
	cards.CategoriesCount = len(categories)
	return cards
}

// criticalEventReasons Warning 事件里会直接导致故障的原因：判为 critical
var criticalEventReasons = map[string]bool{
	"OOMKilling":       true,
	"CrashLoopBackOff": true,
	"FailedScheduling": true,
	"FailedMount":      true,
	"NodeNotReady":     true,
	"FailedBinding":    true,
}

// eventSeverity K8s 事件类型 + 原因 → 严重程度（全项目只在这里判定）
func eventSeverity(eventType, reason string) string {
	switch eventType {
	case "Warning":
		if criticalEventReasons[reason] {
			return model.EventSeverityCritical
		}
		return model.EventSeverityWarning
	case "Error":
		return model.EventSeverityError
	default:
		return model.EventSeverityInfo
	}
}

// formatTime 格式化 time.Time 为 RFC3339 字符串
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
