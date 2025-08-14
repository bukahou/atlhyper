package cluster

import (
	"context"
	"sort"
	"strings"
	"time"

	"NeuroController/external/logger"
)

// ===== 轻量表格行 =====
type RecentAlertRow struct {
	Time      string `json:"time"`
	Severity  string `json:"severity"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Node      string `json:"node"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}

// ===== 趋势点（仅三条线）=====
// TS 表示“该小时桶的起始整点（本地时区）”的 epoch 毫秒
type AlertTrendPoint struct {
	TS       int64 `json:"ts"`
	Critical int   `json:"critical"`
	Warning  int   `json:"warning"`
	Info     int   `json:"info"`
}

// ===== 趋势序列 =====
// From/To 也采用本地时区整点（对外是 ISO 时间串）
type AlertTrendsSeries struct {
	From        time.Time         `json:"from"`         // 本地整点起始
	To          time.Time         `json:"to"`           // 本地整点结束（不含）
	StepMinutes int               `json:"step_minutes"` // 固定 60
	Series      []AlertTrendPoint `json:"series"`       // 固定 24 个点
}

// BuildAlertsViewData
// 返回：一天内事件总数、Recent 表格（默认前 10 条）、一天趋势（本地时区整点分桶，24 组）
func BuildAlertsViewData(ctx context.Context, recentLimit int) (dayTotal int, recent []RecentAlertRow, trends AlertTrendsSeries, err error) {
	// 1) 拉取最近 1 天日志
	logs, err := logger.GetRecentEventLogs(1)
	if err != nil {
		return 0, nil, AlertTrendsSeries{}, err
	}
	dayTotal = len(logs)

	// 2) Recent 表格（按时间倒序 + 截断）
	if recentLimit <= 0 {
		recentLimit = 10
	}
	parseWhen := func(s string) (time.Time, bool) {
		if t, e := time.Parse(time.RFC3339Nano, s); e == nil {
			return t, true
		}
		if t, e := time.Parse(time.RFC3339, s); e == nil {
			return t, true
		}
		return time.Time{}, false
	}
	sort.SliceStable(logs, func(i, j int) bool {
		ti, okI := parseWhen(logs[i].EventTime)
		tj, okJ := parseWhen(logs[j].EventTime)
		switch {
		case okI && okJ:
			return ti.After(tj)
		case okI && !okJ:
			return true
		case !okI && okJ:
			return false
		default:
			return false
		}
	})
	for _, e := range logs {
		recent = append(recent, RecentAlertRow{
			Time:      e.EventTime,
			Severity:  strings.ToLower(e.Severity),
			Kind:      e.Kind,
			Name:      e.Name,
			Namespace: e.Namespace,
			Node:      e.Node,
			Reason:    e.Reason,
			Message:   e.Message,
		})
		if len(recent) >= recentLimit {
			break
		}
	}

	// 3) 趋势：固定最近 24 小时，本地时区整点分桶（Step=1h），产出 24 组
	const (
		step        = time.Hour
		bucketCount = 24
	)
	loc := time.Now().Location()                       // 使用系统本地时区（含 +09:00 等）
	toLocal := time.Now().In(loc).Truncate(step).Add(step) // 对齐到“下一个本地整点”
	fromLocal := toLocal.Add(-time.Duration(bucketCount) * step)

	type bucket struct{ critical, warning, info int }
	buckets := make([]bucket, bucketCount)

	severityKey := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "crit", "critical", "error":
			return "critical"
		case "warn", "warning":
			return "warning"
		default:
			return "info"
		}
	}

	// 写入对应小时桶（以“本地时区”对齐再分桶）
	for _, e := range logs {
		t, ok := parseWhen(e.EventTime)
		if !ok {
			continue
		}
		tLocal := t.In(loc)
		if tLocal.Before(fromLocal) || !tLocal.Before(toLocal) {
			continue
		}
		idx := int(tLocal.Sub(fromLocal) / step) // 0..23
		if idx < 0 || idx >= bucketCount {
			continue
		}
		switch severityKey(e.Severity) {
		case "critical":
			buckets[idx].critical++
		case "warning":
			buckets[idx].warning++
		default:
			buckets[idx].info++
		}
	}

	// 组装固定 24 个点：TS=每个小时桶的“起始整点”（本地）
	series := make([]AlertTrendPoint, 0, bucketCount)
	for i := 0; i < bucketCount; i++ {
		start := fromLocal.Add(time.Duration(i) * step) // 小时开始整点（本地）
		b := buckets[i]
		series = append(series, AlertTrendPoint{
			TS:       start.UnixMilli(), // 前端用本地时区格式化，即可显示为 13:00, 14:00 ...
			Critical: b.critical,
			Warning:  b.warning,
			Info:     b.info,
		})
	}

	trends = AlertTrendsSeries{
		From:        fromLocal,
		To:          toLocal,
		StepMinutes: 60,
		Series:      series, // 恰好 24 个点
	}
	return
}
