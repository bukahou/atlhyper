package cluster

import model_v3 "AtlHyper/model_v3"

// Namespace K8s Namespace 资源模型
type Namespace struct {
	Summary     NamespaceSummary   `json:"summary"`
	Status      NamespaceStatus    `json:"status"`
	Resources   NamespaceResources `json:"resources"`
	Quotas      []ResourceQuota    `json:"quotas,omitempty"`
	LimitRanges []LimitRange       `json:"limitRanges,omitempty"`
	Labels      map[string]string  `json:"labels,omitempty"`
	Annotations map[string]string  `json:"annotations,omitempty"`
}

type NamespaceSummary struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`
}

type NamespaceStatus struct {
	Phase string `json:"phase"`
}

type NamespaceResources struct {
	Pods            int `json:"pods"`
	PodsRunning     int `json:"podsRunning"`
	PodsPending     int `json:"podsPending"`
	PodsFailed      int `json:"podsFailed"`
	PodsSucceeded   int `json:"podsSucceeded"`
	Deployments     int `json:"deployments"`
	StatefulSets    int `json:"statefulSets"`
	DaemonSets      int `json:"daemonSets"`
	ReplicaSets     int `json:"replicaSets"`
	Jobs            int `json:"jobs"`
	CronJobs        int `json:"cronJobs"`
	Services        int `json:"services"`
	Ingresses       int `json:"ingresses"`
	NetworkPolicies int `json:"networkPolicies"`
	ConfigMaps      int `json:"configMaps"`
	Secrets         int `json:"secrets"`
	ServiceAccounts int `json:"serviceAccounts"`
	PVCs            int `json:"pvcs"`
}

func (n *Namespace) GetName() string     { return n.Summary.Name }
func (n *Namespace) IsActive() bool      { return n.Status.Phase == "Active" }
func (n *Namespace) IsTerminating() bool { return n.Status.Phase == "Terminating" }

// ConfigMap K8s ConfigMap（只存 Key，不存 Value）
type ConfigMap struct {
	model_v3.CommonMeta
	DataKeys []string `json:"dataKeys,omitempty"`
}

func (c *ConfigMap) KeyCount() int { return len(c.DataKeys) }

// Secret K8s Secret（只存 Key 和类型，不存 Value）
type Secret struct {
	model_v3.CommonMeta
	Type     string   `json:"type"`
	DataKeys []string `json:"dataKeys,omitempty"`
}

func (s *Secret) KeyCount() int              { return len(s.DataKeys) }
func (s *Secret) IsTLSSecret() bool          { return s.Type == "kubernetes.io/tls" }
func (s *Secret) IsDockerConfigSecret() bool { return s.Type == "kubernetes.io/dockerconfigjson" }
