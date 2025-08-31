// model/ingress/ingress.go
package ingress

import "time"

// ====================== 顶层：Ingress（直接发送/上报这个结构体） ======================

type Ingress struct {
	Summary     IngressSummary        `json:"summary"`                 // 概要（列表常用字段）
	Spec        IngressSpec           `json:"spec"`                    // 规格（class/rules/tls 等）
	Status      IngressStatus         `json:"status"`                  // 运行状态（LB 暴露地址）
	Annotations map[string]string     `json:"annotations,omitempty"`   // 重要注解（可选：按需筛选后放入）
}

// ====================== summary ======================

type IngressSummary struct {
	Name        string      `json:"name"`                              // 名称
	Namespace   string      `json:"namespace"`                         // 命名空间
	Class       string      `json:"class,omitempty"`                   // IngressClass 名（spec.ingressClassName 或注解）
	Controller  string      `json:"controller,omitempty"`              // 对应 IngressClass 的 controller（若已查询到）
	Hosts       []string    `json:"hosts,omitempty"`                   // 规则中的 host 去重汇总
	TLSEnabled  bool        `json:"tlsEnabled"`                        // 是否启用 TLS（有 TLS 项即为 true）
	CreatedAt   time.Time   `json:"createdAt"`                         // 创建时间
	Age         string      `json:"age"`                               // 运行时长（派生显示）
	LoadBalancer []string   `json:"loadBalancer,omitempty"`            // 对外地址（IP/Hostname 汇总）
}

// ====================== spec ======================

type IngressSpec struct {
	IngressClassName          string           `json:"ingressClassName,omitempty"`           // 指定的 IngressClass 名
	LoadBalancerSourceRanges  []string         `json:"loadBalancerSourceRanges,omitempty"`   // 允许访问的源网段
	DefaultBackend            *BackendRef      `json:"defaultBackend,omitempty"`             // 默认后端（未匹配规则时）
	Rules                     []Rule           `json:"rules,omitempty"`                      // 规则列表
	TLS                       []IngressTLS     `json:"tls,omitempty"`                        // TLS 配置
}

// 规则（按 host 分组；下挂多条 path）
type Rule struct {
	Host   string     `json:"host,omitempty"`     // 规则 host，可为空（表示所有 host）
	Paths  []HTTPPath `json:"paths"`              // 路径列表
}

// 单条路径
type HTTPPath struct {
	Path     string     `json:"path,omitempty"`        // 路径（如 /、/api）
	PathType string     `json:"pathType,omitempty"`    // ImplementationSpecific/Exact/Prefix
	Backend  BackendRef `json:"backend"`               // 后端引用
}

// 后端引用：兼容 Service 与 Resource 两种 backend
type BackendRef struct {
	Type     string             `json:"type"`                 // "Service" | "Resource"
	Service  *ServiceBackend    `json:"service,omitempty"`    // Type=Service 时有效
	Resource *ObjectRef         `json:"resource,omitempty"`   // Type=Resource 时有效（本地同命名空间对象）
}

// Service 后端
type ServiceBackend struct {
	Name       string `json:"name"`                 // Service 名
	PortName   string `json:"portName,omitempty"`   // 端口名（可选）
	PortNumber int32  `json:"portNumber,omitempty"` // 端口号（可选）
}

// Resource 后端（TypedLocalObjectReference 等价信息）
type ObjectRef struct {
	APIGroup  string `json:"apiGroup,omitempty"` // API 组（如 "gateway.networking.k8s.io"）
	Kind      string `json:"kind"`               // 资源种类
	Name      string `json:"name"`               // 名称
	Namespace string `json:"namespace,omitempty"`// 命名空间（一般省略=同 ns）
}

// TLS 配置
type IngressTLS struct {
	SecretName string   `json:"secretName"`           // 用于 TLS 的 Secret 名
	Hosts      []string `json:"hosts,omitempty"`      // 适用 host 列表（为空表示全部规则 host）
}

// ====================== status ======================

type IngressStatus struct {
	LoadBalancer []string `json:"loadBalancer,omitempty"` // 暴露的 IP/Hostname 列表（status.loadBalancer.ingress）
}
