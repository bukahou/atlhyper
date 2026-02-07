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
// Ingress Controller 指标数据
// =============================================================================

// IngressMetrics Ingress Controller 采集的原始指标
//
// 从 Prometheus 端点解析出的 Counter 和 Histogram 指标。
// SDK 层只负责 HTTP 采集和 Prometheus 文本解析，
// 业务处理 (增量计算、聚合) 在 Repository 层完成。
type IngressMetrics struct {
	Timestamp  int64                    // Unix 时间戳（秒）
	Counters   []IngressCounterMetric   // Counter 指标（累计值）
	Histograms []IngressHistogramMetric // Histogram 指标
}

// IngressCounterMetric Counter 类型指标
type IngressCounterMetric struct {
	Host       string // 域名或 Traefik service key
	Status     string // HTTP 状态码
	MetricType string // 指标类型: requests / errors
	Value      int64  // 累计值
}

// IngressHistogramMetric Histogram 类型指标
type IngressHistogramMetric struct {
	Host    string           // 域名或 Traefik service key
	Buckets map[string]int64 // 桶数据 (le 字符串 -> count)
	Sum     float64          // 总和（秒）
	Count   int64            // 计数
}

// IngressRouteInfo IngressRoute 解析后的路由信息
//
// 用于建立 Traefik service 名称与实际域名/路径的映射关系。
// 例如: ServiceKey "default-nginx-80@kubernetes" → Domain "api.example.com", PathPrefix "/v1"
type IngressRouteInfo struct {
	Name        string // IngressRoute/Ingress 名称
	Namespace   string // 命名空间
	Domain      string // 域名 (从 Host() 规则解析)
	PathPrefix  string // 路径前缀 (从 PathPrefix() 规则解析)
	ServiceKey  string // Traefik service 标识 (如 namespace-service-port@kubernetes)
	ServiceName string // K8s Service 名称
	ServicePort int    // K8s Service 端口
	TLS         bool   // 是否启用 TLS
}
