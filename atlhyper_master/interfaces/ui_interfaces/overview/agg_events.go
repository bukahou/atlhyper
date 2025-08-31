package overview

import (
	"sort"
	"strings"
	"time"

	event "AtlHyper/model/event"
)

// 24h 内各级别总计
func buildSeverityTotals(events []event.LogEvent, since, until time.Time) SeverityTotals {
	var st SeverityTotals
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
func buildRecentAlerts(events []event.LogEvent, limit int) []event.LogEvent {
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
func filterEventsByWindow(events []event.LogEvent, since, until time.Time) []event.LogEvent {
	out := make([]event.LogEvent, 0, len(events))
	for _, e := range events {
		t := e.Timestamp.UTC()
		if !t.Before(since) && t.Before(until) {
			out = append(out, e)
		}
	}
	return out
}
