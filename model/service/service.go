package service

import "time"

// ====================== 顶层：Service（直接发送这个结构体） ======================

type Service struct {
	Summary  ServiceSummary     `json:"summary"`            // 概要（列表常用字段）
	Spec     ServiceSpec        `json:"spec"`               // 规格（调度/网络策略相关）
	Ports    []ServicePort      `json:"ports,omitempty"`    // Service 暴露的端口
	Selector map[string]string  `json:"selector,omitempty"` // 选择器（可能为空：无选择器）
	Network  ServiceNetwork     `json:"network"`            // 网络信息（ClusterIP/LB/ExternalIP 等）
	Backends *ServiceBackends   `json:"backends,omitempty"` // 端点摘要（来自 EndpointSlice/Endpoints 的聚合）
}

// ====================== summary ======================

type ServiceSummary struct {
	Name        string    `json:"name"`                   // Service 名称
	Namespace   string    `json:"namespace"`              // 命名空间
	Type        string    `json:"type"`                   // ClusterIP/NodePort/LoadBalancer/ExternalName
	CreatedAt   time.Time `json:"createdAt"`              // 创建时间
	Age         string    `json:"age"`                    // 运行时长（派生显示）
	PortsCount  int       `json:"portsCount"`             // 端口数量
	HasSelector bool      `json:"hasSelector"`            // 是否存在 selector
	Badges      []string  `json:"badges,omitempty"`       // UI 徽标（Headless/LB/NoSelector/ExternalName 等）
	// 补充观测性字段（可选：用于快查）
	ClusterIP    string   `json:"clusterIP,omitempty"`    // 主 ClusterIP（有多个时取第一个）
	ExternalName string   `json:"externalName,omitempty"` // ExternalName 的别名（仅 Type=ExternalName）
}

// ====================== spec ======================

type ServiceSpec struct {
	Type                          string   `json:"type"`                                     // ClusterIP/NodePort/LoadBalancer/ExternalName
	SessionAffinity               string   `json:"sessionAffinity,omitempty"`                // None/ClientIP
	SessionAffinityTimeoutSeconds *int32   `json:"sessionAffinityTimeoutSeconds,omitempty"` // (k8s 1.22+)
	ExternalTrafficPolicy         string   `json:"externalTrafficPolicy,omitempty"`          // Cluster/Local（LB/NodePort 生效）
	InternalTrafficPolicy         string   `json:"internalTrafficPolicy,omitempty"`          // Cluster/Local（1.22+）
	IPFamilies                    []string `json:"ipFamilies,omitempty"`                     // IPv4/IPv6
	IPFamilyPolicy                string   `json:"ipFamilyPolicy,omitempty"`                 // SingleStack/PreferDualStack/RequireDualStack
	ClusterIPs                    []string `json:"clusterIPs,omitempty"`                     // 可能双栈
	ExternalIPs                   []string `json:"externalIPs,omitempty"`                    // 手工外部 IP
	LoadBalancerClass             string   `json:"loadBalancerClass,omitempty"`              // LB 类
	LoadBalancerSourceRanges      []string `json:"loadBalancerSourceRanges,omitempty"`       // 允许访问的 CIDR
	PublishNotReadyAddresses      bool     `json:"publishNotReadyAddresses,omitempty"`       // 未就绪也暴露
	AllocateLoadBalancerNodePorts *bool    `json:"allocateLoadBalancerNodePorts,omitempty"`  // LB 是否分配 nodePort
	HealthCheckNodePort           int32    `json:"healthCheckNodePort,omitempty"`            // LB 健康检查端口（Local 时）
	ExternalName                  string   `json:"externalName,omitempty"`                   // Type=ExternalName 时的别名
}

// ====================== ports ======================

type ServicePort struct {
	Name        string `json:"name,omitempty"`      // 端口名
	Protocol    string `json:"protocol"`            // TCP/UDP/SCTP
	Port        int32  `json:"port"`                // Service 端口
	TargetPort  string `json:"targetPort"`          // 以字符串承载 intstr（"80"/"http"）
	NodePort    int32  `json:"nodePort,omitempty"`  // NodePort（Type=NodePort/LB 时）
	AppProtocol string `json:"appProtocol,omitempty"` // 应用协议（如 "http"）
}

// ====================== network ======================

type ServiceNetwork struct {
	ClusterIPs            []string `json:"clusterIPs,omitempty"`            // 主/备 ClusterIP（双栈）
	ExternalIPs           []string `json:"externalIPs,omitempty"`           // 外部 IP 列表
	LoadBalancerIngress   []string `json:"loadBalancerIngress,omitempty"`   // LB 入口：IP 或 Hostname
	IPFamilies            []string `json:"ipFamilies,omitempty"`            // IPv4/IPv6
	IPFamilyPolicy        string   `json:"ipFamilyPolicy,omitempty"`
	ExternalTrafficPolicy string   `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy string   `json:"internalTrafficPolicy,omitempty"`
}

// ====================== backends（来自 EndpointSlice/Endpoints 聚合） ======================

type ServiceBackends struct {
	Summary   BackendSummary   `json:"summary"`             // 端点总体情况
	Ports     []EndpointPort   `json:"ports,omitempty"`     // 端点端口定义（来自 EndpointSlice.Ports 聚合）
	Endpoints []BackendEndpoint`json:"endpoints,omitempty"` // 扁平化的后端地址列表（可按需精简）
}

type BackendSummary struct {
	Ready    int       `json:"ready"`              // Ready 端点数
	NotReady int       `json:"notReady"`           // NotReady 端点数
	Total    int       `json:"total"`              // 总端点数（=Ready+NotReady）
	Slices   int       `json:"slices,omitempty"`   // 聚合的 EndpointSlice 数
	Updated  time.Time `json:"updated,omitempty"`  // 端点观测更新时间
}

type EndpointPort struct {
	Name        string `json:"name,omitempty"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`              // TCP/UDP/SCTP
	AppProtocol string `json:"appProtocol,omitempty"` // 例如 "http"
}

type BackendEndpoint struct {
	Address   string  `json:"address"`                     // IP（或主机名：极少）
	Ready     bool    `json:"ready"`                       // 是否就绪
	NodeName  string  `json:"nodeName,omitempty"`          // 所在节点（如有）
	Zone      string  `json:"zone,omitempty"`              // 拓扑区域（如 "topology.kubernetes.io/zone"）
	TargetRef *K8sRef `json:"targetRef,omitempty"`         // 指向 Pod/其它对象的引用（如有）
}

// 通用 K8s 对象引用（精简版）
type K8sRef struct {
	Kind      string `json:"kind,omitempty"`      // Pod/Node/...
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	UID       string `json:"uid,omitempty"`
}
