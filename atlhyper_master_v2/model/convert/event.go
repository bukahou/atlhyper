// atlhyper_master_v2/model/convert/event.go
// model_v2.Event / database.ClusterEvent → model.EventLog 转换函数
package convert

import (
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// EventLog 转换单个 model_v2.Event 为 Web API 响应
func EventLog(src *model_v2.Event, clusterID string) model.EventLog {
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
		Severity:  mapSeverity(src.Type),
		Time:      formatTime(src.FirstTimestamp),
	}
}

// EventLogs 转换多个 model_v2.Event
func EventLogs(src []model_v2.Event, clusterID string) []model.EventLog {
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
		Severity:  mapSeverity(src.Type),
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

// EventOverview 将事件列表转换为概览（含统计卡片）
func EventOverview(src []model_v2.Event, clusterID string) model.EventOverview {
	rows := EventLogs(src, clusterID)

	kinds := map[string]struct{}{}
	categories := map[string]struct{}{}
	var warning, errCount, info int

	for _, r := range rows {
		if r.Kind != "" {
			kinds[r.Kind] = struct{}{}
		}
		if r.Reason != "" {
			categories[r.Reason] = struct{}{}
		}
		switch r.Severity {
		case "warning":
			warning++
		case "error":
			errCount++
		default:
			info++
		}
	}

	return model.EventOverview{
		Cards: model.EventCards{
			TotalAlerts:     warning + errCount,
			TotalEvents:     len(rows),
			Warning:         warning,
			Info:            info,
			Error:           errCount,
			CategoriesCount: len(categories),
			KindsCount:      len(kinds),
		},
		Rows: rows,
	}
}

// mapSeverity 映射 K8s Event Type 到 severity
func mapSeverity(eventType string) string {
	switch eventType {
	case "Warning":
		return "warning"
	case "Error":
		return "error"
	default:
		return "info"
	}
}

// formatTime 格式化 time.Time 为 RFC3339 字符串
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
