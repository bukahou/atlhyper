package node

import "time"

// NodeDetailDTO —— 节点详情（扁平化）
type NodeDetailDTO struct {
	// 基本
	Name        string    `json:"name"`
	Roles       []string  `json:"roles,omitempty"`
	Ready       bool      `json:"ready"`       // Summary.Ready == "True"
	Schedulable bool      `json:"schedulable"` // !spec.unschedulable
	Age         string    `json:"age,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`

	// 地址与系统
	Hostname     string `json:"hostname,omitempty"`
	InternalIP   string `json:"internalIP,omitempty"`
	ExternalIP   string `json:"externalIP,omitempty"`
	OSImage      string `json:"osImage,omitempty"`
	OS           string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Kernel       string `json:"kernel,omitempty"`
	CRI          string `json:"cri,omitempty"` // containerRuntimeVersion
	Kubelet      string `json:"kubelet,omitempty"`
	KubeProxy    string `json:"kubeProxy,omitempty"`

	// 资源（容量/可分配）
	CPUCapacityCores    int     `json:"cpuCapacityCores,omitempty"`
	CPUAllocatableCores int     `json:"cpuAllocatableCores,omitempty"`
	MemCapacityGiB      float64 `json:"memCapacityGiB,omitempty"`
	MemAllocatableGiB   float64 `json:"memAllocatableGiB,omitempty"`
	PodsCapacity        int     `json:"podsCapacity,omitempty"`
	PodsAllocatable     int     `json:"podsAllocatable,omitempty"`
	EphemeralStorageGiB float64 `json:"ephemeralStorageGiB,omitempty"`

	// 当前指标（扁平）
	CPUUsageCores      float64 `json:"cpuUsageCores,omitempty"`
	CPUUtilPct         float64 `json:"cpuUtilPct,omitempty"`
	MemUsageGiB        float64 `json:"memUsageGiB,omitempty"`
	MemUtilPct         float64 `json:"memUtilPct,omitempty"`
	PodsUsed           int     `json:"podsUsed,omitempty"`
	PodsUtilPct        float64 `json:"podsUtilPct,omitempty"`
	PressureMemory     bool    `json:"pressureMemory,omitempty"`
	PressureDisk       bool    `json:"pressureDisk,omitempty"`
	PressurePID        bool    `json:"pressurePID,omitempty"`
	NetworkUnavailable bool    `json:"networkUnavailable,omitempty"`

	// 调度相关
	PodCIDRs   []string          `json:"podCIDRs,omitempty"`
	ProviderID string            `json:"providerID,omitempty"`

	// 条件/污点/标签（简化）
	Conditions []NodeCondDTO     `json:"conditions,omitempty"`
	Taints     []TaintDTO        `json:"taints,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`

	// 诊断/徽标
	Badges  []string `json:"badges,omitempty"`
	Reason  string   `json:"reason,omitempty"`
	Message string   `json:"message,omitempty"`
}

type NodeCondDTO struct {
	Type      string    `json:"type"`
	Status    string    `json:"status"` // True/False/Unknown
	Reason    string    `json:"reason,omitempty"`
	Message   string    `json:"message,omitempty"`
	Heartbeat time.Time `json:"heartbeat,omitempty"`
	ChangedAt time.Time `json:"changedAt,omitempty"`
}

type TaintDTO struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"` // NoSchedule/PreferNoSchedule/NoExecute
}
