package command

import (
	"errors"
	"fmt"
	"time"
)

// 查询类指令（链路 / 日志 / 指标）的共享参数。
//
// Master 用 ToParams 生成、发送前 Validate；Agent 用 ParseParams 解析、执行前再 Validate。
// 键名沿用既有的 snake_case（线上形状不变）；类型不对时解析失败，⛔ 不再按零值继续。
// 时间：since 为 Go 时长（如 15m）；start_time / end_time 为 RFC 3339，须成对出现，给了就优先于 since。

// QueryParams 查询类指令参数的共同行为
type QueryParams interface {
	ToParams() map[string]any
	Validate() error
}

// 链路查询的子动作
const (
	TraceListTraces     = "list_traces" // 默认
	TraceListServices   = "list_services"
	TraceCorrelations   = "get_correlations"
	TraceTopology       = "get_topology"
	TraceListOperations = "list_operations"
	TraceHTTPStats      = "http_stats"
	TraceDBStats        = "db_stats"
	TraceServiceSeries  = "service_series"
)

// 日志查询的子动作
const (
	LogsList      = ""          // 默认：日志列表
	LogsHistogram = "histogram" // 按时间分桶的计数
)

// 指标查询的子动作
const (
	MetricsSummary    = "get_summary" // 默认
	MetricsListAll    = "list_all"
	MetricsGetNode    = "get_node"
	MetricsGetSeries  = "get_series"
	MetricsGetHistory = "get_history"
)

// 取值范围（Agent 侧仓库也按这些上限截断）
const (
	MaxTraceLimit = 2000
	MaxLogLimit   = 500
	maxIDLength   = 128
)

// TraceSortDurationDesc 链路列表按耗时倒序；不填按时间倒序
const TraceSortDurationDesc = "duration_desc"

var (
	traceSubActions   = setOf(TraceListTraces, TraceListServices, TraceCorrelations, TraceTopology, TraceListOperations, TraceHTTPStats, TraceDBStats, TraceServiceSeries)
	tracesNeedService = setOf(TraceHTTPStats, TraceDBStats, TraceServiceSeries)
	metricsSubActions = setOf(MetricsSummary, MetricsListAll, MetricsGetNode, MetricsGetSeries, MetricsGetHistory)
	correlationModes  = setOf("", "failure", "latency")
	traceSorts        = setOf("", TraceSortDurationDesc)
)

