// atlhyper_master_v2/model/slo.go
// SLO API 类型定义
package model

// ==================== API 响应类型 ====================

// DomainSLO 域名 SLO 信息
type DomainSLO struct {
	Host         string                    `json:"host"`
	IngressName  string                    `json:"ingress_name"`
	IngressClass string                    `json:"ingress_class"`
	Namespace    string                    `json:"namespace"`
	TLS          bool                      `json:"tls"`
	Targets      map[string]*SLOTargetSpec `json:"targets"` // "1d", "7d", "30d"
	Current      *SLOMetrics               `json:"current"`
	Previous     *SLOMetrics               `json:"previous,omitempty"`
	ErrorBudget  float64                   `json:"error_budget_remaining"`
	Status       string                    `json:"status"` // healthy / warning / critical
	Trend        string                    `json:"trend"`  // up / down / stable
}

// SLOTargetSpec SLO 目标规格
type SLOTargetSpec struct {
	Availability float64 `json:"availability"`
	P95Latency   int     `json:"p95_latency"`
}

// SLOMetrics SLO 指标
type SLOMetrics struct {
	Availability   float64 `json:"availability"`
	P95Latency     int     `json:"p95_latency"`
	P99Latency     int     `json:"p99_latency"`
	ErrorRate      float64 `json:"error_rate"`
	RequestsPerSec float64 `json:"requests_per_sec"`
	TotalRequests  int64   `json:"total_requests"`
}

// SLOSummary SLO 汇总信息
type SLOSummary struct {
	TotalServices   int     `json:"total_services"`
	TotalDomains    int     `json:"total_domains"`
	HealthyCount    int     `json:"healthy_count"`
	WarningCount    int     `json:"warning_count"`
	CriticalCount   int     `json:"critical_count"`
	AvgAvailability float64 `json:"avg_availability"`
	AvgErrorBudget  float64 `json:"avg_error_budget"`
	TotalRPS        float64 `json:"total_rps"`
}

// SLODomainsResponse 域名列表响应 (V1 兼容，使用 host/service key)
type SLODomainsResponse struct {
	Domains []DomainSLO `json:"domains"`
	Summary SLOSummary  `json:"summary"`
}

// ==================== V2 API 响应类型（按真实域名分组）====================

// DomainSLOResponseV2 域名级别的 SLO 响应 (V2)
// 以真实域名为单位，包含该域名下的所有后端服务
type DomainSLOResponseV2 struct {
	Domain               string                    `json:"domain"`                 // 真实域名（如 example.com）
	TLS                  bool                      `json:"tls"`                    // 是否启用 TLS
	Services             []ServiceSLO              `json:"services"`               // 该域名下的所有后端服务
	Summary              *SLOMetrics               `json:"summary"`                // 域名级别汇总指标
	Targets              map[string]*SLOTargetSpec  `json:"targets,omitempty"`      // 目标配置 ("1d"/"7d"/"30d")
	Status               string                    `json:"status"`                 // healthy / warning / critical
	ErrorBudgetRemaining float64                   `json:"error_budget_remaining"` // 剩余错误预算
}

// ServiceSLO 后端服务级别的 SLO 数据（Metrics 的实际数据来源）
type ServiceSLO struct {
	ServiceKey   string                    `json:"service_key"`            // Traefik service key (namespace-name-port@kubernetes)
	ServiceName  string                    `json:"service_name"`           // 服务名称
	ServicePort  int                       `json:"service_port"`           // 服务端口
	Namespace    string                    `json:"namespace"`              // 命名空间
	Paths        []string                  `json:"paths"`                  // 使用该服务的路径列表
	IngressName  string                    `json:"ingress_name"`           // IngressRoute/Ingress 名称
	Current      *SLOMetrics               `json:"current"`                // 当前周期指标
	Previous     *SLOMetrics               `json:"previous,omitempty"`     // 上一周期指标（用于对比）
	Targets      map[string]*SLOTargetSpec `json:"targets,omitempty"`      // 目标配置
	Status       string                    `json:"status"`                 // healthy / warning / critical
	ErrorBudget  float64                   `json:"error_budget_remaining"` // 剩余错误预算
}

// SLODomainsResponseV2 域名列表响应 (V2)
type SLODomainsResponseV2 struct {
	Domains []DomainSLOResponseV2 `json:"domains"`
	Summary SLOSummary            `json:"summary"`
}

// SLODomainHistoryItem 域名历史数据项
type SLODomainHistoryItem struct {
	Timestamp    string  `json:"timestamp"`
	Availability float64 `json:"availability"`
	P95Latency   int     `json:"p95_latency"`
	P99Latency   int     `json:"p99_latency"`
	RPS          float64 `json:"rps"`
	ErrorRate    float64 `json:"error_rate"`
}

