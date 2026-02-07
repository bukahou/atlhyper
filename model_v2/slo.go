package model_v2

// ============================================================
// SLO 快照 — Agent 从 OTel Collector 采集后上报
// ============================================================

// SLOSnapshot SLO 快照数据
// 嵌入 ClusterSnapshot.SLOData，随集群快照统一上报
type SLOSnapshot struct {
	Timestamp int64 `json:"timestamp"` // Unix 时间戳（秒）

	// 服务级黄金指标（Linkerd inbound，增量，已 per-pod delta + service 聚合）
	Services []ServiceMetrics `json:"services,omitempty"`

	// 服务调用拓扑（Linkerd outbound，增量）
	Edges []ServiceEdge `json:"edges,omitempty"`

	// 入口指标（增量，Agent 已将秒转为毫秒，Controller 无关）
	Ingress []IngressMetrics `json:"ingress,omitempty"`

	// 路由映射（K8s IngressRoute CRD / 标准 Ingress，不变）
	Routes []IngressRouteInfo `json:"routes,omitempty"`
}

// ============================================================
// 服务级黄金指标
// ============================================================

// ServiceMetrics 单个服务的黄金指标（增量）
// 维度: namespace + name (deployment/daemonset/statefulset)
// 来源: Linkerd inbound otel_response_total + otel_response_latency_ms_*
type ServiceMetrics struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"` // workload name (deployment 等)

	// 请求计数增量（按 status_code 分组）
	Requests []RequestDelta `json:"requests"`

	// 延迟直方图增量（毫秒，已跨 Pod 聚合）
	LatencyBuckets map[string]int64 `json:"latency_buckets,omitempty"` // le(ms string) → delta count
	LatencySum     float64          `json:"latency_sum"`               // 总和（毫秒）
	LatencyCount   int64            `json:"latency_count"`

	// mTLS 覆盖率（从 inbound otel_response_total 的 tls 标签聚合）
	TLSRequestDelta   int64 `json:"tls_request_delta"`   // tls="true" 的请求增量
	TotalRequestDelta int64 `json:"total_request_delta"` // 总请求增量（tls=true + tls=false）
}

// RequestDelta 按状态码分组的请求增量
type RequestDelta struct {
	StatusCode     string `json:"status_code"`    // "200", "404", "503"
	Classification string `json:"classification"` // "success" / "failure" (来自 Linkerd)
	Delta          int64  `json:"delta"`
}

// ============================================================
// 服务拓扑
// ============================================================

// ServiceEdge 服务调用边（增量）
// 来源: Linkerd outbound otel_response_total + otel_response_latency_ms_*
type ServiceEdge struct {
	SrcNamespace string  `json:"src_ns"`
	SrcName      string  `json:"src_name"`
	DstNamespace string  `json:"dst_ns"`
	DstName      string  `json:"dst_name"`
	RequestDelta int64   `json:"request_delta"` // 增量请求数
	FailureDelta int64   `json:"failure_delta"` // 失败请求增量（classification=failure）
	LatencySum   float64 `json:"latency_sum"`   // 延迟总和 (ms)
	LatencyCount int64   `json:"latency_count"` // 延迟请求数
}

// ============================================================
// 入口指标（Ingress Controller 无关）
// ============================================================

// IngressMetrics 入口服务级指标（增量，Controller 无关）
// 维度: service_key（标准化格式: "namespace-service-port"）
//
// 支持的 Ingress Controller:
//   - Traefik: 从 otel_traefik_service_* 指标解析
//   - Nginx:   从 otel_nginx_ingress_controller_* 指标解析
//
// Parser 负责将不同 Controller 的指标归一化到此结构
type IngressMetrics struct {
	ServiceKey string `json:"service_key"` // 标准化: "namespace-service-port"

	// 请求计数增量（按 code + method 分组）
	Requests []IngressRequestDelta `json:"requests"`

	// 延迟直方图增量（毫秒，Agent 已将秒转为毫秒）
	LatencyBuckets map[string]int64 `json:"latency_buckets,omitempty"` // le(ms string) → delta count
	LatencySum     float64          `json:"latency_sum"`               // 毫秒
	LatencyCount   int64            `json:"latency_count"`
}

// IngressRequestDelta 入口请求增量（Controller 无关）
type IngressRequestDelta struct {
	Code   string `json:"code"`   // HTTP 状态码
	Method string `json:"method"` // HTTP 方法
	Delta  int64  `json:"delta"`
}

// ============================================================
// 路由映射
// ============================================================

// IngressRouteInfo IngressRoute 解析后的信息
// 用于建立 Traefik service 名称与域名/路径的映射关系
type IngressRouteInfo struct {
	Name        string `json:"name"`         // IngressRoute 名称
	Namespace   string `json:"namespace"`    // 命名空间
	Domain      string `json:"domain"`       // 域名 (从 Host() 规则解析)
	PathPrefix  string `json:"path_prefix"`  // 路径前缀 (从 PathPrefix() 规则解析)
	ServiceKey  string `json:"service_key"`  // 标准化: "namespace-service-port"
	ServiceName string `json:"service_name"` // K8s Service 名称
	ServicePort int    `json:"service_port"` // K8s Service 端口
	TLS         bool   `json:"tls"`          // 是否启用 TLS
}
