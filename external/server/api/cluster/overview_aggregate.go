package cluster

import (
	"context"
	"time"

	"NeuroController/model/clusteroverview"
)

// ---------------------- 最终返回体（给 Analysis 页） ----------------------

type AnalysisPayload struct {
	// 顶部卡片
	HealthCard  ClusterHealthCard `json:"health_card"`  // 集群健康卡片
	NodesCard   NodesCard         `json:"nodes_card"`   // Nodes 的 Ready 率卡片
	CPUCard     ResourceCard      `json:"cpu_card"`     // 集群 CPU 总使用率卡片（0~100）
	MemCard     ResourceCard      `json:"mem_card"`     // 集群内存总使用率卡片（0~100）

	// Alerts
	AlertsTotal  int                `json:"alerts_total"`   // 一天内 eventlog 条数
	AlertTrends  AlertTrendsSeries  `json:"alert_trends"`   // 1 天趋势（critical/warning/info）
	RecentAlerts []RecentAlertRow   `json:"recent_alerts"`  // 轻量表格数据

	// 物理机 30 分钟折线（1 分钟粒度）
	CPUSeries  [][]float64 `json:"cpu_series"`  // [ [tsMillis, percent], ... ]
	MemSeries  [][]float64 `json:"mem_series"`  // [ [tsMillis, percent], ... ]
	TempSeries [][]float64 `json:"temp_series"` // [ [tsMillis, ℃], ... ] 取每分钟最高 CPU 温

	// 各节点 CPU/内存使用率（来自 metrics-server 二次加工）
	NodeUsages []clusteroverview.NodeResourceUsage `json:"node_usages"`
}

// ---------------------- 聚合函数：返回最终数据 ----------------------

// BuildClusterOverviewAggregated
// 返回 Analysis 页所需的：
// - 集群健康/Nodes/总 CPU 使用率/总内存使用率卡片
// - 一天内事件总数、趋势、Recent 表格
// - 30 分钟 CPU/内存/温度折线
// - 各节点 CPU/内存使用率
func BuildClusterOverviewAggregated(ctx context.Context, clusterID string) (*AnalysisPayload, error) {
	// 1) 概览（已二次加工：健康卡片/Nodes卡片/CPU卡片/内存卡片/NodeResourceUsage）
	ovForAnalysis, err := LoadOverviewForAnalysis()
	if err != nil {
		return nil, err
	}

	// 2) 物理机 30 分钟折线（1 分钟粒度；内部自动以 now 与 30m 上限）
	cpuSeries, memSeries, tempSeries := computeClusterUsageSeriesAll(ctx, clusterID, time.Time{}, 0)

	// 3) Alerts：一天窗口（总数/轻量表格/趋势）
	dayTotal, recentAlerts, alertTrends, err := BuildAlertsViewData(ctx, clusterID, 10)
	if err != nil {
		return nil, err
	}

	// 4) 组装统一返回体
	out := &AnalysisPayload{
		HealthCard:  ovForAnalysis.HealthCard,
		NodesCard:   ovForAnalysis.NodesCard,
		CPUCard:     ovForAnalysis.CPUCard,
		MemCard:     ovForAnalysis.MemCard,
		NodeUsages:  ovForAnalysis.Nodes,

		AlertsTotal:  dayTotal,
		RecentAlerts: recentAlerts,
		AlertTrends:  alertTrends,

		CPUSeries:  cpuSeries,
		MemSeries:  memSeries,
		TempSeries: tempSeries,
	}

	return out, nil
}
