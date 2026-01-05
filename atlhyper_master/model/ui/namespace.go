// atlhyper_master/dto/ui/namespace.go
// Namespace UI DTOs
package ui

import "time"

// ====================== Overview ======================

// NamespaceOverviewDTO - 概览
type NamespaceOverviewDTO struct {
	Cards NamespaceOverviewCards `json:"cards"`
	Rows  []NamespaceRowDTO      `json:"rows"`
}

type NamespaceOverviewCards struct {
	TotalNamespaces int `json:"totalNamespaces"`
	ActiveCount     int `json:"activeCount"`
	Terminating     int `json:"terminating"`
	TotalPods       int `json:"totalPods"`
}

type NamespaceRowDTO struct {
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	PodCount        int       `json:"podCount"`
	LabelCount      int       `json:"labelCount"`
	AnnotationCount int       `json:"annotationCount"`
	CreatedAt       time.Time `json:"createdAt"`
}

// ====================== Detail ======================

// NamespaceDetailDTO - 详情页
type NamespaceDetailDTO struct {
	Name      string            `json:"name"`
	Phase     string            `json:"phase"`
	CreatedAt time.Time         `json:"createdAt"`
	Age       string            `json:"age,omitempty"`

	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	LabelCount      int               `json:"labelCount"`
	AnnotationCount int               `json:"annotationCount"`

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

	Quotas      []NamespaceQuotaDTO      `json:"quotas,omitempty"`
	LimitRanges []NamespaceLimitRangeDTO `json:"limitRanges,omitempty"`
	Metrics     *NamespaceMetricsDTO     `json:"metrics,omitempty"`
	Badges      []string                 `json:"badges,omitempty"`
}

type NamespaceQuotaDTO struct {
	Name   string            `json:"name"`
	Scopes []string          `json:"scopes,omitempty"`
	Hard   map[string]string `json:"hard,omitempty"`
	Used   map[string]string `json:"used,omitempty"`
}

type NamespaceLimitRangeDTO struct {
	Name  string                      `json:"name"`
	Items []NamespaceLimitRangeItem   `json:"items"`
}

type NamespaceLimitRangeItem struct {
	Type                 string            `json:"type"`
	Max                  map[string]string `json:"max,omitempty"`
	Min                  map[string]string `json:"min,omitempty"`
	Default              map[string]string `json:"default,omitempty"`
	DefaultRequest       map[string]string `json:"defaultRequest,omitempty"`
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio,omitempty"`
}

type NamespaceMetricsDTO struct {
	CPU    NamespaceResourceAgg `json:"cpu"`
	Memory NamespaceResourceAgg `json:"memory"`
}

type NamespaceResourceAgg struct {
	Usage     string  `json:"usage"`
	Requests  string  `json:"requests,omitempty"`
	Limits    string  `json:"limits,omitempty"`
	UtilPct   float64 `json:"utilPct,omitempty"`
	UtilBasis string  `json:"utilBasis,omitempty"`
	QuotaHard string  `json:"quotaHard,omitempty"`
	QuotaUsed string  `json:"quotaUsed,omitempty"`
}
