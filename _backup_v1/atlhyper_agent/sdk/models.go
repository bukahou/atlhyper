// sdk/models.go
// SDK 数据模型定义 - 平台无关的数据结构
package sdk

import "time"

// ==================== 通用元数据 ====================

// ObjectMeta 通用元数据
type ObjectMeta struct {
	Name              string
	Namespace         string
	UID               string
	Labels            map[string]string
	Annotations       map[string]string
	CreationTimestamp time.Time
}

// ==================== Pod 模型 ====================

// PodInfo Pod 信息
type PodInfo struct {
	Meta       ObjectMeta
	Phase      string // Running, Pending, Succeeded, Failed, Unknown
	NodeName   string
	PodIP      string
	HostIP     string
	Containers []ContainerInfo
	Conditions []PodCondition
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	Name         string
	Image        string
	Ready        bool
	RestartCount int32
	State        string // Running, Waiting, Terminated
	StateReason  string
	StateMessage string
}

// PodCondition Pod 条件
type PodCondition struct {
	Type    string // Ready, Initialized, ContainersReady, PodScheduled
	Status  bool
	Reason  string
	Message string
}

// PodMetrics Pod 指标
type PodMetrics struct {
	Namespace   string
	Name        string
	CPUUsage    int64 // millicores
	MemoryUsage int64 // bytes
}

// ==================== Node 模型 ====================

// NodeInfo Node 信息
type NodeInfo struct {
	Meta          ObjectMeta
	Unschedulable bool
	Conditions    []NodeCondition
	Addresses     []NodeAddress
	Capacity      ResourceList
	Allocatable   ResourceList
	NodeInfo      NodeSystemInfo
}

// NodeCondition Node 条件
type NodeCondition struct {
	Type    string // Ready, MemoryPressure, DiskPressure, PIDPressure, NetworkUnavailable
	Status  bool
	Reason  string
	Message string
}

// NodeAddress Node 地址
type NodeAddress struct {
	Type    string // InternalIP, ExternalIP, Hostname
	Address string
}

// NodeSystemInfo Node 系统信息
type NodeSystemInfo struct {
	KernelVersion           string
	OSImage                 string
	ContainerRuntimeVersion string
	KubeletVersion          string
	Architecture            string
	OperatingSystem         string
}

// NodeMetrics Node 指标
type NodeMetrics struct {
	Name        string
	CPUUsage    int64 // millicores
	MemoryUsage int64 // bytes
}

// ==================== Deployment 模型 ====================

// DeploymentInfo Deployment 信息
type DeploymentInfo struct {
	Meta              ObjectMeta
	Replicas          int32
	ReadyReplicas     int32
	AvailableReplicas int32
	UpdatedReplicas   int32
	Containers        []ContainerSpec
}

// ContainerSpec 容器规格
type ContainerSpec struct {
	Name  string
	Image string
}

// ==================== Service 模型 ====================

// ServiceInfo Service 信息
type ServiceInfo struct {
	Meta        ObjectMeta
	Type        string // ClusterIP, NodePort, LoadBalancer, ExternalName
	ClusterIP   string
	ExternalIPs []string
	Ports       []ServicePort
	Selector    map[string]string
}

// ServicePort Service 端口
type ServicePort struct {
	Name       string
	Port       int32
	TargetPort int32
	NodePort   int32
	Protocol   string
}

// ==================== Namespace 模型 ====================

// NamespaceInfo Namespace 信息
type NamespaceInfo struct {
	Meta   ObjectMeta
	Phase  string // Active, Terminating
	Labels map[string]string
}

// ==================== ConfigMap 模型 ====================

// ConfigMapInfo ConfigMap 信息
type ConfigMapInfo struct {
	Meta ObjectMeta
	Data map[string]string
}

// ==================== Ingress 模型 ====================

// IngressInfo Ingress 信息
type IngressInfo struct {
	Meta       ObjectMeta
	ClassName  string
	Rules      []IngressRule
	TLS        []IngressTLS
	DefaultBackend *IngressBackend
}

// IngressRule Ingress 规则
type IngressRule struct {
	Host  string
	Paths []IngressPath
}

// IngressPath Ingress 路径
type IngressPath struct {
	Path        string
	PathType    string
	ServiceName string
	ServicePort int32
}

// IngressTLS Ingress TLS 配置
type IngressTLS struct {
	Hosts      []string
	SecretName string
}

// IngressBackend Ingress 后端
type IngressBackend struct {
	ServiceName string
	ServicePort int32
}

// ==================== 通用类型 ====================

// ResourceList 资源列表
type ResourceList struct {
	CPU    string // 如 "4" 或 "4000m"
	Memory string // 如 "8Gi"
	Pods   string // 如 "110"
}
