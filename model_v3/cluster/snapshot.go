package cluster

import (
	"time"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// ClusterSnapshot Agent 采集的完整集群状态
//
// NodeMetrics 和 SLO 不再随快照上报，改由 Master 从 ClickHouse 查询。
type ClusterSnapshot struct {
	ClusterID string    `json:"clusterId"`
	FetchedAt time.Time `json:"fetchedAt"`

	// 工作负载
	Pods         []Pod         `json:"pods"`
	Deployments  []Deployment  `json:"deployments"`
	StatefulSets []StatefulSet `json:"statefulSets"`
	DaemonSets   []DaemonSet   `json:"daemonSets"`
	ReplicaSets  []ReplicaSet  `json:"replicaSets,omitempty"`
	Jobs         []Job         `json:"jobs,omitempty"`
	CronJobs     []CronJob     `json:"cronJobs,omitempty"`

	// 网络
	Services  []Service `json:"services"`
	Ingresses []Ingress `json:"ingresses"`

	// 配置
	Namespaces []Namespace `json:"namespaces"`
	ConfigMaps []ConfigMap `json:"configMaps"`
	Secrets    []Secret    `json:"secrets,omitempty"`

	// 策略与配额
	ResourceQuotas  []ResourceQuota  `json:"resourceQuotas,omitempty"`
	LimitRanges     []LimitRange     `json:"limitRanges,omitempty"`
	NetworkPolicies []NetworkPolicy  `json:"networkPolicies,omitempty"`
	ServiceAccounts []ServiceAccount `json:"serviceAccounts,omitempty"`

	// 存储
	PersistentVolumes      []PersistentVolume      `json:"pvs,omitempty"`
	PersistentVolumeClaims []PersistentVolumeClaim `json:"pvcs,omitempty"`

	// 集群
	Nodes  []Node  `json:"nodes"`
	Events []Event `json:"events"`

	// OTel 可观测性快照（Agent 从 ClickHouse 定期聚合，随快照上报）
	OTel *OTelSnapshot `json:"otel,omitempty"`

	// 摘要
	Summary ClusterSummary `json:"summary"`
}

// ClusterSummary 集群摘要统计
type ClusterSummary struct {
	TotalNodes         int `json:"totalNodes"`
	ReadyNodes         int `json:"readyNodes"`
	TotalPods          int `json:"totalPods"`
	RunningPods        int `json:"runningPods"`
	PendingPods        int `json:"pendingPods"`
	FailedPods         int `json:"failedPods"`
	TotalDeployments   int `json:"totalDeployments"`
	HealthyDeployments int `json:"healthyDeployments"`
	TotalStatefulSets  int `json:"totalStatefulSets"`
	TotalDaemonSets    int `json:"totalDaemonSets"`
	TotalServices      int `json:"totalServices"`
	TotalIngresses     int `json:"totalIngresses"`
	TotalNamespaces    int `json:"totalNamespaces"`
	TotalEvents        int `json:"totalEvents"`
	WarningEvents      int `json:"warningEvents"`
}

func (s *ClusterSnapshot) GenerateSummary() ClusterSummary {
	sum := ClusterSummary{
		TotalNodes: len(s.Nodes), TotalPods: len(s.Pods),
		TotalDeployments: len(s.Deployments), TotalStatefulSets: len(s.StatefulSets),
		TotalDaemonSets: len(s.DaemonSets), TotalServices: len(s.Services),
		TotalIngresses: len(s.Ingresses), TotalNamespaces: len(s.Namespaces),
		TotalEvents: len(s.Events),
	}
	for _, n := range s.Nodes {
		if n.IsReady() {
			sum.ReadyNodes++
		}
	}
	for _, p := range s.Pods {
		switch p.Status.Phase {
		case "Running":
			sum.RunningPods++
		case "Pending":
			sum.PendingPods++
		case "Failed":
			sum.FailedPods++
		}
	}
	for _, d := range s.Deployments {
		if d.IsHealthy() {
			sum.HealthyDeployments++
		}
	}
	for _, e := range s.Events {
		if e.IsWarning() {
			sum.WarningEvents++
		}
	}
	return sum
}

func (s *ClusterSnapshot) GetNodeReadyPercent() float64 {
	if s.Summary.TotalNodes == 0 {
		return 0
	}
	return float64(s.Summary.ReadyNodes) / float64(s.Summary.TotalNodes) * 100
}

func (s *ClusterSnapshot) GetPodRunningPercent() float64 {
	if s.Summary.TotalPods == 0 {
		return 0
	}
	return float64(s.Summary.RunningPods) / float64(s.Summary.TotalPods) * 100
}

// OTelSnapshot OTel 可观测性快照（Agent 从 ClickHouse 定期聚合）
//
// 包含两类数据：
//   - 标量摘要（15 个字段）：用于概览卡片展示
//   - Dashboard 列表（8 个字段）：用于 Dashboard 页面直读，避免 Command 机制的延迟
//
// 随快照一起上报给 Master，Master 内存直读即可返回前端。
// 详细数据（Trace Detail、Log Query 等）仍通过 Command 机制按需查询。
type OTelSnapshot struct {
	// ===== 标量摘要 =====

	// APM 服务概览
	TotalServices   int     `json:"totalServices"`
	HealthyServices int     `json:"healthyServices"`
	TotalRPS        float64 `json:"totalRps"`
	AvgSuccessRate  float64 `json:"avgSuccessRate"`
	AvgP99Ms        float64 `json:"avgP99Ms"`

	// SLO 概览
	IngressServices int     `json:"ingressServices"`
	IngressAvgRPS   float64 `json:"ingressAvgRps"`
	MeshServices    int     `json:"meshServices"`
	MeshAvgMTLS     float64 `json:"meshAvgMtls"`

	// 基础设施指标概览
	MonitoredNodes int     `json:"monitoredNodes"`
	AvgCPUPct      float64 `json:"avgCpuPct"`
	AvgMemPct      float64 `json:"avgMemPct"`
	MaxCPUPct      float64 `json:"maxCpuPct"`
	MaxMemPct      float64 `json:"maxMemPct"`

	// ===== Dashboard 列表数据 =====

	// Metrics Dashboard
	MetricsSummary *metrics.Summary      `json:"metricsSummary,omitempty"`
	MetricsNodes   []metrics.NodeMetrics `json:"metricsNodes,omitempty"`

	// APM Dashboard
	APMServices []apm.APMService `json:"apmServices,omitempty"`
	APMTopology *apm.Topology    `json:"apmTopology,omitempty"`

	// SLO Dashboard
	SLOSummary  *slo.SLOSummary  `json:"sloSummary,omitempty"`
	SLOIngress  []slo.IngressSLO `json:"sloIngress,omitempty"`
	SLOServices []slo.ServiceSLO `json:"sloServices,omitempty"`
	SLOEdges    []slo.ServiceEdge `json:"sloEdges,omitempty"`
}
