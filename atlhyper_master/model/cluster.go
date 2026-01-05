// atlhyper_master/model/cluster.go
// 集群概览 DTO（Master → UI）
package model

type ClusterResourceSummary struct {
	// CPU
	TotalCPUMilli int64   `json:"total_cpu_milli"`
	UsedCPUMilli  int64   `json:"used_cpu_milli"`
	CPUPercent    float64 `json:"cpu_percent"`
	TotalCPUCores float64 `json:"total_cpu_cores"`
	UsedCPUCores  float64 `json:"used_cpu_cores"`

	// Memory
	TotalMemoryBytes int64   `json:"total_memory_bytes"`
	UsedMemoryBytes  int64   `json:"used_memory_bytes"`
	MemoryPercent    float64 `json:"memory_percent"`
}

// NodeResourceUsage 逐节点资源使用
type NodeResourceUsage struct {
	NodeName string `json:"node_name"`

	// CPU（按 allocatable 作为总量；used 来自 metrics-server）
	TotalCPUMilli int64   `json:"total_cpu_milli"`
	UsedCPUMilli  int64   `json:"used_cpu_milli"`
	CPUPercent    float64 `json:"cpu_percent"`
	TotalCores    float64 `json:"total_cores"`
	UsedCores     float64 `json:"used_cores"`

	// Memory（bytes）
	TotalMemoryBytes int64   `json:"total_memory_bytes"`
	UsedMemoryBytes  int64   `json:"used_memory_bytes"`
	MemoryPercent    float64 `json:"memory_percent"`

	// 可选状态信息
	Ready bool   `json:"ready"`
	Role  string `json:"role,omitempty"`
}

type ClusterOverview struct {
	TotalNodes   int                     `json:"total_nodes"`
	ReadyNodes   int                     `json:"ready_nodes"`
	TotalPods    int                     `json:"total_pods"`
	AbnormalPods int                     `json:"abnormal_pods"`
	K8sVersion   string                  `json:"k8s_version"`
	HasMetrics   bool                    `json:"has_metrics_server"`
	Resources    *ClusterResourceSummary `json:"resources,omitempty"`
	Nodes        []NodeResourceUsage     `json:"nodes,omitempty"`
}
