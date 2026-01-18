package model_v2

import "time"

// ============================================================
// Service 模型（嵌套结构，对齐 model/k8s/service.go）
// ============================================================

// Service K8s Service 资源模型
type Service struct {
	Summary  ServiceSummary    `json:"summary"`
	Spec     ServiceSpec       `json:"spec"`
	Ports    []ServicePort     `json:"ports,omitempty"`
	Selector map[string]string `json:"selector,omitempty"`
	Network  ServiceNetwork    `json:"network"`
	Backends *ServiceBackends  `json:"backends,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ServiceSummary Service 摘要信息
type ServiceSummary struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Type         string    `json:"type"` // ClusterIP, NodePort, LoadBalancer, ExternalName
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age"`
	PortsCount   int       `json:"portsCount"`
	HasSelector  bool      `json:"hasSelector"`
	Badges       []string  `json:"badges,omitempty"`
	ClusterIP    string    `json:"clusterIP,omitempty"`
	ExternalName string    `json:"externalName,omitempty"`
}

// ServiceSpec Service 规格
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

// ServicePort Service 端口定义
type ServicePort struct {
	Name        string `json:"name,omitempty"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"targetPort"`
	NodePort    int32  `json:"nodePort,omitempty"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

// ServiceNetwork Service 网络信息
type ServiceNetwork struct {
	ClusterIPs            []string `json:"clusterIPs,omitempty"`
	ExternalIPs           []string `json:"externalIPs,omitempty"`
	LoadBalancerIngress   []string `json:"loadBalancerIngress,omitempty"`
	IPFamilies            []string `json:"ipFamilies,omitempty"`
	IPFamilyPolicy        string   `json:"ipFamilyPolicy,omitempty"`
	ExternalTrafficPolicy string   `json:"externalTrafficPolicy,omitempty"`
	InternalTrafficPolicy string   `json:"internalTrafficPolicy,omitempty"`
}

// ServiceBackends Service 后端端点信息
type ServiceBackends struct {
	Summary   BackendSummary    `json:"summary"`
	Ports     []EndpointPort    `json:"ports,omitempty"`
	Endpoints []BackendEndpoint `json:"endpoints,omitempty"`
}

// BackendSummary 后端摘要
type BackendSummary struct {
	Ready    int       `json:"ready"`
	NotReady int       `json:"notReady"`
	Total    int       `json:"total"`
	Slices   int       `json:"slices,omitempty"`
	Updated  time.Time `json:"updated,omitempty"`
}

// EndpointPort 端点端口
type EndpointPort struct {
	Name        string `json:"name,omitempty"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

// BackendEndpoint 后端端点
type BackendEndpoint struct {
	Address   string  `json:"address"`
	Ready     bool    `json:"ready"`
	NodeName  string  `json:"nodeName,omitempty"`
	Zone      string  `json:"zone,omitempty"`
	TargetRef *K8sRef `json:"targetRef,omitempty"`
}

// K8sRef K8s 对象引用
type K8sRef struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	UID       string `json:"uid,omitempty"`
}

// ============================================================
// 辅助方法
// ============================================================

// GetName 获取名称
func (s *Service) GetName() string {
	return s.Summary.Name
}

// GetNamespace 获取命名空间
func (s *Service) GetNamespace() string {
	return s.Summary.Namespace
}

// GetType 获取类型
func (s *Service) GetType() string {
	return s.Summary.Type
}

// IsClusterIP 判断是否是 ClusterIP 类型
func (s *Service) IsClusterIP() bool {
	return s.Summary.Type == "ClusterIP"
}

// IsNodePort 判断是否是 NodePort 类型
func (s *Service) IsNodePort() bool {
	return s.Summary.Type == "NodePort"
}

// IsLoadBalancer 判断是否是 LoadBalancer 类型
func (s *Service) IsLoadBalancer() bool {
	return s.Summary.Type == "LoadBalancer"
}

// IsExternalName 判断是否是 ExternalName 类型
func (s *Service) IsExternalName() bool {
	return s.Summary.Type == "ExternalName"
}

// IsHeadless 判断是否是 Headless Service
func (s *Service) IsHeadless() bool {
	return s.Summary.ClusterIP == "None"
}

// HasEndpoints 判断是否有端点
func (s *Service) HasEndpoints() bool {
	return s.Backends != nil && s.Backends.Summary.Total > 0
}

// ============================================================
// Ingress 模型（嵌套结构）
// ============================================================

// Ingress K8s Ingress 资源模型
type Ingress struct {
	Summary     IngressSummary    `json:"summary"`
	Spec        IngressSpec       `json:"spec"`
	Status      IngressStatus     `json:"status"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// IngressSummary Ingress 摘要信息
type IngressSummary struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	CreatedAt    time.Time `json:"createdAt"`
	Age          string    `json:"age"`
	IngressClass string    `json:"ingressClass,omitempty"`
	HostsCount   int       `json:"hostsCount"`
	PathsCount   int       `json:"pathsCount"`
	TLSEnabled   bool      `json:"tlsEnabled"`
	Hosts        []string  `json:"hosts,omitempty"`
}

// IngressSpec Ingress 规格
type IngressSpec struct {
	IngressClassName string          `json:"ingressClassName,omitempty"`
	DefaultBackend   *IngressBackend `json:"defaultBackend,omitempty"`
	Rules            []IngressRule   `json:"rules,omitempty"`
	TLS              []IngressTLS    `json:"tls,omitempty"`
}

// IngressStatus Ingress 状态
type IngressStatus struct {
	LoadBalancer []string `json:"loadBalancer,omitempty"`
}

// IngressBackend Ingress 后端配置
type IngressBackend struct {
	Type     string                 `json:"type"` // "Service" or "Resource"
	Service  *IngressServiceBackend `json:"service,omitempty"`
	Resource *IngressResourceRef    `json:"resource,omitempty"`
}

// IngressServiceBackend Service 类型后端
type IngressServiceBackend struct {
	Name       string `json:"name"`
	PortName   string `json:"portName,omitempty"`
	PortNumber int32  `json:"portNumber,omitempty"`
}

// IngressResourceRef Resource 类型后端
type IngressResourceRef struct {
	APIGroup  string `json:"apiGroup,omitempty"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// IngressRule Ingress 路由规则
type IngressRule struct {
	Host  string        `json:"host,omitempty"`
	Paths []IngressPath `json:"paths"`
}

// IngressPath Ingress 路径配置
type IngressPath struct {
	Path     string          `json:"path"`
	PathType string          `json:"pathType"`
	Backend  *IngressBackend `json:"backend,omitempty"`
}

// IngressTLS Ingress TLS 配置
type IngressTLS struct {
	Hosts      []string `json:"hosts,omitempty"`
	SecretName string   `json:"secretName,omitempty"`
}

// ============================================================
// Ingress 辅助方法
// ============================================================

// GetName 获取名称
func (i *Ingress) GetName() string {
	return i.Summary.Name
}

// GetNamespace 获取命名空间
func (i *Ingress) GetNamespace() string {
	return i.Summary.Namespace
}

// GetHosts 获取所有主机名
func (i *Ingress) GetHosts() []string {
	return i.Summary.Hosts
}

// HasTLS 判断是否启用 TLS
func (i *Ingress) HasTLS() bool {
	return i.Summary.TLSEnabled
}
