package cluster

import "time"

// ClusterOverview 集群概览
type ClusterOverview struct {
	ClusterID string            `json:"clusterId"`
	Cards     OverviewCards     `json:"cards"`
	Workloads OverviewWorkloads `json:"workloads"`
	Alerts    OverviewAlerts    `json:"alerts"`
	Nodes     OverviewNodes     `json:"nodes"`
}

type OverviewCards struct {
	ClusterHealth ClusterHealth   `json:"clusterHealth"`
	NodeReady     ResourceReady   `json:"nodeReady"`
	CPUUsage      ResourcePercent `json:"cpuUsage"`
	MemUsage      ResourcePercent `json:"memUsage"`
	Events24h     int             `json:"events24h"`
}

type ClusterHealth struct {
	Status string `json:"status"`
	// Reason 原因代码（HealthReason*），文案由前端按语言出；健康时为空
	Reason           string  `json:"reason,omitempty"`
	NodeReadyPercent float64 `json:"nodeReadyPercent"`
	PodReadyPercent  float64 `json:"podReadyPercent"`
	// NodeReadyStatus / PodReadyStatus 两项就绪率各自的结论（与 Status 同一套取值）
	NodeReadyStatus string `json:"nodeReadyStatus"`
	PodReadyStatus  string `json:"podReadyStatus"`
}

type ResourceReady struct {
	Total   int     `json:"total"`
	Ready   int     `json:"ready"`
	Percent float64 `json:"percent"`
}

type ResourcePercent struct {
	Percent float64 `json:"percent"`
}

type OverviewWorkloads struct {
	Summary   WorkloadSummary       `json:"summary"`
	PodStatus PodStatusDistribution `json:"podStatus"`
	PeakStats *PeakStats            `json:"peakStats,omitempty"`
}

type WorkloadSummary struct {
	Deployments  WorkloadStatus `json:"deployments"`
	DaemonSets   WorkloadStatus `json:"daemonSets"`
	StatefulSets WorkloadStatus `json:"statefulSets"`
	Jobs         JobStatus      `json:"jobs"`
}

type WorkloadStatus struct {
	Total int `json:"total"`
	Ready int `json:"ready"`
}

type JobStatus struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type PodStatusDistribution struct {
	Total            int     `json:"total"`
	Running          int     `json:"running"`
	Pending          int     `json:"pending"`
	Failed           int     `json:"failed"`
	Succeeded        int     `json:"succeeded"`
	Unknown          int     `json:"unknown"`
	RunningPercent   float64 `json:"runningPercent"`
	PendingPercent   float64 `json:"pendingPercent"`
	FailedPercent    float64 `json:"failedPercent"`
	SucceededPercent float64 `json:"succeededPercent"`
}

type PeakStats struct {
	PeakCPU     float64 `json:"peakCpu"`
	PeakCPUNode string  `json:"peakCpuNode"`
	PeakMem     float64 `json:"peakMem"`
	PeakMemNode string  `json:"peakMemNode"`
	HasData     bool    `json:"hasData"`
}

type OverviewAlerts struct {
	Trend  []AlertTrendPoint `json:"trend"`
	Totals AlertTotals       `json:"totals"`
	Recent []RecentAlert     `json:"recent"`
}

type AlertTrendPoint struct {
	At    time.Time      `json:"at"`
	Kinds map[string]int `json:"kinds"`
}

type AlertTotals struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

type RecentAlert struct {
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Message   string `json:"message"`
	Reason    string `json:"reason"`
}

type OverviewNodes struct {
	Usage []NodeUsage `json:"usage"`
}

type NodeUsage struct {
	Node     string  `json:"node"`
	CPUUsage float64 `json:"cpuUsage"`
	MemUsage float64 `json:"memUsage"`
}

// 集群健康结论
const (
	HealthHealthy   = "Healthy"
	HealthDegraded  = "Degraded"
	HealthUnhealthy = "Unhealthy"
)

// 健康原因代码（按严重程度先后取第一个命中的）
const (
	HealthReasonNodeReadyLow      = "nodeReadyLow"      // 节点就绪率低于 degraded 线
	HealthReasonPodReadyLow       = "podReadyLow"       // Pod 就绪率低于 degraded 线
	HealthReasonSomeNodesNotReady = "someNodesNotReady" // 部分节点未就绪
	HealthReasonSomePodsNotReady  = "somePodsNotReady"  // 部分 Pod 未就绪
)

// 就绪率阈值：全项目只有这一份（整体结论、原因、两条进度条都用它）
const (
	nodeReadyHealthyPct  = 90.0
	nodeReadyDegradedPct = 70.0
	podReadyHealthyPct   = 80.0
	podReadyDegradedPct  = 60.0
)

func readinessStatus(pct, healthy, degraded float64) string {
	switch {
	case pct >= healthy:
		return HealthHealthy
	case pct >= degraded:
		return HealthDegraded
	default:
		return HealthUnhealthy
	}
}

// NodeReadyStatus 节点就绪率的结论
func NodeReadyStatus(pct float64) string {
	return readinessStatus(pct, nodeReadyHealthyPct, nodeReadyDegradedPct)
}

// PodReadyStatus Pod 就绪率的结论
func PodReadyStatus(pct float64) string {
	return readinessStatus(pct, podReadyHealthyPct, podReadyDegradedPct)
}

// CalculateHealthStatus 集群整体结论 = 两项里更差的一个
func CalculateHealthStatus(nodeReadyPct, podReadyPct float64) string {
	node, pod := NodeReadyStatus(nodeReadyPct), PodReadyStatus(podReadyPct)
	for _, worst := range []string{HealthUnhealthy, HealthDegraded} {
		if node == worst || pod == worst {
			return worst
		}
	}
	return HealthHealthy
}

// CalculateHealthReason 原因代码；健康时为空
func CalculateHealthReason(nodeReadyPct, podReadyPct float64) string {
	switch {
	case nodeReadyPct < nodeReadyDegradedPct:
		return HealthReasonNodeReadyLow
	case podReadyPct < podReadyDegradedPct:
		return HealthReasonPodReadyLow
	case nodeReadyPct < nodeReadyHealthyPct:
		return HealthReasonSomeNodesNotReady
	case podReadyPct < podReadyHealthyPct:
		return HealthReasonSomePodsNotReady
	default:
		return ""
	}
}
