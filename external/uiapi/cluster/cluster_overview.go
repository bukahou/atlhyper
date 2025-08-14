package cluster

import (
	"NeuroController/model/clusteroverview"
	"NeuroController/sync/center/http/uiapi"
)

// ===================== 数据结构（Analysis 页面用） =====================

type HealthStatus string

const (
	HealthHealthy  HealthStatus = "Healthy"
	HealthDegraded HealthStatus = "Degraded"
	HealthUnhealthy HealthStatus = "Unhealthy"
)

type ClusterHealthCard struct {
	Status        HealthStatus `json:"status"`          // Healthy / Degraded / Unhealthy
	Reason        string       `json:"reason"`          // 简短原因说明
	NodeReadyPct  float64      `json:"node_ready_pct"`  // 节点就绪率 0~100
	PodHealthyPct float64      `json:"pod_healthy_pct"` // Pod 健康率 0~100（= (TotalPods-AbnormalPods)/TotalPods*100）
}

type NodesCard struct {
	TotalNodes   int     `json:"total_nodes"`
	ReadyNodes   int     `json:"ready_nodes"`
	NodeReadyPct float64 `json:"node_ready_pct"` // 0~100
}

type ResourceCard struct {
	Percent float64 `json:"percent"` // 0~100
	// 可选扩展：Total/Used（展示用），如需可加：
	// Total human-readable / raw
	// Used  human-readable / raw
	// 这里只保留 Percent，数据来自 overview.resources
}

type OverviewForAnalysis struct {
	HealthCard ClusterHealthCard                    `json:"health_card"`
	NodesCard  NodesCard                            `json:"nodes_card"`
	CPUCard    ResourceCard                         `json:"cpu_card"`
	MemCard    ResourceCard                         `json:"mem_card"`
	Nodes      []clusteroverview.NodeResourceUsage  `json:"nodes"` // 各节点 CPU/内存使用率
}

// ===================== 入口函数 =====================

// LoadOverviewForAnalysis
// 从 uiapi.GetClusterOverview 拉取原始概览数据，并二次加工为 Analysis 页所需的 5 块内容。
func LoadOverviewForAnalysis() (*OverviewForAnalysis, error) {
	ov, err := uiapi.GetClusterOverview()
	if err != nil {
		return nil, err
	}

	// 1) 计算 Node Ready 率
	var nodeReadyPct float64
	if ov.TotalNodes > 0 {
		nodeReadyPct = (float64(ov.ReadyNodes) / float64(ov.TotalNodes)) * 100.0
	}

	// 2) 计算 Pod 健康率（= (TotalPods - AbnormalPods) / TotalPods）
	var podHealthyPct float64
	if ov.TotalPods > 0 {
		podHealthyPct = (float64(ov.TotalPods-ov.AbnormalPods) / float64(ov.TotalPods)) * 100.0
	}

	// 3) 健康判定（简化阈值，可按需调整）
	//   - Healthy:  NodeReadyPct >= 99% 且 PodHealthyPct >= 98%
	//   - Degraded: NodeReadyPct >= 90% 且 PodHealthyPct >= 90%
	//   - 其他情况: Unhealthy
	health := HealthUnhealthy
	reason := ""
	switch {
	case nodeReadyPct >= 99.0 && podHealthyPct >= 98.0:
		health = HealthHealthy
		reason = "All nodes ready and pods mostly healthy"
	case nodeReadyPct >= 90.0 && podHealthyPct >= 90.0:
		health = HealthDegraded
		reason = "Minor degradation (nodes/pods below target)"
	default:
		health = HealthUnhealthy
		// 给出更具体的原因提示
		switch {
		case ov.TotalNodes > 0 && nodeReadyPct < 90.0 && ov.TotalPods > 0 && podHealthyPct < 90.0:
			reason = "Low node ready rate and low pod health rate"
		case ov.TotalNodes > 0 && nodeReadyPct < 90.0:
			reason = "Low node ready rate"
		case ov.TotalPods > 0 && podHealthyPct < 90.0:
			reason = "Low pod health rate"
		default:
			reason = "Insufficient data"
		}
	}

	// 4) CPU / Memory 总使用率（来自 metrics-server 汇总）
	var cpuPct, memPct float64
	if ov.Resources != nil {
		cpuPct = ov.Resources.CPUPercent
		memPct = ov.Resources.MemoryPercent
	}

	res := &OverviewForAnalysis{
		HealthCard: ClusterHealthCard{
			Status:        health,
			Reason:        reason,
			NodeReadyPct:  round2(nodeReadyPct),
			PodHealthyPct: round2(podHealthyPct),
		},
		NodesCard: NodesCard{
			TotalNodes:   ov.TotalNodes,
			ReadyNodes:   ov.ReadyNodes,
			NodeReadyPct: round2(nodeReadyPct),
		},
		CPUCard: ResourceCard{
			Percent: round2(cpuPct),
		},
		MemCard: ResourceCard{
			Percent: round2(memPct),
		},
		Nodes: ov.Nodes, // 已含每节点 CPU/内存使用率与总量
	}

	return res, nil
}

// 简单保留两位小数（显示友好）
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100.0
}

