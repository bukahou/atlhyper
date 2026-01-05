// model/k8s/service.go
// Service 资源模型
package k8s

import "time"

// ====================== 顶层：Service ======================

type Service struct {
	Summary  ServiceSummary    `json:"summary"`
	Spec     ServiceSpec       `json:"spec"`
	Ports    []ServicePort     `json:"ports,omitempty"`
	Selector map[string]string `json:"selector,omitempty"`
	Network  ServiceNetwork    `json:"network"`
	Backends *ServiceBackends  `json:"backends,omitempty"`
}

// ====================== summary ======================

type ServiceSummary struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age"`
	PortsCount   int       `json:"portsCount"`
	HasSelector  bool      `json:"hasSelector"`
	Badges       []string  `json:"badges,omitempty"`
	ClusterIP    string    `json:"clusterIP,omitempty"`
	ExternalName string    `json:"externalName,omitempty"`
}

// ====================== spec ======================

type ServiceSpec struct {
	Type                          string   `json:"type"`
	SessionAffinity               string   `json:"sessionAffinity,omitempty"`
	SessionAffinityTimeoutSeconds *int32   `json:"sessionAffinityTimeoutSeconds,omitempty"`
	ExternalTrafficPolicy         string   `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy         string   `json:"internalTrafficPolicy,omitempty"`
	IPFamilies                    []string `json:"ipFamilies,omitempty"`
	IPFamilyPolicy                string   `json:"ipFamilyPolicy,omitempty"`
	ClusterIPs                    []string `json:"clusterIPs,omitempty"`
	ExternalIPs                   []string `json:"externalIPs,omitempty"`
	LoadBalancerClass             string   `json:"loadBalancerClass,omitempty"`
	LoadBalancerSourceRanges      []string `json:"loadBalancerSourceRanges,omitempty"`
	PublishNotReadyAddresses      bool     `json:"publishNotReadyAddresses,omitempty"`
	AllocateLoadBalancerNodePorts *bool    `json:"allocateLoadBalancerNodePorts,omitempty"`
	HealthCheckNodePort           int32    `json:"healthCheckNodePort,omitempty"`
	ExternalName                  string   `json:"externalName,omitempty"`
}

// ====================== ports ======================

type ServicePort struct {
	Name        string `json:"name,omitempty"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"targetPort"`
	NodePort    int32  `json:"nodePort,omitempty"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

// ====================== network ======================

type ServiceNetwork struct {
	ClusterIPs            []string `json:"clusterIPs,omitempty"`
	ExternalIPs           []string `json:"externalIPs,omitempty"`
	LoadBalancerIngress   []string `json:"loadBalancerIngress,omitempty"`
	IPFamilies            []string `json:"ipFamilies,omitempty"`
	IPFamilyPolicy        string   `json:"ipFamilyPolicy,omitempty"`
	ExternalTrafficPolicy string   `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy string   `json:"internalTrafficPolicy,omitempty"`
}

// ====================== backends ======================

type ServiceBackends struct {
	Summary   BackendSummary    `json:"summary"`
	Ports     []EndpointPort    `json:"ports,omitempty"`
	Endpoints []BackendEndpoint `json:"endpoints,omitempty"`
}

type BackendSummary struct {
	Ready    int       `json:"ready"`
	NotReady int       `json:"notReady"`
	Total    int       `json:"total"`
	Slices   int       `json:"slices,omitempty"`
	Updated  time.Time `json:"updated,omitempty"`
}

type EndpointPort struct {
	Name        string `json:"name,omitempty"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type BackendEndpoint struct {
	Address   string  `json:"address"`
	Ready     bool    `json:"ready"`
	NodeName  string  `json:"nodeName,omitempty"`
	Zone      string  `json:"zone,omitempty"`
	TargetRef *K8sRef `json:"targetRef,omitempty"`
}

// K8sRef 通用 K8s 对象引用
type K8sRef struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	UID       string `json:"uid,omitempty"`
}
