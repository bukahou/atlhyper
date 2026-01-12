package overview

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/model/dto"
)

// BuildOverview 聚合总入口
func BuildOverview(ctx context.Context, clusterID string) (*dto.OverviewDTO, error) {
	// 时间窗口
	until := time.Now().UTC()
	sinceTr := until.Add(-15 * time.Minute) // 资源趋势：15m
	since24h := until.Add(-24 * time.Hour)  // Alerts：24h 窗口

	// 拉取仅需的四种数据
	pods, _ := fetchPods(ctx, clusterID)
	nodes, _ := fetchNodes(ctx, clusterID)
	eventsAll, _ := fetchEvents(ctx, clusterID, 50000)              // 大 limit，覆盖 24h
	series, _ := fetchMetricsRange(ctx, clusterID, sinceTr, until)   // 资源趋势

	// Alerts 统一在 24h 窗口内处理
	events24 := filterEventsByWindow(eventsAll, since24h, until)

	// 一次遍历节点，同时得到：
	// 1) 节点使用率表 rows
	// 2) 总 CPU 使用率卡片 cpuCard
	// 3) 总内存使用率卡片 memCard
	rows, cpuCard, memCard := buildNodeUsagesAndTotals(nodes)

	dto := &dto.OverviewDTO{
		ClusterID: clusterID,
		Cards: dto.OverviewCardsDTO{
			ClusterHealth: buildClusterHealth(pods, nodes), // Pod/Node Ready 率
			NodeReady:     buildNodeReady(nodes),           // Nodes 卡片
			CPUUsage:      cpuCard,                         // 集群总 CPU 使用率（加权）
			MemUsage:      memCard,                         // 集群总内存使用率（加权）
			Events24h:     len(events24),                   // 24h 事件总数
		},
		Trends: dto.OverviewTrendsDTO{
			ResourceUsage: buildResourceUsageTrends(series, sinceTr, until),
			PeakStats:     buildPeakStats(series),
		},
		Alerts: dto.OverviewAlertsDTO{
			Totals: buildSeverityTotals(events24, since24h, until),
			Trend:  buildAlertHourlyFromEvents(events24, since24h, until),
			Recent: buildRecentAlerts(events24, 10),
		},
		Nodes: dto.OverviewNodeSection{
			Usage: rows, // 各节点 CPU/内存使用率
		},
	}

	return dto, nil
}
