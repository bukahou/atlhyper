// Package slo SLO 数据仓库
//
// types.go - 内部增量类型定义
//
// 定义 per-pod delta 计算后的中间数据结构。
// 这些类型仅在 Repository 内部使用，不暴露给上层。
//
// 数据流: OTelRawMetrics (累积) → filter → per-pod delta → *Delta 类型 → aggregate → model_v2.*
package slo

// =============================================================================
// Linkerd 增量类型
// =============================================================================

// linkerdResponseDelta otel_response_total 的 per-pod 增量
type linkerdResponseDelta struct {
	Namespace      string
	Deployment     string
	Pod            string
	Direction      string // "inbound" / "outbound"
	StatusCode     string
	Classification string // "success" / "failure"
	TLS            string // "true" / "false" (inbound only)
	DstNamespace   string // outbound only
	DstDeployment  string // outbound only
	Delta          float64
}

// linkerdBucketDelta otel_response_latency_ms_bucket 的 per-pod 增量
type linkerdBucketDelta struct {
	Namespace  string
	Deployment string
	Pod        string
	Direction  string
	Le         string // bucket 边界 (ms)
	Delta      float64
}

// linkerdSumDelta otel_response_latency_ms_sum 的 per-pod 增量
type linkerdSumDelta struct {
	Namespace  string
	Deployment string
	Pod        string
	Direction  string
	Delta      float64 // 毫秒
}

// linkerdCountDelta otel_response_latency_ms_count 的 per-pod 增量
type linkerdCountDelta struct {
	Namespace  string
	Deployment string
	Pod        string
	Direction  string
	Delta      float64
}

// =============================================================================
// Ingress 增量类型（Controller 无关，已归一化）
// =============================================================================

// ingressRequestDelta 入口请求计数增量
type ingressRequestDelta struct {
	ServiceKey string
	Code       string
	Method     string
	Delta      float64
}

// ingressBucketDelta 入口延迟桶增量
type ingressBucketDelta struct {
	ServiceKey string
	Le         string // 秒（aggregate 阶段转为毫秒）
	Delta      float64
}

// ingressSumDelta 入口延迟总和增量
type ingressSumDelta struct {
	ServiceKey string
	Delta      float64 // 秒（aggregate 阶段转为毫秒）
}

// ingressCountDelta 入口延迟计数增量
type ingressCountDelta struct {
	ServiceKey string
	Delta      float64
}
