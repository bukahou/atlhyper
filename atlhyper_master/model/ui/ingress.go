// atlhyper_master/dto/ui/ingress.go
// Ingress UI DTOs
package ui

import "time"

// ====================== Overview ======================

// IngressOverviewDTO - 概览页
type IngressOverviewDTO struct {
	Cards IngressOverviewCards `json:"cards"`
	Rows  []IngressRowSimple   `json:"rows"`
}

type IngressOverviewCards struct {
	TotalIngresses int `json:"totalIngresses"`
	UsedHosts      int `json:"usedHosts"`
	TLSCerts       int `json:"tlsCerts"`
	TotalPaths     int `json:"totalPaths"`
}

type IngressRowSimple struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Host        string    `json:"host"`
	Path        string    `json:"path"`
	ServiceName string    `json:"serviceName"`
	ServicePort string    `json:"servicePort"`
	TLS         string    `json:"tls"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ====================== Detail ======================

// IngressDetailDTO - 详情页
type IngressDetailDTO struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Class        string    `json:"class,omitempty"`
	Controller   string    `json:"controller,omitempty"`
	Hosts        []string  `json:"hosts,omitempty"`
	TLSEnabled   bool      `json:"tlsEnabled"`
	LoadBalancer []string  `json:"loadBalancer,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age,omitempty"`

	Spec        IngressSpecDTO    `json:"spec"`
	Status      IngressStatusDTO  `json:"status"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type IngressSpecDTO struct {
	IngressClassName         string             `json:"ingressClassName,omitempty"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty"`
	DefaultBackend           *IngressBackendRef `json:"defaultBackend,omitempty"`
	Rules                    []IngressRuleDTO   `json:"rules,omitempty"`
	TLS                      []IngressTLSDTO    `json:"tls,omitempty"`
}

type IngressRuleDTO struct {
	Host  string             `json:"host,omitempty"`
	Paths []IngressHTTPPath  `json:"paths"`
}

type IngressHTTPPath struct {
	Path     string            `json:"path,omitempty"`
	PathType string            `json:"pathType,omitempty"`
	Backend  IngressBackendRef `json:"backend"`
}

type IngressBackendRef struct {
	Type     string                   `json:"type"`
	Service  *IngressServiceBackend   `json:"service,omitempty"`
	Resource *IngressObjectRef        `json:"resource,omitempty"`
}

type IngressServiceBackend struct {
	Name       string `json:"name"`
	PortName   string `json:"portName,omitempty"`
	PortNumber int32  `json:"portNumber,omitempty"`
}

type IngressObjectRef struct {
	APIGroup  string `json:"apiGroup,omitempty"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type IngressTLSDTO struct {
	SecretName string   `json:"secretName"`
	Hosts      []string `json:"hosts,omitempty"`
}

type IngressStatusDTO struct {
	LoadBalancer []string `json:"loadBalancer,omitempty"`
}
