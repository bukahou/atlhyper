// model/k8s/ingress.go
// Ingress 资源模型
package k8s

import "time"

// ====================== 顶层：Ingress ======================

type Ingress struct {
	Summary     IngressSummary    `json:"summary"`
	Spec        IngressSpec       `json:"spec"`
	Status      IngressStatus     `json:"status"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ====================== summary ======================

type IngressSummary struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Class        string    `json:"class,omitempty"`
	Controller   string    `json:"controller,omitempty"`
	Hosts        []string  `json:"hosts,omitempty"`
	TLSEnabled   bool      `json:"tlsEnabled"`
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age"`
	LoadBalancer []string  `json:"loadBalancer,omitempty"`
}

// ====================== spec ======================

type IngressSpec struct {
	IngressClassName         string       `json:"ingressClassName,omitempty"`
	LoadBalancerSourceRanges []string     `json:"loadBalancerSourceRanges,omitempty"`
	DefaultBackend           *BackendRef  `json:"defaultBackend,omitempty"`
	Rules                    []Rule       `json:"rules,omitempty"`
	TLS                      []IngressTLS `json:"tls,omitempty"`
}

type Rule struct {
	Host  string     `json:"host,omitempty"`
	Paths []HTTPPath `json:"paths"`
}

type HTTPPath struct {
	Path     string     `json:"path,omitempty"`
	PathType string     `json:"pathType,omitempty"`
	Backend  BackendRef `json:"backend"`
}

type BackendRef struct {
	Type     string          `json:"type"`
	Service  *ServiceBackend `json:"service,omitempty"`
	Resource *ObjectRef      `json:"resource,omitempty"`
}

type ServiceBackend struct {
	Name       string `json:"name"`
	PortName   string `json:"portName,omitempty"`
	PortNumber int32  `json:"portNumber,omitempty"`
}

type ObjectRef struct {
	APIGroup  string `json:"apiGroup,omitempty"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type IngressTLS struct {
	SecretName string   `json:"secretName"`
	Hosts      []string `json:"hosts,omitempty"`
}

// ====================== status ======================

type IngressStatus struct {
	LoadBalancer []string `json:"loadBalancer,omitempty"`
}
