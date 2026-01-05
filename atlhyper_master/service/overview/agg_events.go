package overview

import (
	"sort"
	"strings"
	"time"

	"AtlHyper/atlhyper_master/model/ui"
	"AtlHyper/model/transport"
)

// 24h 内各级别总计
func buildSeverityTotals(events []transport.LogEvent, since, until time.Time) ui.SeverityTotals {
	var st ui.SeverityTotals
	for _, e := range events {
		t := e.Timestamp.UTC()
		if t.Before(since) || !t.Before(until) {
			continue
		}
		switch strings.ToLower(e.Severity) {
		case "critical":
			st.Critical++
		case "warning":
			st.Warning++
		default:
			st.Info++
		}
	}
	return st
}

// 最新 N 条（已在 24h 窗口内）
func buildRecentAlerts(events []transport.LogEvent, limit int) []transport.LogEvent {
	// 按时间倒序
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	if limit > 0 && len(events) > limit {
		return events[:limit]
	}
	return events
}

// 过滤 24h 窗口（或任意窗口）
func filterEventsByWindow(events []transport.LogEvent, since, until time.Time) []transport.LogEvent {
	out := make([]transport.LogEvent, 0, len(events))
	for _, e := range events {
		t := e.Timestamp.UTC()
		if !t.Before(since) && t.Before(until) {
			out = append(out, e)
		}
	}
	return out
}
