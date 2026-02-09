// Package sdk 封装外部客户端
//
// types.go - SDK 层公共类型定义
//
// 本文件定义 SDK 层使用的各种选项和数据类型。
// 这些类型被接口方法使用，也被实现层使用。
package sdk

// =============================================================================
// 查询选项
// =============================================================================

// ListOptions 列表查询选项
type ListOptions struct {
	LabelSelector string // 标签选择器，如 "app=nginx"
	FieldSelector string // 字段选择器，如 "status.phase=Running"
	Limit         int64  // 限制返回数量
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	GracePeriodSeconds *int64 // 优雅终止时间 (秒)
	Force              bool   // 是否强制删除
}

// LogOptions 日志选项
type LogOptions struct {
	Container    string // 容器名称 (多容器 Pod 需指定)
	TailLines    int64  // 返回最后 N 行
	SinceSeconds int64  // 返回最近 N 秒的日志
	Timestamps   bool   // 是否包含时间戳
	Previous     bool   // 是否获取之前容器的日志
}

// =============================================================================
// 资源标识
// =============================================================================

// GroupVersionKind 资源类型标识
type GroupVersionKind struct {
	Group   string // API 组，如 "apps"
	Version string // API 版本，如 "v1"
	Kind    string // 资源类型，如 "Deployment"
}

// =============================================================================
// 动态请求/响应
// =============================================================================

// DynamicRequest 动态请求 (仅 GET)
type DynamicRequest struct {
	Path  string            // API 路径
	Query map[string]string // 查询参数
}

// DynamicResponse 动态响应
type DynamicResponse struct {
	StatusCode int    // HTTP 状态码
	Body       []byte // 响应体
}

// =============================================================================
// Metrics 数据
// =============================================================================

// NodeMetrics Node 资源使用量
//
// 来自 metrics-server，包含 CPU 和内存的实时使用量
type NodeMetrics struct {
	CPU    string // CPU 使用量，如 "2300m" (2.3核)
	Memory string // 内存使用量，如 "18534Mi"
}

// PodMetrics Pod 资源使用量
//
// 来自 metrics-server，包含各容器的 CPU 和内存使用量
type PodMetrics struct {
	Namespace  string             // Pod 命名空间
	Name       string             // Pod 名称
	Containers []ContainerMetrics // 各容器的资源使用量
}

// ContainerMetrics 容器资源使用量
type ContainerMetrics struct {
	Name   string // 容器名称
	CPU    string // CPU 使用量，如 "100m"
	Memory string // 内存使用量，如 "128Mi"
}

// =============================================================================
// OTel Collector 原始指标（SDK 内部，不暴露给 Master）
// =============================================================================

// OTelRawMetrics OTel 采集的原始指标（per-pod 级别）
type OTelRawMetrics struct {
	// Linkerd 请求计数 (otel_response_total)
	LinkerdResponses []LinkerdResponseMetric

	// Linkerd 延迟 (otel_response_latency_ms_bucket/sum/count)
	LinkerdLatencyBuckets []LinkerdLatencyBucketMetric
	LinkerdLatencySums    []LinkerdLatencySumMetric
	LinkerdLatencyCounts  []LinkerdLatencyCountMetric

	// 入口请求计数（Controller 无关，Parser 归一化后）
	IngressRequests []IngressRequestMetric

	// 入口延迟（Controller 无关，Parser 归一化后，单位: 秒）
	IngressLatencyBuckets []IngressLatencyBucketMetric
	IngressLatencySums    []IngressLatencySumMetric
	IngressLatencyCounts  []IngressLatencyCountMetric
}

// ---- Linkerd 类型 ----

// LinkerdResponseMetric otel_response_total 单条指标
type LinkerdResponseMetric struct {
	Namespace      string  // 源 pod 所在 namespace
	Deployment     string  // 源 deployment
	Pod            string  // 源 pod name
	Direction      string  // "inbound" / "outbound"
	StatusCode     string  // "200", "503"
	Classification string  // "success" / "failure"
	RouteName      string  // "default" / "probe"
	SrvPort        string  // 业务端口 "8200" / admin "4191"
	DstNamespace   string  // outbound: 目标 namespace
	DstDeployment  string  // outbound: 目标 deployment
	TLS            string  // "true" / "false"（inbound 的 mTLS 状态）
	Value          float64 // 累积值
}

// LinkerdLatencyBucketMetric otel_response_latency_ms_bucket 单条
type LinkerdLatencyBucketMetric struct {
	Namespace  string
	Deployment string
	Pod        string
	Direction  string
	Le         string  // bucket 边界 (ms): "1", "5", "100", "+Inf"
	Value      float64
}

// LinkerdLatencySumMetric otel_response_latency_ms_sum 单条
type LinkerdLatencySumMetric struct {
	Namespace      string
	Deployment     string
	Pod            string
	Direction      string
	DstNamespace   string // outbound 时的目标 namespace
	DstDeployment  string // outbound 时的目标 deployment
	Value          float64 // 毫秒
}

// LinkerdLatencyCountMetric otel_response_latency_ms_count 单条
type LinkerdLatencyCountMetric struct {
	Namespace      string
	Deployment     string
	Pod            string
	Direction      string
	DstNamespace   string // outbound 时的目标 namespace
	DstDeployment  string // outbound 时的目标 deployment
	Value          float64
}

// ---- 入口类型（Controller 无关） ----
//
// Parser 将不同 Controller 的原始指标归一化到以下通用结构。
// ServiceKey 统一为 "namespace-service-port" 格式。
//
// 归一化映射:
//   Traefik: service="ns-svc-port@kubernetes"  → ServiceKey="ns-svc-port"（去 @kubernetes 后缀）
//   Nginx:   namespace="ns", service="svc", service_port="port"
//            → ServiceKey="ns-svc-port"

// IngressRequestMetric 入口请求计数指标（归一化后）
type IngressRequestMetric struct {
	ServiceKey string  // 标准化: "namespace-service-port"
	Code       string  // "200"
	Method     string  // "GET"
	Value      float64 // 累积值
}

// IngressLatencyBucketMetric 入口延迟桶指标（归一化后）
type IngressLatencyBucketMetric struct {
	ServiceKey string
	Le         string  // bucket 边界 (秒): "0.1", "0.3", "5", "+Inf"
	Value      float64
}

// IngressLatencySumMetric 入口延迟总和指标（归一化后）
type IngressLatencySumMetric struct {
	ServiceKey string
	Value      float64 // 秒
}

// IngressLatencyCountMetric 入口延迟计数指标（归一化后）
type IngressLatencyCountMetric struct {
	ServiceKey string
	Value      float64
}

// =============================================================================
// IngressRoute 路由信息（保留，用于路由映射采集）
// =============================================================================

// IngressRouteInfo IngressRoute 解析后的路由信息
//
// 用于建立 service 标识与实际域名/路径的映射关系。
// 例如: ServiceKey "default-nginx-80" → Domain "api.example.com", PathPrefix "/v1"
type IngressRouteInfo struct {
	Name        string // IngressRoute/Ingress 名称
	Namespace   string // 命名空间
	Domain      string // 域名 (从 Host() 规则解析)
	PathPrefix  string // 路径前缀 (从 PathPrefix() 规则解析)
	ServiceKey  string // 标准化: "namespace-service-port"
	ServiceName string // K8s Service 名称
	ServicePort int    // K8s Service 端口
	TLS         bool   // 是否启用 TLS
}
