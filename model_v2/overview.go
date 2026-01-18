package model_v2

import "time"

// ============================================================
// ClusterOverview 集群概览（API 响应）
// ============================================================

// ClusterOverview 集群概览
//
// /api/v2/overview 接口的响应数据结构。
// 为 Web Overview 页面提供汇总数据。
type ClusterOverview struct {
	ClusterID string            `json:"cluster_id"`
	Cards     OverviewCards     `json:"cards"`
	Workloads OverviewWorkloads `json:"workloads"` // 替代原来的 Trends
	Alerts    OverviewAlerts    `json:"alerts"`
	Nodes     OverviewNodes     `json:"nodes"`
}

// ============================================================
// 卡片数据
// ============================================================

// OverviewCards 概览卡片数据
type OverviewCards struct {
	ClusterHealth ClusterHealth   `json:"cluster_health"`
	NodeReady     ResourceReady   `json:"node_ready"`
	CPUUsage      ResourcePercent `json:"cpu_usage"`
	MemUsage      ResourcePercent `json:"mem_usage"`
	Events24h     int             `json:"events_24h"`
}

// ClusterHealth 集群健康状态
type ClusterHealth struct {
	Status           string  `json:"status"` // Healthy, Degraded, Unhealthy
	Reason           string  `json:"reason,omitempty"`
	NodeReadyPercent float64 `json:"node_ready_percent"`
	PodReadyPercent  float64 `json:"pod_ready_percent"`
}

// ResourceReady 资源就绪统计
type ResourceReady struct {
	Total   int     `json:"total"`
	Ready   int     `json:"ready"`
	Percent float64 `json:"percent"`
}

// ResourcePercent 资源百分比
type ResourcePercent struct {
	Percent float64 `json:"percent"`
}

// ============================================================
// 工作负载数据（替代原来的趋势数据）
// ============================================================

// OverviewWorkloads 工作负载概览
type OverviewWorkloads struct {
	Summary   WorkloadSummary       `json:"summary"`    // 工作负载统计
	PodStatus PodStatusDistribution `json:"pod_status"` // Pod 状态分布
	PeakStats *PeakStats            `json:"peak_stats,omitempty"` // 资源峰值（保留）
}

// WorkloadSummary 工作负载统计
type WorkloadSummary struct {
	Deployments  WorkloadStatus `json:"deployments"`
	DaemonSets   WorkloadStatus `json:"daemonsets"`
	StatefulSets WorkloadStatus `json:"statefulsets"`
	Jobs         JobStatus      `json:"jobs"`
}

// WorkloadStatus 工作负载状态
type WorkloadStatus struct {
	Total int `json:"total"` // 总数
	Ready int `json:"ready"` // 就绪数
}

// JobStatus Job 状态
type JobStatus struct {
	Total     int `json:"total"`     // 总数
	Running   int `json:"running"`   // 运行中
	Succeeded int `json:"succeeded"` // 已完成
	Failed    int `json:"failed"`    // 失败
}

// PodStatusDistribution Pod 状态分布
type PodStatusDistribution struct {
	Total     int     `json:"total"`
	Running   int     `json:"running"`
	Pending   int     `json:"pending"`
	Failed    int     `json:"failed"`
	Succeeded int     `json:"succeeded"`
	Unknown   int     `json:"unknown"`
	// 百分比（便于前端显示）
	RunningPercent   float64 `json:"running_percent"`
	PendingPercent   float64 `json:"pending_percent"`
	FailedPercent    float64 `json:"failed_percent"`
	SucceededPercent float64 `json:"succeeded_percent"`
}

// PeakStats 峰值统计
type PeakStats struct {
	PeakCPU     float64 `json:"peak_cpu"`
	PeakCPUNode string  `json:"peak_cpu_node"`
	PeakMem     float64 `json:"peak_mem"`
	PeakMemNode string  `json:"peak_mem_node"`
	HasData     bool    `json:"has_data"`
}

// ============================================================
// 告警数据
// ============================================================

// OverviewAlerts 告警数据
type OverviewAlerts struct {
	Trend  []AlertTrendPoint `json:"trend"`
	Totals AlertTotals       `json:"totals"`
	Recent []RecentAlert     `json:"recent"`
}

// AlertTrendPoint 告警趋势点（按资源类型统计）
type AlertTrendPoint struct {
	At    time.Time      `json:"at"`
	Kinds map[string]int `json:"kinds"` // 每种资源类型的告警数量: {"Pod": 5, "Node": 2}
}

// AlertTotals 告警统计
type AlertTotals struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

// RecentAlert 最近告警
type RecentAlert struct {
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"` // critical, warning, info
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Message   string `json:"message"`
	Reason    string `json:"reason"`
}

// ============================================================
// 节点数据
// ============================================================

// OverviewNodes 节点数据
type OverviewNodes struct {
	Usage []NodeUsage `json:"usage"`
}

// NodeUsage 节点使用率
type NodeUsage struct {
	Node     string  `json:"node"`
	CPUUsage float64 `json:"cpu_usage"`
	MemUsage float64 `json:"mem_usage"`
}

// ============================================================
// 辅助函数
// ============================================================

// CalculateHealthStatus 根据就绪百分比计算健康状态
func CalculateHealthStatus(nodeReadyPct, podReadyPct float64) string {
	if nodeReadyPct >= 90 && podReadyPct >= 80 {
		return "Healthy"
	}
	if nodeReadyPct >= 70 && podReadyPct >= 60 {
		return "Degraded"
	}
	return "Unhealthy"
}

// CalculateHealthReason 生成健康状态原因
func CalculateHealthReason(nodeReadyPct, podReadyPct float64) string {
	if nodeReadyPct < 70 {
		return "节点就绪率过低"
	}
	if podReadyPct < 60 {
		return "Pod 就绪率过低"
	}
	if nodeReadyPct < 90 {
		return "部分节点未就绪"
	}
	if podReadyPct < 80 {
		return "部分 Pod 未就绪"
	}
	return ""
}