func setOf(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

// TimeWindow 查询的时间范围
type TimeWindow struct {
	Since     string `json:"since,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

// SinceOr since 换成时长；没给时用 def（Validate 已保证格式正确）
func (w TimeWindow) SinceOr(def time.Duration) time.Duration {
	if d, err := time.ParseDuration(w.Since); err == nil && d > 0 {
		return d
	}
	return def
}

func (w TimeWindow) validate() error {
	if w.Since != "" {
		if d, err := time.ParseDuration(w.Since); err != nil || d <= 0 {
			return fmt.Errorf("since 须为正的时长（如 15m），收到 %q", w.Since)
		}
	}
	if (w.StartTime == "") != (w.EndTime == "") {
		return errors.New("start_time 与 end_time 须成对给出")
	}
	if w.StartTime == "" {
		return nil
	}
	start, err := time.Parse(time.RFC3339Nano, w.StartTime)
	if err != nil {
		return fmt.Errorf("start_time 须为 RFC 3339 时间，收到 %q", w.StartTime)
	}
	end, err := time.Parse(time.RFC3339Nano, w.EndTime)
	if err != nil {
		return fmt.Errorf("end_time 须为 RFC 3339 时间，收到 %q", w.EndTime)
	}
	if !start.Before(end) {
		return errors.New("start_time 须早于 end_time")
	}
	return nil
}

// QueryTracesParams query_traces 指令的参数
type QueryTracesParams struct {
	SubAction string `json:"sub_action,omitempty"` // 见 Trace* 常量；空 = list_traces
	TimeWindow
	Service       string  `json:"service,omitempty"`
	Operation     string  `json:"operation,omitempty"`
	MinDurationMs float64 `json:"min_duration_ms,omitempty"`
	Limit         int     `json:"limit,omitempty"`
	Sort          string  `json:"sort,omitempty"` // 空 = 按时间倒序；duration_desc = 按耗时倒序
	StatusCode    string  `json:"status_code,omitempty"`
	Method        string  `json:"method,omitempty"`
	Mode          string  `json:"mode,omitempty"` // get_correlations：failure（默认）/ latency
}

// Action 子动作（空按默认）
func (p QueryTracesParams) Action() string {
	if p.SubAction == "" {
		return TraceListTraces
	}
	return p.SubAction
}

// ToParams 转成 Command.Params
func (p QueryTracesParams) ToParams() map[string]any { return toParams(p) }

// Validate 子动作、各子动作的必填项与取值范围
func (p QueryTracesParams) Validate() error {
	action := p.Action()
	if !traceSubActions[action] {
		return fmt.Errorf("未知的链路查询 sub_action: %q", p.SubAction)
	}
	if tracesNeedService[action] && p.Service == "" {
		return fmt.Errorf("%s 须给出 service", action)
	}
	if p.Limit < 0 || p.Limit > MaxTraceLimit {
		return fmt.Errorf("limit 须在 0–%d 之间，收到 %d", MaxTraceLimit, p.Limit)
	}
	if p.MinDurationMs < 0 {
		return errors.New("min_duration_ms 不能为负")
	}
	if !correlationModes[p.Mode] {
		return fmt.Errorf("未知的 mode: %q", p.Mode)
	}
	if !traceSorts[p.Sort] {
		return fmt.Errorf("未知的 sort: %q", p.Sort)
	}
	return p.TimeWindow.validate()
}

// QueryTraceDetailParams query_trace_detail 指令的参数
type QueryTraceDetailParams struct {
	TraceID string `json:"trace_id"`
}

// ToParams 转成 Command.Params
func (p QueryTraceDetailParams) ToParams() map[string]any { return toParams(p) }

// Validate trace_id 必填
func (p QueryTraceDetailParams) Validate() error {
	if p.TraceID == "" || len(p.TraceID) > maxIDLength {
		return errors.New("trace_id 必填且不超过 128 个字符")
	}
	return nil
}

// QueryLogsParams query_logs 指令的参数
type QueryLogsParams struct {
	SubAction string `json:"sub_action,omitempty"` // 空 = 日志列表；histogram = 直方图
	TimeWindow
	Query   string `json:"query,omitempty"`
	Service string `json:"service,omitempty"`
	Level   string `json:"level,omitempty"`
	Scope   string `json:"scope,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

// ToParams 转成 Command.Params
func (p QueryLogsParams) ToParams() map[string]any { return toParams(p) }

// Validate 子动作与取值范围
func (p QueryLogsParams) Validate() error {
	if p.SubAction != LogsList && p.SubAction != LogsHistogram {
		return fmt.Errorf("未知的日志查询 sub_action: %q", p.SubAction)
	}
	if p.Limit < 0 || p.Limit > MaxLogLimit {
		return fmt.Errorf("limit 须在 0–%d 之间，收到 %d", MaxLogLimit, p.Limit)
	}
	if p.Offset < 0 {
		return errors.New("offset 不能为负")
	}
	if len(p.TraceID) > maxIDLength || len(p.SpanID) > maxIDLength {
		return errors.New("trace_id / span_id 不超过 128 个字符")
	}
	return p.TimeWindow.validate()
}

// QueryMetricsParams query_metrics 指令的参数
type QueryMetricsParams struct {
	SubAction string `json:"sub_action,omitempty"` // 见 Metrics* 常量；空 = get_summary
	NodeName  string `json:"node_name,omitempty"`
	Metric    string `json:"metric,omitempty"`
	Since     string `json:"since,omitempty"`
}

// Action 子动作（空按默认）
func (p QueryMetricsParams) Action() string {
	if p.SubAction == "" {
		return MetricsSummary
	}
	return p.SubAction
}

// SinceOr since 换成时长；没给时用 def
func (p QueryMetricsParams) SinceOr(def time.Duration) time.Duration {
	return TimeWindow{Since: p.Since}.SinceOr(def)
}

// ToParams 转成 Command.Params
func (p QueryMetricsParams) ToParams() map[string]any { return toParams(p) }

// Validate 子动作与各子动作的必填项
func (p QueryMetricsParams) Validate() error {
	action := p.Action()
	if !metricsSubActions[action] {
		return fmt.Errorf("未知的指标查询 sub_action: %q", p.SubAction)
	}
	switch action {
	case MetricsGetNode, MetricsGetHistory:
		if p.NodeName == "" {
			return fmt.Errorf("%s 须给出 node_name", action)
		}
	case MetricsGetSeries:
		if p.NodeName == "" || p.Metric == "" {
			return fmt.Errorf("%s 须给出 node_name 与 metric", action)
		}
	}
	return TimeWindow{Since: p.Since}.validate()
}
