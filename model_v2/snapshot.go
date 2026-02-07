package model_v2

import "time"

// ============================================================
// ClusterSnapshot 集群快照
// ============================================================

// ClusterSnapshot 集群快照
//
// Agent 采集的完整集群状态，通过 HTTP 推送给 Master。
// 包含所有 K8s 资源的当前状态，用于展示和分析。
type ClusterSnapshot struct {
	// 标识
	ClusterID string    `json:"cluster_id"` // 集群 ID
	FetchedAt time.Time `json:"fetched_at"` // 采集时间

	// ========== 工作负载 ==========
	Pods         []Pod         `json:"pods"`
	Deployments  []Deployment  `json:"deployments"`
	StatefulSets []StatefulSet `json:"statefulsets"`
	DaemonSets   []DaemonSet   `json:"daemonsets"`
	ReplicaSets  []ReplicaSet  `json:"replicasets,omitempty"`
	Jobs         []Job         `json:"jobs,omitempty"`
	CronJobs     []CronJob     `json:"cronjobs,omitempty"`

	// ========== 网络 ==========
	Services  []Service `json:"services"`
	Ingresses []Ingress `json:"ingresses"`

	// ========== 配置 ==========
	Namespaces []Namespace `json:"namespaces"`
	ConfigMaps []ConfigMap `json:"configmaps"`
	Secrets    []Secret    `json:"secrets,omitempty"`

	// ========== 策略与配额 ==========
	ResourceQuotas  []ResourceQuota  `json:"resourceQuotas,omitempty"`
	LimitRanges     []LimitRange     `json:"limitRanges,omitempty"`
	NetworkPolicies []NetworkPolicy  `json:"networkPolicies,omitempty"`
	ServiceAccounts []ServiceAccount `json:"serviceAccounts,omitempty"`

	// ========== 存储 ==========
	PersistentVolumes      []PersistentVolume      `json:"pvs,omitempty"`
	PersistentVolumeClaims []PersistentVolumeClaim `json:"pvcs,omitempty"`

	// ========== 集群 ==========
	Nodes  []Node  `json:"nodes"`
	Events []Event `json:"events"`

	// ========== 硬件指标 ==========
	// NodeMetrics 存储每个节点的详细硬件指标
	// key: 节点名称, value: 节点指标快照
	NodeMetrics map[string]*NodeMetricsSnapshot `json:"node_metrics,omitempty"`

	// ========== SLO 指标 ==========
	// SLOData OTel Collector 采集的 SLO 指标数据（可选）
	// 包含服务网格(Linkerd) + 入口(Ingress Controller) + 拓扑(Edge) 三层数据
	SLOData *SLOSnapshot `json:"slo_data,omitempty"`

	// ========== 摘要 ==========
	Summary ClusterSummary `json:"summary"`
}

// ============================================================
// ClusterSummary 集群摘要
// ============================================================

// ClusterSummary 集群摘要统计
//
// 快照的汇总信息，用于快速展示集群状态。
// 避免前端遍历完整数据计算统计。
type ClusterSummary struct {
	// Node 统计
	TotalNodes int `json:"total_nodes"` // 总节点数
	ReadyNodes int `json:"ready_nodes"` // 就绪节点数

	// Pod 统计
	TotalPods   int `json:"total_pods"`   // 总 Pod 数
	RunningPods int `json:"running_pods"` // Running 状态
	PendingPods int `json:"pending_pods"` // Pending 状态
	FailedPods  int `json:"failed_pods"`  // Failed 状态

	// Deployment 统计
	TotalDeployments   int `json:"total_deployments"`   // 总 Deployment 数
	HealthyDeployments int `json:"healthy_deployments"` // 健康 Deployment 数

	// 其他资源统计
	TotalStatefulSets int `json:"total_statefulsets"`
	TotalDaemonSets   int `json:"total_daemonsets"`
	TotalServices     int `json:"total_services"`
	TotalIngresses    int `json:"total_ingresses"`
	TotalNamespaces   int `json:"total_namespaces"`

	// Event 统计
	TotalEvents   int `json:"total_events"`   // 总事件数
	WarningEvents int `json:"warning_events"` // Warning 事件数
}

// ============================================================
// ClusterSnapshot 辅助方法
// ============================================================

// GenerateSummary 从快照数据生成摘要
//
// 遍历快照中的资源，统计各类指标。
func (s *ClusterSnapshot) GenerateSummary() ClusterSummary {
	summary := ClusterSummary{
		TotalNodes:        len(s.Nodes),
		TotalPods:         len(s.Pods),
		TotalDeployments:  len(s.Deployments),
		TotalStatefulSets: len(s.StatefulSets),
		TotalDaemonSets:   len(s.DaemonSets),
		TotalServices:     len(s.Services),
		TotalIngresses:    len(s.Ingresses),
		TotalNamespaces:   len(s.Namespaces),
		TotalEvents:       len(s.Events),
	}

	// 统计 Node 状态
	for _, node := range s.Nodes {
		if node.IsReady() {
			summary.ReadyNodes++
		}
	}

	// 统计 Pod 状态
	for _, pod := range s.Pods {
		switch pod.Status.Phase {
		case "Running":
			summary.RunningPods++
		case "Pending":
			summary.PendingPods++
		case "Failed":
			summary.FailedPods++
		}
	}

	// 统计 Deployment 状态
	for _, deploy := range s.Deployments {
		if deploy.IsHealthy() {
			summary.HealthyDeployments++
		}
	}

	// 统计 Event 状态
	for _, event := range s.Events {
		if event.IsWarning() {
			summary.WarningEvents++
		}
	}

	return summary
}

// GetNodeReadyPercent 获取节点就绪百分比
func (s *ClusterSnapshot) GetNodeReadyPercent() float64 {
	if s.Summary.TotalNodes == 0 {
		return 0
	}
	return float64(s.Summary.ReadyNodes) / float64(s.Summary.TotalNodes) * 100
}

// GetPodRunningPercent 获取 Pod 运行百分比
func (s *ClusterSnapshot) GetPodRunningPercent() float64 {
	if s.Summary.TotalPods == 0 {
		return 0
	}
	return float64(s.Summary.RunningPods) / float64(s.Summary.TotalPods) * 100
}

// GetDeploymentHealthyPercent 获取 Deployment 健康百分比
func (s *ClusterSnapshot) GetDeploymentHealthyPercent() float64 {
	if s.Summary.TotalDeployments == 0 {
		return 0
	}
	return float64(s.Summary.HealthyDeployments) / float64(s.Summary.TotalDeployments) * 100
}
