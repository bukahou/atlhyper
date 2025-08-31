// ui_interfaces/namespace/dto_detail.go
package namespace

import "time"

// NamespaceDetailDTO —— 详情页扁平化
type NamespaceDetailDTO struct {
	// 基本
	Name      string            `json:"name"`
	Phase     string            `json:"phase"` // Active / Terminating
	CreatedAt time.Time         `json:"createdAt"`
	Age       string            `json:"age,omitempty"`

	// 概览信息
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	LabelCount      int               `json:"labelCount"`
	AnnotationCount int               `json:"annotationCount"`

	// 计数（来自 store 的 counts）
	Pods          int `json:"pods"`
	PodsRunning   int `json:"podsRunning"`
	PodsPending   int `json:"podsPending"`
	PodsFailed    int `json:"podsFailed"`
	PodsSucceeded int `json:"podsSucceeded"`
	Deployments   int `json:"deployments"`
	StatefulSets  int `json:"statefulSets"`
	DaemonSets    int `json:"daemonSets"`
	Jobs          int `json:"jobs"`
	CronJobs      int `json:"cronJobs"`
	Services      int `json:"services"`
	Ingresses     int `json:"ingresses"`
	ConfigMaps    int `json:"configMaps"`
	Secrets       int `json:"secrets"`
	PVCs          int `json:"persistentVolumeClaims"`
	NetworkPolicies int `json:"networkPolicies"`
	ServiceAccounts int `json:"serviceAccounts"`

	// 配额 / 限制（直接沿用精简结构）
	Quotas      []ResourceQuotaDTO `json:"quotas,omitempty"`
	LimitRanges []LimitRangeDTO    `json:"limitRanges,omitempty"`

	// 指标（如有）
	Metrics *NamespaceMetricsDTO `json:"metrics,omitempty"`

	// 徽标（如有）
	Badges []string `json:"badges,omitempty"`
}

// ----- 复用精简结构（DTO 版本） -----

type ResourceQuotaDTO struct {
	Name   string            `json:"name"`
	Scopes []string          `json:"scopes,omitempty"`
	Hard   map[string]string `json:"hard,omitempty"`
	Used   map[string]string `json:"used,omitempty"`
}

type LimitRangeDTO struct {
	Name  string            `json:"name"`
	Items []LimitRangeItemDTO `json:"items"`
}

type LimitRangeItemDTO struct {
	Type                 string            `json:"type"`
	Max                  map[string]string `json:"max,omitempty"`
	Min                  map[string]string `json:"min,omitempty"`
	Default              map[string]string `json:"default,omitempty"`
	DefaultRequest       map[string]string `json:"defaultRequest,omitempty"`
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio,omitempty"`
}

type NamespaceMetricsDTO struct {
	CPU    ResourceAggDTO `json:"cpu"`
	Memory ResourceAggDTO `json:"memory"`
}

type ResourceAggDTO struct {
	Usage     string  `json:"usage"`
	Requests  string  `json:"requests,omitempty"`
	Limits    string  `json:"limits,omitempty"`
	UtilPct   float64 `json:"utilPct,omitempty"`
	UtilBasis string  `json:"utilBasis,omitempty"`
	QuotaHard string  `json:"quotaHard,omitempty"`
	QuotaUsed string  `json:"quotaUsed,omitempty"`
}
