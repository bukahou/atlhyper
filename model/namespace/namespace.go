// NeuroController/model/namespace/namespace.go
package namespace

import "time"

// ====================== 顶层：Namespace（直接发送这个结构体） ======================

type Namespace struct {
	Summary     NamespaceSummary   `json:"summary"`                // 概要（列表常用字段）
	Counts      NamespaceCounts    `json:"counts"`                 // 资源计数（便于概要/告警）
	Quotas      []ResourceQuota    `json:"quotas,omitempty"`       // ResourceQuota 精简视图
	LimitRanges []LimitRange       `json:"limitRanges,omitempty"`  // LimitRange 精简视图
	Metrics     *NamespaceMetrics  `json:"metrics,omitempty"`      // 聚合指标（可为空）
	Badges      []string           `json:"badges,omitempty"`       // UI 徽标（如 Terminating/QuotaExceeded）
}

// ====================== summary ======================

type NamespaceSummary struct {
	Name        string    `json:"name"`                     // Namespace 名称
	Phase       string    `json:"phase"`                    // Active/Terminating
	CreatedAt   time.Time `json:"createdAt"`                // 创建时间
	Age         string    `json:"age"`                      // 运行时长（派生显示）
	Labels      map[string]string `json:"labels,omitempty"` // 方便筛选
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ====================== counts ======================

type NamespaceCounts struct {
	Pods          int `json:"pods"`           // Pod 总数
	PodsRunning   int `json:"podsRunning"`    // Running
	PodsPending   int `json:"podsPending"`    // Pending
	PodsFailed    int `json:"podsFailed"`     // Failed
	PodsSucceeded int `json:"podsSucceeded"`  // Succeeded
	Deployments   int `json:"deployments"`    // apps/v1 Deployment
	StatefulSets  int `json:"statefulSets"`   // apps/v1 StatefulSet
	DaemonSets    int `json:"daemonSets"`     // apps/v1 DaemonSet
	Jobs          int `json:"jobs"`           // batch/v1 Job
	CronJobs      int `json:"cronJobs"`       // batch/v1 CronJob
	Services      int `json:"services"`       // v1 Service
	Ingresses     int `json:"ingresses"`      // networking.k8s.io/v1 Ingress
	ConfigMaps    int `json:"configMaps"`     // v1 ConfigMap
	Secrets       int `json:"secrets"`        // v1 Secret
	PVCs          int `json:"persistentVolumeClaims"` // v1 PVC
	NetworkPolicies int `json:"networkPolicies"`      // networking.k8s.io/v1 NetworkPolicy
	ServiceAccounts int `json:"serviceAccounts"`      // v1 ServiceAccount
}

// ====================== quotas（ResourceQuota 精简） ======================

type ResourceQuota struct {
	Name   string            `json:"name"`             // RQ 名称
	Scopes []string          `json:"scopes,omitempty"` // 如 Terminating/NotBestEffort 等
	Hard   map[string]string `json:"hard,omitempty"`   // 规格上限（如 "pods":"100","requests.cpu":"4"）
	Used   map[string]string `json:"used,omitempty"`   // 已使用（来自 status）
	// 可根据需要追加 status 条件/时间戳等
}

// ====================== limitRanges（LimitRange 精简） ======================

type LimitRange struct {
	Name  string            `json:"name"`        // LR 名称
	Items []LimitRangeItem  `json:"items"`       // 各类型（Container/Pod/PVC 等）的限制
}

type LimitRangeItem struct {
	Type                    string            `json:"type"`                              // "Container"/"Pod"/"PersistentVolumeClaim"
	Max                     map[string]string `json:"max,omitempty"`                     // 允许的最大值
	Min                     map[string]string `json:"min,omitempty"`                     // 允许的最小值
	Default                 map[string]string `json:"default,omitempty"`                 // 默认 limits
	DefaultRequest          map[string]string `json:"defaultRequest,omitempty"`          // 默认 requests
	MaxLimitRequestRatio    map[string]string `json:"maxLimitRequestRatio,omitempty"`    // 限制 L/R 比
}

// ====================== metrics（聚合指标，可为空） ======================

// 汇总该命名空间中所有运行中 Pod（或可配置范围）的 CPU/内存使用、请求、限制。
// 注意：UtilPct 的分母可按策略选择（limit 优先，其次 request）；或与 ResourceQuota.hard 比较。
type NamespaceMetrics struct {
	CPU       ResourceAgg `json:"cpu"`       // usage/request/limit 及使用率
	Memory    ResourceAgg `json:"memory"`    // usage/request/limit 及使用率
	// 也可扩展存储、网络等
}

type ResourceAgg struct {
	Usage        string  `json:"usage"`                  // 如 "1500m"/"2.3Gi"
	Requests     string  `json:"requests,omitempty"`     // 聚合 requests
	Limits       string  `json:"limits,omitempty"`       // 聚合 limits
	UtilPct      float64 `json:"utilPct,omitempty"`      // usage / (limit|request) * 100（0-100）
	UtilBasis    string  `json:"utilBasis,omitempty"`    // 说明使用率分母的依据："limit" / "request" / "quota"
	QuotaHard    string  `json:"quotaHard,omitempty"`    // 若按配额计算，则填配额上限（如 "requests.cpu":"4" 的合并基准）
	QuotaUsed    string  `json:"quotaUsed,omitempty"`    // 若使用配额状态，也可回填
}
