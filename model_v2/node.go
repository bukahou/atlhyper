package model_v2

import "time"

// ============================================================
// Node 模型（与 model/k8s/node.go 结构一致，确保 JSON 解码正确）
// ============================================================

// Node K8s Node 资源模型
type Node struct {
	Summary     NodeSummary       `json:"summary"`
	Spec        NodeSpec          `json:"spec"`
	Capacity    NodeResources     `json:"capacity"`
	Allocatable NodeResources     `json:"allocatable"`
	Addresses   NodeAddresses     `json:"addresses"`
	Info        NodeInfo          `json:"info"`
	Conditions  []NodeCondition   `json:"conditions,omitempty"`
	Taints      []NodeTaint       `json:"taints,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Metrics     *NodeMetrics      `json:"metrics,omitempty"`
}

// NodeSummary 节点摘要
type NodeSummary struct {
	Name         string    `json:"name"`
	Roles        []string  `json:"roles,omitempty"`
	Ready        string    `json:"ready"` // "True", "False", "Unknown"
	Schedulable  bool      `json:"schedulable"`
	Age          string    `json:"age"`
	CreationTime time.Time `json:"creationTime"`
	Badges       []string  `json:"badges,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Message      string    `json:"message,omitempty"`
}

// NodeSpec 节点规格
type NodeSpec struct {
	PodCIDRs      []string `json:"podCIDRs,omitempty"`
	ProviderID    string   `json:"providerID,omitempty"`
	Unschedulable bool     `json:"unschedulable,omitempty"`
}

// NodeResources 节点资源
type NodeResources struct {
	CPU              string            `json:"cpu,omitempty"`
	Memory           string            `json:"memory,omitempty"`
	Pods             string            `json:"pods,omitempty"`
	EphemeralStorage string            `json:"ephemeralStorage,omitempty"`
	ScalarResources  map[string]string `json:"scalarResources,omitempty"`
}

// NodeAddresses 节点地址
type NodeAddresses struct {
	Hostname   string     `json:"hostname,omitempty"`
	InternalIP string     `json:"internalIP,omitempty"`
	ExternalIP string     `json:"externalIP,omitempty"`
	All        []NodeAddr `json:"all,omitempty"`
}

// NodeAddr 地址
type NodeAddr struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// NodeInfo 节点系统信息
type NodeInfo struct {
	OSImage                 string `json:"osImage,omitempty"`
	OperatingSystem         string `json:"operatingSystem,omitempty"`
	Architecture            string `json:"architecture,omitempty"`
	KernelVersion           string `json:"kernelVersion,omitempty"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion,omitempty"`
	KubeletVersion          string `json:"kubeletVersion,omitempty"`
	KubeProxyVersion        string `json:"kubeProxyVersion,omitempty"`
}

// NodeCondition 节点状态条件
type NodeCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"` // "True", "False", "Unknown"
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastHeartbeatTime  time.Time `json:"lastHeartbeatTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// NodeTaint 节点污点
type NodeTaint struct {
	Key       string     `json:"key"`
	Value     string     `json:"value,omitempty"`
	Effect    string     `json:"effect"` // NoSchedule, PreferNoSchedule, NoExecute
	TimeAdded *time.Time `json:"timeAdded,omitempty"`
}

// NodeMetrics 节点指标
type NodeMetrics struct {
	CPU      NodeResourceMetric `json:"cpu"`
	Memory   NodeResourceMetric `json:"memory"`
	Pods     PodCountMetric     `json:"pods"`
	Pressure PressureFlags      `json:"pressure,omitempty"`
}

// NodeResourceMetric 资源指标
type NodeResourceMetric struct {
	Usage       string  `json:"usage"`
	Allocatable string  `json:"allocatable,omitempty"`
	Capacity    string  `json:"capacity,omitempty"`
	UtilPct     float64 `json:"utilPct,omitempty"`
}

// PodCountMetric Pod 数量指标
type PodCountMetric struct {
	Used     int     `json:"used"`
	Capacity int     `json:"capacity"`
	UtilPct  float64 `json:"utilPct,omitempty"`
}

// PressureFlags 压力标志
type PressureFlags struct {
	MemoryPressure     bool `json:"memoryPressure,omitempty"`
	DiskPressure       bool `json:"diskPressure,omitempty"`
	PIDPressure        bool `json:"pidPressure,omitempty"`
	NetworkUnavailable bool `json:"networkUnavailable,omitempty"`
}

// ============================================================
// 辅助方法
// ============================================================

// GetName 获取节点名称
func (n *Node) GetName() string {
	return n.Summary.Name
}

// IsReady 判断节点是否就绪
func (n *Node) IsReady() bool {
	return n.Summary.Ready == "True"
}

// IsSchedulable 判断节点是否可调度
func (n *Node) IsSchedulable() bool {
	return n.Summary.Schedulable && !n.Spec.Unschedulable
}

// IsMaster 判断是否是 Master 节点
func (n *Node) IsMaster() bool {
	for _, role := range n.Summary.Roles {
		if role == "master" || role == "control-plane" {
			return true
		}
	}
	return false
}
