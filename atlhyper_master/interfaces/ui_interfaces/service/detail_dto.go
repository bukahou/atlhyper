package service

import "time"

// ServiceDetailDTO —— Service 详情（扁平化、UI 友好）
type ServiceDetailDTO struct {
	// 基本
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"createdAt"`
	Age        string    `json:"age,omitempty"`

	// 选择器 & 端口
	Selector map[string]string `json:"selector,omitempty"`
	Ports    []ServicePortDTO  `json:"ports,omitempty"`

	// 网络信息
	ClusterIPs          []string `json:"clusterIPs,omitempty"`          // 双栈/多 IP
	ExternalIPs         []string `json:"externalIPs,omitempty"`
	LoadBalancerIngress []string `json:"loadBalancerIngress,omitempty"` // IP/Hostname

	// 重要 spec
	SessionAffinity               string   `json:"sessionAffinity,omitempty"`
	SessionAffinityTimeoutSeconds *int32   `json:"sessionAffinityTimeoutSeconds,omitempty"`
	ExternalTrafficPolicy         string   `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy         string   `json:"internalTrafficPolicy,omitempty"`
	IPFamilies                    []string `json:"ipFamilies,omitempty"`
	IPFamilyPolicy                string   `json:"ipFamilyPolicy,omitempty"`
	LoadBalancerClass             string   `json:"loadBalancerClass,omitempty"`
	LoadBalancerSourceRanges      []string `json:"loadBalancerSourceRanges,omitempty"`
	AllocateLoadBalancerNodePorts *bool    `json:"allocateLoadBalancerNodePorts,omitempty"`
	HealthCheckNodePort           int32    `json:"healthCheckNodePort,omitempty"`
	ExternalName                  string   `json:"externalName,omitempty"`

	// 端点聚合（如已采集）
	Backends *BackendsDTO `json:"backends,omitempty"`

	// 快速徽标/标识（可选）
	Badges []string `json:"badges,omitempty"`
}

type ServicePortDTO struct {
	Name        string `json:"name,omitempty"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"targetPort"`
	NodePort    int32  `json:"nodePort,omitempty"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type BackendsDTO struct {
	Ready    int                `json:"ready"`
	NotReady int                `json:"notReady"`
	Total    int                `json:"total"`
	Slices   int                `json:"slices,omitempty"`
	Updated  time.Time          `json:"updated,omitempty"`
	Ports    []EndpointPortDTO  `json:"ports,omitempty"`
	Endpoints []BackendEndpointDTO `json:"endpoints,omitempty"`
}

type EndpointPortDTO struct {
	Name        string `json:"name,omitempty"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type BackendEndpointDTO struct {
	Address   string       `json:"address"`
	Ready     bool         `json:"ready"`
	NodeName  string       `json:"nodeName,omitempty"`
	Zone      string       `json:"zone,omitempty"`
	TargetRef *K8sRefDTO   `json:"targetRef,omitempty"`
}

type K8sRefDTO struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	UID       string `json:"uid,omitempty"`
}
