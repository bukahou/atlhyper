// model/k8s/namespace.go
// Namespace 资源模型
package k8s

import "time"

// ====================== 顶层：Namespace ======================

type Namespace struct {
	Summary     NamespaceSummary  `json:"summary"`
	Counts      NamespaceCounts   `json:"counts"`
	Quotas      []ResourceQuota   `json:"quotas,omitempty"`
	LimitRanges []LimitRange      `json:"limitRanges,omitempty"`
	Metrics     *NamespaceMetrics `json:"metrics,omitempty"`
	Badges      []string          `json:"badges,omitempty"`
}

// ====================== summary ======================

type NamespaceSummary struct {
	Name        string            `json:"name"`
	Phase       string            `json:"phase"`
	CreatedAt   time.Time         `json:"createdAt"`
	Age         string            `json:"age"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ====================== counts ======================

type NamespaceCounts struct {
	Pods            int `json:"pods"`
	PodsRunning     int `json:"podsRunning"`
	PodsPending     int `json:"podsPending"`
	PodsFailed      int `json:"podsFailed"`
	PodsSucceeded   int `json:"podsSucceeded"`
	Deployments     int `json:"deployments"`
	StatefulSets    int `json:"statefulSets"`
	DaemonSets      int `json:"daemonSets"`
	Jobs            int `json:"jobs"`
	CronJobs        int `json:"cronJobs"`
	Services        int `json:"services"`
	Ingresses       int `json:"ingresses"`
	ConfigMaps      int `json:"configMaps"`
	Secrets         int `json:"secrets"`
	PVCs            int `json:"persistentVolumeClaims"`
	NetworkPolicies int `json:"networkPolicies"`
	ServiceAccounts int `json:"serviceAccounts"`
}

// ====================== quotas ======================

type ResourceQuota struct {
	Name   string            `json:"name"`
	Scopes []string          `json:"scopes,omitempty"`
	Hard   map[string]string `json:"hard,omitempty"`
	Used   map[string]string `json:"used,omitempty"`
}

// ====================== limitRanges ======================

type LimitRange struct {
	Name  string           `json:"name"`
	Items []LimitRangeItem `json:"items"`
}

type LimitRangeItem struct {
	Type                 string            `json:"type"`
	Max                  map[string]string `json:"max,omitempty"`
	Min                  map[string]string `json:"min,omitempty"`
	Default              map[string]string `json:"default,omitempty"`
	DefaultRequest       map[string]string `json:"defaultRequest,omitempty"`
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio,omitempty"`
}

// ====================== metrics ======================

type NamespaceMetrics struct {
	CPU    ResourceAgg `json:"cpu"`
	Memory ResourceAgg `json:"memory"`
}

type ResourceAgg struct {
	Usage     string  `json:"usage"`
	Requests  string  `json:"requests,omitempty"`
	Limits    string  `json:"limits,omitempty"`
	UtilPct   float64 `json:"utilPct,omitempty"`
	UtilBasis string  `json:"utilBasis,omitempty"`
	QuotaHard string  `json:"quotaHard,omitempty"`
	QuotaUsed string  `json:"quotaUsed,omitempty"`
}
