package clusteroverview

type ClusterResourceSummary struct {
	// CPU
	TotalCPUMilli    int64   `json:"total_cpu_milli"`
	UsedCPUMilli     int64   `json:"used_cpu_milli"`
	CPUPercent       float64 `json:"cpu_percent"`
	TotalCPUCores    float64 `json:"total_cpu_cores"`
	UsedCPUCores     float64 `json:"used_cpu_cores"`

	// Memory
	TotalMemoryBytes int64   `json:"total_memory_bytes"`
	UsedMemoryBytes  int64   `json:"used_memory_bytes"`
	MemoryPercent    float64 `json:"memory_percent"`
}

// 新增：逐节点资源使用
type NodeResourceUsage struct {
	NodeName string `json:"node_name"`

	// CPU（按 allocatable 作为总量；used 来自 metrics-server）
	TotalCPUMilli int64   `json:"total_cpu_milli"`
	UsedCPUMilli  int64   `json:"used_cpu_milli"`
	CPUPercent    float64 `json:"cpu_percent"`   // 0~100
	TotalCores    float64 `json:"total_cores"`   // = TotalCPUMilli / 1000
	UsedCores     float64 `json:"used_cores"`    // = UsedCPUMilli / 1000

	// Memory（bytes）
	TotalMemoryBytes int64   `json:"total_memory_bytes"`
	UsedMemoryBytes  int64   `json:"used_memory_bytes"`
	MemoryPercent    float64 `json:"memory_percent"` // 0~100

	// 可选状态信息（便于前端展示/排序）
	Ready bool   `json:"ready"`
	Role  string `json:"role,omitempty"` // 若需要可从 labels 解析（如 node-role.kubernetes.io/xxx）
}

type ClusterOverview struct {
	TotalNodes   int                     `json:"total_nodes"`
	ReadyNodes   int                     `json:"ready_nodes"`
	TotalPods    int                     `json:"total_pods"`
	AbnormalPods int                     `json:"abnormal_pods"`
	K8sVersion   string                  `json:"k8s_version"`
	HasMetrics   bool                    `json:"has_metrics_server"`
	Resources    *ClusterResourceSummary `json:"resources,omitempty"`

	// 新增：逐节点资源使用列表
	Nodes []NodeResourceUsage `json:"nodes,omitempty"`
}