// SLODomainHistoryResponse 域名历史响应
type SLODomainHistoryResponse struct {
	Host    string                 `json:"host"`
	History []SLODomainHistoryItem `json:"history"`
}

// SLOStatusHistoryItem 状态变更历史项
type SLOStatusHistoryItem struct {
	Host                 string  `json:"host"`
	TimeRange            string  `json:"time_range"`
	OldStatus            string  `json:"old_status"`
	NewStatus            string  `json:"new_status"`
	Availability         float64 `json:"availability"`
	P95Latency           int     `json:"p95_latency"`
	ErrorBudgetRemaining float64 `json:"error_budget_remaining"`
	ChangedAt            string  `json:"changed_at"`
}

// ==================== 延迟分布 API 类型 ====================

// LatencyBucket 延迟分布桶
type LatencyBucket struct {
	LE    float64 `json:"le"`    // 上界 (ms)
	Count int64   `json:"count"` // 该桶内的请求数
}

// MethodBreakdown HTTP 方法分布
type MethodBreakdown struct {
	Method string `json:"method"` // GET, POST, PUT, DELETE, OTHER
	Count  int64  `json:"count"`
}

// StatusCodeBreakdown 状态码分布
type StatusCodeBreakdown struct {
	Code  string `json:"code"`  // "2xx", "3xx", "4xx", "5xx"
	Count int64  `json:"count"`
}

// LatencyDistributionResponse 延迟分布响应
type LatencyDistributionResponse struct {
	Domain        string                `json:"domain"`
	TotalRequests int64                 `json:"total_requests"`
	P50LatencyMs  int                   `json:"p50_latency_ms"`
	P95LatencyMs  int                   `json:"p95_latency_ms"`
	P99LatencyMs  int                   `json:"p99_latency_ms"`
	AvgLatencyMs  int                   `json:"avg_latency_ms"`
	Buckets       []LatencyBucket       `json:"buckets"`
	Methods       []MethodBreakdown     `json:"methods"`
	StatusCodes   []StatusCodeBreakdown `json:"status_codes"`
}

// ==================== API 请求类型 ====================

// UpdateSLOTargetRequest 更新 SLO 目标请求
type UpdateSLOTargetRequest struct {
	ClusterID          string  `json:"cluster_id"`
	Host               string  `json:"host"`
	TimeRange          string  `json:"time_range"` // "1d", "7d", "30d"
	AvailabilityTarget float64 `json:"availability_target"`
	P95LatencyTarget   int     `json:"p95_latency_target"`
}

// SLOQueryParams SLO 查询参数
type SLOQueryParams struct {
	ClusterID string `form:"cluster_id"`
	TimeRange string `form:"time_range"` // "1d", "7d", "30d"
	Host      string `form:"host"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}

// ==================== 服务网格 API 响应类型 ====================

// ServiceMeshTopologyResponse 服务拓扑响应
type ServiceMeshTopologyResponse struct {
	Nodes []ServiceNodeResponse `json:"nodes"`
	Edges []ServiceEdgeResponse `json:"edges"`
}

// ServiceNodeResponse 服务节点响应
type ServiceNodeResponse struct {
	ID            string  `json:"id"`             // "namespace/name"
	Name          string  `json:"name"`
	Namespace     string  `json:"namespace"`
	RPS           float64 `json:"rps"`
	AvgLatencyMs  int     `json:"avg_latency"`
	P50LatencyMs  int     `json:"p50_latency"`
	P95LatencyMs  int     `json:"p95_latency"`
	P99LatencyMs  int     `json:"p99_latency"`
	ErrorRate     float64 `json:"error_rate"`
	Availability  float64 `json:"availability"`
	Status        string  `json:"status"`         // healthy/warning/critical
	MtlsPercent   float64 `json:"mtls_percent"`
	TotalRequests int64   `json:"total_requests"`
}

// ServiceEdgeResponse 服务拓扑边响应
type ServiceEdgeResponse struct {
	Source       string  `json:"source"`      // "namespace/name"
	Target       string  `json:"target"`
	RPS          float64 `json:"rps"`
	AvgLatencyMs int     `json:"avg_latency"`
	ErrorRate    float64 `json:"error_rate"`
}

// ServiceDetailResponse 服务详情响应
type ServiceDetailResponse struct {
	ServiceNodeResponse
	History     []ServiceHistoryPoint `json:"history"`
	Upstreams   []ServiceEdgeResponse `json:"upstreams"`
	Downstreams []ServiceEdgeResponse `json:"downstreams"`
}

// ServiceHistoryPoint 服务历史数据点
type ServiceHistoryPoint struct {
	Timestamp    string  `json:"timestamp"`
	RPS          float64 `json:"rps"`
	P95LatencyMs int     `json:"p95_latency"`
	ErrorRate    float64 `json:"error_rate"`
	Availability float64 `json:"availability"`
	MtlsPercent  float64 `json:"mtls_percent"`
}
