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
	Status           string  `json:"status"`
	Reason           string  `json:"reason,omitempty"`
	NodeReadyPercent float64 `json:"nodeReadyPercent"`
	PodReadyPercent  float64 `json:"podReadyPercent"`
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

func CalculateHealthStatus(nodeReadyPct, podReadyPct float64) string {
	if nodeReadyPct >= 90 && podReadyPct >= 80 {
		return "Healthy"
	}
	if nodeReadyPct >= 70 && podReadyPct >= 60 {
		return "Degraded"
	}
	return "Unhealthy"
}

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
