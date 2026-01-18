package model_v2

// ============================================================
// Namespace 模型（嵌套结构）
// ============================================================

// Namespace K8s Namespace 资源模型
//
// Namespace 提供资源隔离的虚拟集群。
// 采用嵌套结构，包含完整的资源统计信息。
type Namespace struct {
	// 摘要信息
	Summary NamespaceSummary `json:"summary"`

	// 状态
	Status NamespaceStatus `json:"status"`

	// 资源统计
	Resources NamespaceResources `json:"resources"`

	// 配额
	Quotas []ResourceQuota `json:"quotas,omitempty"`

	// 限制范围
	LimitRanges []LimitRange `json:"limitRanges,omitempty"`

	// 标签和注解
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NamespaceSummary Namespace 摘要
type NamespaceSummary struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`
}

// NamespaceStatus Namespace 状态
type NamespaceStatus struct {
	Phase string `json:"phase"` // Active, Terminating
}

// NamespaceResources Namespace 资源统计
type NamespaceResources struct {
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
	ReplicaSets  int `json:"replicaSets"`
	Jobs         int `json:"jobs"`
	CronJobs     int `json:"cronJobs"`

	// 网络统计
	Services        int `json:"services"`
	Ingresses       int `json:"ingresses"`
	NetworkPolicies int `json:"networkPolicies"`

	// 配置统计
	ConfigMaps      int `json:"configMaps"`
	Secrets         int `json:"secrets"`
	ServiceAccounts int `json:"serviceAccounts"`

	// 存储统计
	PVCs int `json:"pvcs"`
}

// ============================================================
// Namespace 辅助方法
// ============================================================

// GetName 获取名称
func (n *Namespace) GetName() string {
	return n.Summary.Name
}

// IsActive 判断 Namespace 是否活跃
func (n *Namespace) IsActive() bool {
	return n.Status.Phase == "Active"
}

// IsTerminating 判断 Namespace 是否正在删除
func (n *Namespace) IsTerminating() bool {
	return n.Status.Phase == "Terminating"
}

// GetLabelCount 获取标签数量
func (n *Namespace) GetLabelCount() int {
	return len(n.Labels)
}

// GetAnnotationCount 获取注解数量
func (n *Namespace) GetAnnotationCount() int {
	return len(n.Annotations)
}

// ============================================================
// ConfigMap 模型
// ============================================================

// ConfigMap K8s ConfigMap 资源模型
//
// ConfigMap 存储非机密的配置数据。
// 注意：只存储 Key 列表，不存储 Value（避免敏感信息泄露）。
type ConfigMap struct {
	CommonMeta

	// 数据键列表（不存储值）
	DataKeys []string `json:"data_keys,omitempty"`
}

// KeyCount 返回数据键数量
func (c *ConfigMap) KeyCount() int {
	return len(c.DataKeys)
}

// ============================================================
// Secret 模型
// ============================================================

// Secret K8s Secret 资源模型
//
// Secret 存储密码、OAuth 令牌、SSH 密钥等敏感信息。
// 注意：只存储 Key 列表和类型，不存储 Value。
type Secret struct {
	CommonMeta

	// 类型
	Type string `json:"type"` // Opaque, kubernetes.io/tls, kubernetes.io/dockerconfigjson 等

	// 数据键列表（不存储值）
	DataKeys []string `json:"data_keys,omitempty"`
}

// KeyCount 返回数据键数量
func (s *Secret) KeyCount() int {
	return len(s.DataKeys)
}

// IsTLSSecret 判断是否是 TLS 类型
func (s *Secret) IsTLSSecret() bool {
	return s.Type == "kubernetes.io/tls"
}

// IsDockerConfigSecret 判断是否是 Docker 配置类型
func (s *Secret) IsDockerConfigSecret() bool {
	return s.Type == "kubernetes.io/dockerconfigjson"
}
