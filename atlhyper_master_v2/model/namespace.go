// atlhyper_master_v2/model/namespace.go
// Namespace Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// NamespaceItem Namespace 列表项（扁平）
type NamespaceItem struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	PodCount        int    `json:"podCount"`
	LabelCount      int    `json:"labelCount"`
	AnnotationCount int    `json:"annotationCount"`
	CreatedAt       string `json:"createdAt"`
}

// NamespaceOverviewCards Namespace 概览统计
type NamespaceOverviewCards struct {
	TotalNamespaces int `json:"totalNamespaces"`
	ActiveCount     int `json:"activeCount"`
	Terminating     int `json:"terminating"`
	TotalPods       int `json:"totalPods"`
}

// NamespaceOverview Namespace 概览
type NamespaceOverview struct {
	Cards NamespaceOverviewCards `json:"cards"`
	Rows  []NamespaceItem        `json:"rows"`
}

// NamespaceDetail Namespace 详情（扁平 + 配额/限制范围）
type NamespaceDetail struct {
	// 基本信息
	Name      string `json:"name"`
	Phase     string `json:"phase"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age,omitempty"`

	// 标签与注解
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	LabelCount      int               `json:"labelCount"`
	AnnotationCount int               `json:"annotationCount"`

	// Pod 统计
	Pods          int `json:"pods"`
	PodsRunning   int `json:"podsRunning"`
	PodsPending   int `json:"podsPending"`
	PodsFailed    int `json:"podsFailed"`
	PodsSucceeded int `json:"podsSucceeded"`

	// 工作负载统计
	Deployments  int `json:"deployments"`
	StatefulSets int `json:"statefulSets"`
	DaemonSets   int `json:"daemonSets"`
	Jobs         int `json:"jobs"`
	CronJobs     int `json:"cronJobs"`

	// 网络统计
	Services        int `json:"services"`
	Ingresses       int `json:"ingresses"`
	NetworkPolicies int `json:"networkPolicies"`

	// 配置统计
	ConfigMaps             int `json:"configMaps"`
	Secrets                int `json:"secrets"`
	PersistentVolumeClaims int `json:"persistentVolumeClaims"`
	ServiceAccounts        int `json:"serviceAccounts"`

	// 配额与限制范围
	Quotas      interface{} `json:"quotas,omitempty"`
	LimitRanges interface{} `json:"limitRanges,omitempty"`
}
