// atlhyper_master/dto/ui/service.go
// Service UI DTOs
package ui

import "time"

// ====================== Overview ======================

// ServiceOverviewDTO - Service 概览
type ServiceOverviewDTO struct {
	Cards ServiceCards       `json:"cards"`
	Rows  []ServiceRowSimple `json:"rows"`
}

type ServiceCards struct {
	TotalServices    int `json:"totalServices"`
	ExternalServices int `json:"externalServices"`
	InternalServices int `json:"internalServices"`
	HeadlessServices int `json:"headlessServices"`
}

type ServiceRowSimple struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	Type       string    `json:"type"`
	ClusterIP  string    `json:"clusterIP"`
	Ports      string    `json:"ports"`
	Protocol   string    `json:"protocol"`
	Selector   string    `json:"selector"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ====================== Detail ======================

// ServiceDetailDTO - Service 详情
type ServiceDetailDTO struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"createdAt"`
	Age        string    `json:"age,omitempty"`

	Selector map[string]string  `json:"selector,omitempty"`
	Ports    []ServicePortDTO   `json:"ports,omitempty"`

	ClusterIPs          []string `json:"clusterIPs,omitempty"`
	ExternalIPs         []string `json:"externalIPs,omitempty"`
	LoadBalancerIngress []string `json:"loadBalancerIngress,omitempty"`

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

	Backends *ServiceBackendsDTO `json:"backends,omitempty"`
	Badges   []string            `json:"badges,omitempty"`
}

type ServicePortDTO struct {
	Name        string `json:"name,omitempty"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"targetPort"`
	NodePort    int32  `json:"nodePort,omitempty"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type ServiceBackendsDTO struct {
	Ready     int                      `json:"ready"`
	NotReady  int                      `json:"notReady"`
	Total     int                      `json:"total"`
	Slices    int                      `json:"slices,omitempty"`
	Updated   time.Time                `json:"updated,omitempty"`
	Ports     []ServiceEndpointPortDTO `json:"ports,omitempty"`
	Endpoints []ServiceBackendEndpoint `json:"endpoints,omitempty"`
}

type ServiceEndpointPortDTO struct {
	Name        string `json:"name,omitempty"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type ServiceBackendEndpoint struct {
	Address   string          `json:"address"`
	Ready     bool            `json:"ready"`
	NodeName  string          `json:"nodeName,omitempty"`
	Zone      string          `json:"zone,omitempty"`
	TargetRef *ServiceK8sRef  `json:"targetRef,omitempty"`
}

type ServiceK8sRef struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	UID       string `json:"uid,omitempty"`
}
