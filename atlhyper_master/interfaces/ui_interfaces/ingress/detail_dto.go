// ui_interfaces/ingress/dto_detail.go
package ingress

import "time"

// IngressDetailDTO —— 详情页（扁平化但保留关键结构）
type IngressDetailDTO struct {
	// Summary
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Class        string    `json:"class,omitempty"`
	Controller   string    `json:"controller,omitempty"`
	Hosts        []string  `json:"hosts,omitempty"`
	TLSEnabled   bool      `json:"tlsEnabled"`
	LoadBalancer []string  `json:"loadBalancer,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age,omitempty"`

	// Spec（保留结构，便于详情展示）
	Spec IngressSpecDTO `json:"spec"`

	// Status（与 summary.loadBalancer 一致，这里留作扩展）
	Status IngressStatusDTO `json:"status"`

	// 注解（可选）
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ----- Spec/Status DTO -----

type IngressSpecDTO struct {
	IngressClassName         string           `json:"ingressClassName,omitempty"`
	LoadBalancerSourceRanges []string         `json:"loadBalancerSourceRanges,omitempty"`
	DefaultBackend           *BackendRefDTO   `json:"defaultBackend,omitempty"`
	Rules                    []RuleDTO        `json:"rules,omitempty"`
	TLS                      []IngressTLSDTO  `json:"tls,omitempty"`
}

type RuleDTO struct {
	Host  string        `json:"host,omitempty"`
	Paths []HTTPPathDTO `json:"paths"`
}

type HTTPPathDTO struct {
	Path     string       `json:"path,omitempty"`
	PathType string       `json:"pathType,omitempty"`
	Backend  BackendRefDTO`json:"backend"`
}

type BackendRefDTO struct {
	Type     string              `json:"type"` // "Service" | "Resource"
	Service  *ServiceBackendDTO  `json:"service,omitempty"`
	Resource *ObjectRefDTO       `json:"resource,omitempty"`
}

type ServiceBackendDTO struct {
	Name       string `json:"name"`
	PortName   string `json:"portName,omitempty"`
	PortNumber int32  `json:"portNumber,omitempty"`
}

type ObjectRefDTO struct {
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
