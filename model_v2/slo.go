package model_v2

// ============================================================
// SLO 指标数据
// Agent 采集 Ingress Controller 的 Prometheus 指标并上报
// ============================================================

// IngressMetrics Ingress 指标数据
// Agent 采集后上报给 Master
type IngressMetrics struct {
	Timestamp int64 `json:"timestamp"` // Unix 时间戳（秒）

	// Counter 指标（累计值）
	Counters []IngressCounterMetric `json:"counters,omitempty"`

	// Histogram 指标
	Histograms []IngressHistogramMetric `json:"histograms,omitempty"`
}

// IngressCounterMetric Counter 类型指标
// 用于统计请求数、错误数等累计值
type IngressCounterMetric struct {
	Host         string `json:"host"`                    // 域名
	IngressName  string `json:"ingress_name,omitempty"`  // Ingress 名称
	IngressClass string `json:"ingress_class,omitempty"` // Ingress Class
	Namespace    string `json:"namespace,omitempty"`     // 命名空间
	Service      string `json:"service,omitempty"`       // 后端 Service
	TLS          bool   `json:"tls,omitempty"`           // 是否 TLS
	Method       string `json:"method,omitempty"`        // HTTP 方法
	Status       string `json:"status"`                  // HTTP 状态码
	MetricType   string `json:"metric_type"`             // 指标类型: requests / errors
	Value        int64  `json:"value"`                   // 累计值
}

// IngressHistogramMetric Histogram 类型指标
// 用于统计延迟分布
type IngressHistogramMetric struct {
	Host        string            `json:"host"`                   // 域名
	IngressName string            `json:"ingress_name,omitempty"` // Ingress 名称
	Namespace   string            `json:"namespace,omitempty"`    // 命名空间
	Buckets     map[string]int64  `json:"buckets"`                // 桶数据 (le字符串 -> count)
	Sum         float64           `json:"sum"`                    // 总和（秒）
	Count       int64             `json:"count"`                  // 计数
}

// SLOPushRequest Agent 推送 SLO 指标的请求
type SLOPushRequest struct {
	ClusterID     string             `json:"cluster_id"`
	Metrics       IngressMetrics     `json:"metrics"`
	IngressRoutes []IngressRouteInfo `json:"ingress_routes,omitempty"` // IngressRoute 映射信息
}

// ============================================================
// SLO 快照数据
// Agent 采集后嵌入 ClusterSnapshot 一起上报
// ============================================================

// SLOSnapshot SLO 快照数据
// 包含 Ingress 指标和路由映射，嵌入 ClusterSnapshot 统一推送
type SLOSnapshot struct {
	Metrics IngressMetrics   `json:"metrics"`               // 指标数据
	Routes  []IngressRouteInfo `json:"routes,omitempty"`    // 路由映射
}

// ============================================================
// IngressRoute 映射信息
// 用于将 Traefik service 名称映射到实际的域名和路径
// ============================================================

// IngressRouteInfo IngressRoute 解析后的信息
// 用于建立 Traefik service 名称与域名/路径的映射关系
type IngressRouteInfo struct {
	Name        string `json:"name"`         // IngressRoute 名称
	Namespace   string `json:"namespace"`    // 命名空间
	Domain      string `json:"domain"`       // 域名 (从 Host() 规则解析)
	PathPrefix  string `json:"path_prefix"`  // 路径前缀 (从 PathPrefix() 规则解析)
	ServiceKey  string `json:"service_key"`  // Traefik service 标识 (如 namespace-service-port@kubernetes)
	ServiceName string `json:"service_name"` // K8s Service 名称
	ServicePort int    `json:"service_port"` // K8s Service 端口
	TLS         bool   `json:"tls"`          // 是否启用 TLS
}
