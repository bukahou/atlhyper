// model/k8s/node.go
// Node 资源模型
package k8s

import "time"

// ====================== 顶层：Node ======================

type Node struct {
	Summary     NodeSummary       `json:"summary"`
	Spec        NodeSpec          `json:"spec"`
	Capacity    NodeResources     `json:"capacity"`
	Allocatable NodeResources     `json:"allocatable"`
	Addresses   NodeAddresses     `json:"addresses"`
	Info        NodeInfo          `json:"info"`
	Conditions  []NodeCondition   `json:"conditions,omitempty"`
	Taints      []Taint           `json:"taints,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Metrics     *NodeMetrics      `json:"metrics,omitempty"`
}

// ====================== summary ======================

type NodeSummary struct {
	Name         string    `json:"name"`
	Roles        []string  `json:"roles,omitempty"`
	Ready        string    `json:"ready"`
	Schedulable  bool      `json:"schedulable"`
	Age          string    `json:"age"`
	CreationTime time.Time `json:"creationTime"`
	Badges       []string  `json:"badges,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Message      string    `json:"message,omitempty"`
}

// ====================== spec ======================

type NodeSpec struct {
	PodCIDRs      []string `json:"podCIDRs,omitempty"`
	ProviderID    string   `json:"providerID,omitempty"`
	Unschedulable bool     `json:"unschedulable,omitempty"`
}

// ====================== resources ======================

type NodeResources struct {
	CPU              string            `json:"cpu,omitempty"`
	Memory           string            `json:"memory,omitempty"`
	Pods             string            `json:"pods,omitempty"`
	EphemeralStorage string            `json:"ephemeralStorage,omitempty"`
	ScalarResources  map[string]string `json:"scalarResources,omitempty"`
}

// ====================== addresses ======================

type NodeAddresses struct {
	Hostname   string `json:"hostname,omitempty"`
	InternalIP string `json:"internalIP,omitempty"`
	ExternalIP string `json:"externalIP,omitempty"`
	All        []Addr `json:"all,omitempty"`
}

type Addr struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// ====================== info ======================

type NodeInfo struct {
	OSImage                 string `json:"osImage,omitempty"`
	OperatingSystem         string `json:"operatingSystem,omitempty"`
	Architecture            string `json:"architecture,omitempty"`
	KernelVersion           string `json:"kernelVersion,omitempty"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion,omitempty"`
	KubeletVersion          string `json:"kubeletVersion,omitempty"`
	KubeProxyVersion        string `json:"kubeProxyVersion,omitempty"`
}

// ====================== conditions & taints ======================

type NodeCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastHeartbeatTime  time.Time `json:"lastHeartbeatTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

type Taint struct {
	Key       string     `json:"key"`
	Value     string     `json:"value,omitempty"`
	Effect    string     `json:"effect"`
	TimeAdded *time.Time `json:"timeAdded,omitempty"`
}

// ====================== metrics ======================

type NodeMetrics struct {
	CPU      NodeResourceMetric `json:"cpu"`
	Memory   NodeResourceMetric `json:"memory"`
	Pods     PodCountMetric     `json:"pods"`
	Pressure PressureFlags      `json:"pressure,omitempty"`
}

type NodeResourceMetric struct {
	Usage       string  `json:"usage"`
	Allocatable string  `json:"allocatable,omitempty"`
	Capacity    string  `json:"capacity,omitempty"`
	UtilPct     float64 `json:"utilPct,omitempty"`
}

type PodCountMetric struct {
	Used     int     `json:"used"`
	Capacity int     `json:"capacity"`
	UtilPct  float64 `json:"utilPct,omitempty"`
}

type PressureFlags struct {
	MemoryPressure     bool `json:"memoryPressure,omitempty"`
	DiskPressure       bool `json:"diskPressure,omitempty"`
	PIDPressure        bool `json:"pidPressure,omitempty"`
	NetworkUnavailable bool `json:"networkUnavailable,omitempty"`
}
