package cluster

// ResourceQuota K8s ResourceQuota 资源模型
type ResourceQuota struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	CreatedAt   string            `json:"createdAt"`
	Age         string            `json:"age"`
	Scopes      []string          `json:"scopes,omitempty"`
	Hard        map[string]string `json:"hard,omitempty"`
	Used        map[string]string `json:"used,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// LimitRange K8s LimitRange 资源模型
type LimitRange struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	CreatedAt   string            `json:"createdAt"`
	Age         string            `json:"age"`
	Items       []LimitRangeItem  `json:"items"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type LimitRangeItem struct {
	Type                 string            `json:"type"`
	Default              map[string]string `json:"default,omitempty"`
	DefaultRequest       map[string]string `json:"defaultRequest,omitempty"`
	Max                  map[string]string `json:"max,omitempty"`
	Min                  map[string]string `json:"min,omitempty"`
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio,omitempty"`
}

// NetworkPolicy K8s NetworkPolicy 资源模型
type NetworkPolicy struct {
	Name             string              `json:"name"`
	Namespace        string              `json:"namespace"`
	CreatedAt        string              `json:"createdAt"`
	Age              string              `json:"age"`
	PodSelector      string              `json:"podSelector,omitempty"`
	PolicyTypes      []string            `json:"policyTypes,omitempty"`
	IngressRuleCount int                 `json:"ingressRuleCount"`
	EgressRuleCount  int                 `json:"egressRuleCount"`
	IngressRules     []NetworkPolicyRule `json:"ingressRules,omitempty"`
	EgressRules      []NetworkPolicyRule `json:"egressRules,omitempty"`
	Labels           map[string]string   `json:"labels,omitempty"`
	Annotations      map[string]string   `json:"annotations,omitempty"`
}

type NetworkPolicyRule struct {
	Peers []NetworkPolicyPeer `json:"peers,omitempty"`
	Ports []NetworkPolicyPort `json:"ports,omitempty"`
}

type NetworkPolicyPeer struct {
	Type     string   `json:"type"`
	Selector string   `json:"selector,omitempty"`
	CIDR     string   `json:"cidr,omitempty"`
	Except   []string `json:"except,omitempty"`
}

type NetworkPolicyPort struct {
	Protocol string `json:"protocol"`
	Port     string `json:"port"`
	EndPort  *int32 `json:"endPort,omitempty"`
}

// ServiceAccount K8s ServiceAccount 资源模型
type ServiceAccount struct {
	Name                         string            `json:"name"`
	Namespace                    string            `json:"namespace"`
	CreatedAt                    string            `json:"createdAt"`
	Age                          string            `json:"age"`
	SecretsCount                 int               `json:"secretsCount"`
	ImagePullSecretsCount        int               `json:"imagePullSecretsCount"`
	AutomountServiceAccountToken *bool             `json:"automountServiceAccountToken,omitempty"`
	SecretNames                  []string          `json:"secretNames,omitempty"`
	ImagePullSecretNames         []string          `json:"imagePullSecretNames,omitempty"`
	Labels                       map[string]string `json:"labels,omitempty"`
	Annotations                  map[string]string `json:"annotations,omitempty"`
}
