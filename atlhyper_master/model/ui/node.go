// atlhyper_master/dto/ui/node.go
// Node UI DTOs
package ui

import "time"

// ====================== Overview ======================

// NodeOverviewDTO - 节点概览
type NodeOverviewDTO struct {
	Cards NodeCards       `json:"cards"`
	Rows  []NodeRowSimple `json:"rows"`
}

type NodeCards struct {
	TotalNodes     int     `json:"totalNodes"`
	ReadyNodes     int     `json:"readyNodes"`
	TotalCPU       int     `json:"totalCPU"`
	TotalMemoryGiB float64 `json:"totalMemoryGiB"`
}

type NodeRowSimple struct {
	Name         string  `json:"name"`
	Ready        bool    `json:"ready"`
	InternalIP   string  `json:"internalIP"`
	OSImage      string  `json:"osImage"`
	Architecture string  `json:"architecture"`
	CPUCores     int     `json:"cpuCores"`
	MemoryGiB    float64 `json:"memoryGiB"`
	Schedulable  bool    `json:"schedulable"`
}

// ====================== Detail ======================

// NodeDetailDTO - 节点详情
type NodeDetailDTO struct {
	Name        string    `json:"name"`
	Roles       []string  `json:"roles,omitempty"`
	Ready       bool      `json:"ready"`
	Schedulable bool      `json:"schedulable"`
	Age         string    `json:"age,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`

	Hostname     string `json:"hostname,omitempty"`
	InternalIP   string `json:"internalIP,omitempty"`
	ExternalIP   string `json:"externalIP,omitempty"`
	OSImage      string `json:"osImage,omitempty"`
	OS           string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Kernel       string `json:"kernel,omitempty"`
	CRI          string `json:"cri,omitempty"`
	Kubelet      string `json:"kubelet,omitempty"`
	KubeProxy    string `json:"kubeProxy,omitempty"`

	CPUCapacityCores    int     `json:"cpuCapacityCores,omitempty"`
	CPUAllocatableCores int     `json:"cpuAllocatableCores,omitempty"`
	MemCapacityGiB      float64 `json:"memCapacityGiB,omitempty"`
	MemAllocatableGiB   float64 `json:"memAllocatableGiB,omitempty"`
	PodsCapacity        int     `json:"podsCapacity,omitempty"`
	PodsAllocatable     int     `json:"podsAllocatable,omitempty"`
	EphemeralStorageGiB float64 `json:"ephemeralStorageGiB,omitempty"`

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

	PodCIDRs   []string          `json:"podCIDRs,omitempty"`
	ProviderID string            `json:"providerID,omitempty"`

	Conditions []NodeCondDTO     `json:"conditions,omitempty"`
	Taints     []NodeTaintDTO    `json:"taints,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`

	Badges  []string `json:"badges,omitempty"`
	Reason  string   `json:"reason,omitempty"`
	Message string   `json:"message,omitempty"`
}

type NodeCondDTO struct {
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Message   string    `json:"message,omitempty"`
	Heartbeat time.Time `json:"heartbeat,omitempty"`
	ChangedAt time.Time `json:"changedAt,omitempty"`
}

type NodeTaintDTO struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}
