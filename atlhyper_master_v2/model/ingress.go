// atlhyper_master_v2/model/ingress.go
// Ingress Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// IngressItem Ingress 列表项（扁平）
type IngressItem struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	ServicePort string `json:"servicePort"`
	TLS         bool   `json:"tls"`
	CreatedAt   string `json:"createdAt"`
}

// IngressOverviewCards Ingress 概览统计
type IngressOverviewCards struct {
	TotalIngresses int `json:"totalIngresses"`
	UsedHosts      int `json:"usedHosts"`
	TLSCerts       int `json:"tlsCerts"`
	TotalPaths     int `json:"totalPaths"`
}

// IngressOverview Ingress 概览
type IngressOverview struct {
	Cards IngressOverviewCards `json:"cards"`
	Rows  []IngressItem        `json:"rows"`
}

// IngressDetail Ingress 详情（扁平 + 嵌套 spec/status）
type IngressDetail struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Class     string `json:"class,omitempty"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age,omitempty"`

	// 摘要
	Hosts        []string `json:"hosts,omitempty"`
	TLSEnabled   bool     `json:"tlsEnabled"`
	LoadBalancer []string `json:"loadBalancer,omitempty"`

	// 规格与状态（保留原始结构）
	Spec   interface{} `json:"spec,omitempty"`
	Status interface{} `json:"status,omitempty"`

	// 注解
	Annotations map[string]string `json:"annotations,omitempty"`
}
