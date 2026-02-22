// Package slo 定义 SLO 数据模型
//
// 数据源: Traefik (otel_metrics_sum + otel_metrics_histogram)
//         Linkerd (otel_metrics_gauge)
package slo

import "time"

// ============================================================
// IngressSLO — Traefik 入口服务级指标
// ============================================================

type IngressSLO struct {
	ServiceKey    string            `json:"serviceKey"`
	DisplayName   string            `json:"displayName"`
	RPS           float64           `json:"rps"`
	SuccessRate   float64           `json:"successRate"`
	ErrorRate     float64           `json:"errorRate"`
	P50Ms         float64           `json:"p50Ms"`
	P90Ms         float64           `json:"p90Ms"`
	P99Ms         float64           `json:"p99Ms"`
	AvgMs         float64           `json:"avgMs"`
	StatusCodes   []StatusCodeCount `json:"statusCodes"`
	TotalRequests int64             `json:"totalRequests"`
	TotalErrors   int64             `json:"totalErrors"`
}

// StatusCodeCount HTTP 状态码分布
type StatusCodeCount struct {
	Code  string `json:"code"`
	Count int64  `json:"count"`
}

// ============================================================
// ServiceSLO — Linkerd 服务网格指标
// ============================================================

type ServiceSLO struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	RPS         float64           `json:"rps"`
	SuccessRate float64           `json:"successRate"`
	P50Ms       float64           `json:"p50Ms"`
	P90Ms       float64           `json:"p90Ms"`
	P99Ms       float64           `json:"p99Ms"`
	MTLSRate    float64           `json:"mtlsRate"`
	StatusCodes []StatusCodeCount `json:"statusCodes"`
}

// ============================================================
// ServiceEdge — 服务间调用关系（Linkerd outbound）
// ============================================================

type ServiceEdge struct {
	SrcNamespace string  `json:"srcNamespace"`
	SrcName      string  `json:"srcName"`
	DstNamespace string  `json:"dstNamespace"`
	DstName      string  `json:"dstName"`
	RPS          float64 `json:"rps"`
	SuccessRate  float64 `json:"successRate"`
	AvgMs        float64 `json:"avgMs"`
}

// ============================================================
// SLO 时序数据
// ============================================================

type DataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	RPS         float64   `json:"rps"`
	SuccessRate float64   `json:"successRate"`
	P50Ms       float64   `json:"p50Ms"`
	P99Ms       float64   `json:"p99Ms"`
}

type TimeSeries struct {
	Namespace string      `json:"namespace,omitempty"`
	Name      string      `json:"name"`
	Points    []DataPoint `json:"points"`
}

// ============================================================
// SLOSummary — 仪表盘摘要
// ============================================================

type SLOSummary struct {
	TotalServices    int     `json:"totalServices"`
	HealthyServices  int     `json:"healthyServices"`
	WarningServices  int     `json:"warningServices"`
	CriticalServices int     `json:"criticalServices"`
	AvgSuccessRate   float64 `json:"avgSuccessRate"`
	TotalRPS         float64 `json:"totalRps"`
	AvgP99Ms         float64 `json:"avgP99Ms"`
}
