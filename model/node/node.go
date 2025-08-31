package node

import "time"

// ====================== 顶层：Node（直接发送这个结构体） ======================

type Node struct {
	Summary     NodeSummary      `json:"summary"`                 // 概要（列表常用）
	Spec        NodeSpec         `json:"spec"`                    // 调度/污点/标签等
	Capacity    NodeResources    `json:"capacity"`                // 节点可提供的总量（来自 status.capacity）
	Allocatable NodeResources    `json:"allocatable"`             // 可分配（来自 status.allocatable）
	Addresses   NodeAddresses    `json:"addresses"`               // IP/主机名等
	Info        NodeInfo         `json:"info"`                    // 版本/内核/CRI 等
	Conditions  []NodeCondition  `json:"conditions,omitempty"`    // 就绪/压力等
	Taints      []Taint          `json:"taints,omitempty"`        // 污点（调度相关）
	Labels      map[string]string`json:"labels,omitempty"`        // 标签（可在后端裁剪或前端懒加载）
	Metrics     *NodeMetrics     `json:"metrics,omitempty"`       // 运行时指标（可为空）
}

// ====================== summary ======================

type NodeSummary struct {
	Name          string    `json:"name"`                         // 节点名
	Roles         []string  `json:"roles,omitempty"`              // 解析自 labels：node-role.kubernetes.io/*
	Ready         string    `json:"ready"`                        // "True"/"False"/"Unknown"
	Schedulable   bool      `json:"schedulable"`                  // 是否可调度（!spec.unschedulable）
	Age           string    `json:"age"`                          // 运行时长（派生显示）
	CreationTime  time.Time `json:"creationTime"`                 // 创建时间
	Badges        []string  `json:"badges,omitempty"`             // UI 徽标（NotReady/MemoryPressure 等）
	Reason        string    `json:"reason,omitempty"`             // 汇总原因（非 Ready 时）
	Message       string    `json:"message,omitempty"`            // 汇总消息
}

// ====================== spec ======================

type NodeSpec struct {
	PodCIDRs     []string `json:"podCIDRs,omitempty"`             // 可能有多 CIDR
	ProviderID   string   `json:"providerID,omitempty"`
	Unschedulable bool    `json:"unschedulable,omitempty"`        // 与 Summary.Schedulable 相反
}

// ====================== resources ======================
// 使用字符串承载 K8s 资源计量（与 Pod 模型一致，便于直接显示/比较）
// 例如：{"cpu":"8","memory":"32Gi","pods":"110","ephemeral-storage":"100Gi"}

type NodeResources struct {
	CPU              string            `json:"cpu,omitempty"`               // 逻辑核数（如 "8"）
	Memory           string            `json:"memory,omitempty"`            // 如 "32Gi"
	Pods             string            `json:"pods,omitempty"`              // 最大可调度 Pod 数
	EphemeralStorage string            `json:"ephemeralStorage,omitempty"`  // 如 "100Gi"
	ScalarResources  map[string]string `json:"scalarResources,omitempty"`   // 其它标量：hugepages-*, gpu 等
}

// ====================== addresses ======================

type NodeAddresses struct {
	Hostname   string   `json:"hostname,omitempty"`
	InternalIP string   `json:"internalIP,omitempty"`
	ExternalIP string   `json:"externalIP,omitempty"`
	All        []Addr   `json:"all,omitempty"` // 完整地址列表
}

type Addr struct {
	Type    string `json:"type"`    // Hostname/InternalIP/ExternalIP/...
	Address string `json:"address"`
}

// ====================== info ======================

type NodeInfo struct {
	OSImage                 string `json:"osImage,omitempty"`
	OperatingSystem         string `json:"operatingSystem,omitempty"`  // linux/windows
	Architecture            string `json:"architecture,omitempty"`     // amd64/arm64 ...
	KernelVersion           string `json:"kernelVersion,omitempty"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion,omitempty"` // e.g. containerd://1.7.x
	KubeletVersion          string `json:"kubeletVersion,omitempty"`
	KubeProxyVersion        string `json:"kubeProxyVersion,omitempty"`
}

// ====================== conditions & taints ======================

type NodeCondition struct {
	Type               string    `json:"type"`                // Ready/MemoryPressure/DiskPressure/PIDPressure/NetworkUnavailable
	Status             string    `json:"status"`              // True/False/Unknown
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastHeartbeatTime  time.Time `json:"lastHeartbeatTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

type Taint struct {
	Key       string     `json:"key"`
	Value     string     `json:"value,omitempty"`
	Effect    string     `json:"effect"`                 // NoSchedule/PreferNoSchedule/NoExecute
	TimeAdded *time.Time `json:"timeAdded,omitempty"`
}

// ====================== metrics ======================
// 约定：UtilPct = usage / (allocatable|capacity) * 100
// - 优先用 Allocatable；若为 0 再退回 Capacity；均为 0 时不计算（UtilPct 省略或置 0）

type NodeMetrics struct {
	CPU       ResourceMetric `json:"cpu"`                     // cores & 利用率
	Memory    ResourceMetric `json:"memory"`                  // bytes & 利用率
	Pods      PodCountMetric `json:"pods"`                    // 运行中的 Pod 数
	Pressure  PressureFlags  `json:"pressure,omitempty"`      // 快速标识压力类条件
}

type ResourceMetric struct {
	Usage        string  `json:"usage"`                        // 如 "3500m"/"28Gi"
	Allocatable  string  `json:"allocatable,omitempty"`        // 来自节点 allocatable
	Capacity     string  `json:"capacity,omitempty"`           // 来自节点 capacity
	UtilPct      float64 `json:"utilPct,omitempty"`            // 0-100，见上方约定
}

type PodCountMetric struct {
	Used     int     `json:"used"`                            // 当前已调度/运行中 Pod 数（可按 Node 统计）
	Capacity int     `json:"capacity"`                        // 节点可承载 Pod 总数（来自 capacity.pods）
	UtilPct  float64 `json:"utilPct,omitempty"`               // 0-100
}

type PressureFlags struct {
	MemoryPressure   bool `json:"memoryPressure,omitempty"`
	DiskPressure     bool `json:"diskPressure,omitempty"`
	PIDPressure      bool `json:"pidPressure,omitempty"`
	NetworkUnavailable bool `json:"networkUnavailable,omitempty"`
}
