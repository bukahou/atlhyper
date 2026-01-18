package model_v2

// ============================================================
// ResourceQuota 资源配额模型
// ============================================================

// ResourceQuota K8s ResourceQuota 资源模型
//
// ResourceQuota 用于限制 Namespace 中的资源使用总量。
type ResourceQuota struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`

	// 配额范围
	Scopes []string `json:"scopes,omitempty"`

	// 硬限制
	Hard map[string]string `json:"hard,omitempty"`

	// 已使用量
	Used map[string]string `json:"used,omitempty"`
}

// ============================================================
// LimitRange 限制范围模型
// ============================================================

// LimitRange K8s LimitRange 资源模型
//
// LimitRange 用于限制 Namespace 中容器/Pod 的资源使用范围。
type LimitRange struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`

	// 限制项
	Items []LimitRangeItem `json:"items"`
}

// LimitRangeItem 限制范围项
type LimitRangeItem struct {
	// 类型: Container, Pod, PersistentVolumeClaim
	Type string `json:"type"`

	// 默认值
	Default map[string]string `json:"default,omitempty"`

	// 默认请求值
	DefaultRequest map[string]string `json:"defaultRequest,omitempty"`

	// 最大值
	Max map[string]string `json:"max,omitempty"`

	// 最小值
	Min map[string]string `json:"min,omitempty"`

	// 最大限制/请求比例
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio,omitempty"`
}

// ============================================================
// NetworkPolicy 网络策略模型
// ============================================================

// NetworkPolicy K8s NetworkPolicy 资源模型
//
// NetworkPolicy 用于控制 Pod 之间的网络流量。
type NetworkPolicy struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`

	// Pod 选择器（JSON 字符串形式）
	PodSelector string `json:"podSelector,omitempty"`

	// 策略类型
	PolicyTypes []string `json:"policyTypes,omitempty"`

	// 入站规则数量
	IngressRuleCount int `json:"ingressRuleCount"`

	// 出站规则数量
	EgressRuleCount int `json:"egressRuleCount"`
}

// ============================================================
// ServiceAccount 服务账号模型
// ============================================================

// ServiceAccount K8s ServiceAccount 资源模型
//
// ServiceAccount 为 Pod 提供身份标识。
type ServiceAccount struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`

	// 关联的 Secret 数量
	SecretsCount int `json:"secretsCount"`

	// ImagePullSecrets 数量
	ImagePullSecretsCount int `json:"imagePullSecretsCount"`

	// 是否自动挂载 token
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty"`
}
