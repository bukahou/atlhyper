package snapshot

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/config"
	"AtlHyper/atlhyper_agent_v2/repository/ch"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// 2026-09-11 ClickHouse 负载排查：Dashboard 刷新时 GetMetricsSummary 内部就是
// ListAllNodeMetrics，而 otel_collector 两处分别调用 ⇒ 每周期全节点扫描跑两遍
// （8 节点 × 17 条 SQL 白跑一次，占单 agent 查询量的 48%）。SLO 同形。
// 修法：概览从同一周期取回的列表派生，⛔ 不再单独打库。

type dedupCounters struct {
	listNodes, summaryNodes, listSLO5m, summarySLO int32
}

func newDedupService(c *dedupCounters, nodes []metrics.NodeMetrics, ing []slo.IngressSLO) *snapshotService {
	m := &mock.MetricsQueryRepository{
		ListAllNodeMetricsFn: func(ctx context.Context) ([]metrics.NodeMetrics, error) {
			atomic.AddInt32(&c.listNodes, 1)
			return nodes, nil
		},
		GetMetricsSummaryFn: func(ctx context.Context) (*metrics.Summary, error) {
			atomic.AddInt32(&c.summaryNodes, 1)
			return &metrics.Summary{TotalNodes: 99}, nil // 哨兵：若被采用说明走了旧路径
		},
	}
	s := &mock.SLOQueryRepository{
		ListIngressSLOFn: func(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
			if since == 5*time.Minute {
				atomic.AddInt32(&c.listSLO5m, 1)
			}
			return ing, nil
		},
		GetSLOSummaryFn: func(ctx context.Context) (*slo.SLOSummary, error) {
			atomic.AddInt32(&c.summarySLO, 1)
			return &slo.SLOSummary{TotalServices: 99}, nil
		},
	}
	return &snapshotService{
		dashboardRepo:   ch.NewDashboardRepository(m, &mock.TraceQueryRepository{}, s, &mock.LogQueryRepository{}, nil),
		sloWindowCaches: map[string]*sloWindowCache{},
	}
}

func TestGetOTelSnapshot_SummariesDerivedFromLists(t *testing.T) {
	nodes := []metrics.NodeMetrics{
		{CPU: metrics.NodeCPU{UsagePct: 10}, Memory: metrics.NodeMemory{UsagePct: 50}, Temperature: metrics.NodeTemperature{CPUTempC: 60}},
		{CPU: metrics.NodeCPU{UsagePct: 30}, Memory: metrics.NodeMemory{UsagePct: 70}, Temperature: metrics.NodeTemperature{CPUTempC: 70}},
	}
	ing := []slo.IngressSLO{
		{SuccessRate: 99.95, RPS: 10, P99Ms: 100},
		{SuccessRate: 98.0, RPS: 5, P99Ms: 300},
	}
	var c dedupCounters
	svc := newDedupService(&c, nodes, ing)

	snap := svc.getOTelSnapshot(context.Background())

	if got := atomic.LoadInt32(&c.listNodes); got != 1 {
		t.Errorf("ListAllNodeMetrics 应恰好调用 1 次（去重前是 2 次），实际 %d", got)
	}
	if got := atomic.LoadInt32(&c.summaryNodes); got != 0 {
		t.Errorf("GetMetricsSummary 不应再被 Dashboard 路径调用，实际 %d 次", got)
	}
	if got := atomic.LoadInt32(&c.listSLO5m); got != 1 {
		t.Errorf("ListIngressSLO(5min) 应恰好调用 1 次，实际 %d", got)
	}
	if got := atomic.LoadInt32(&c.summarySLO); got != 0 {
		t.Errorf("GetSLOSummary 不应再被 Dashboard 路径调用，实际 %d 次", got)
	}

	wantM := metrics.SummarizeNodes(nodes)
	if snap.MetricsSummary == nil || *snap.MetricsSummary != *wantM {
		t.Errorf("MetricsSummary 应从列表派生 %+v，得到 %+v", wantM, snap.MetricsSummary)
	}
	wantS := slo.SummarizeIngress(ing)
	if snap.SLOSummary == nil || *snap.SLOSummary != *wantS {
		t.Errorf("SLOSummary 应从列表派生 %+v，得到 %+v", wantS, snap.SLOSummary)
	}
	if len(snap.MetricsNodes) != 2 || len(snap.SLOIngress) != 2 {
		t.Errorf("列表本身应照常填充，得到 nodes=%d slo=%d", len(snap.MetricsNodes), len(snap.SLOIngress))
	}
}

func TestGetOTelSnapshot_EmptyListsStillYieldSummaries(t *testing.T) {
	var c dedupCounters
	svc := newDedupService(&c, nil, nil)
	snap := svc.getOTelSnapshot(context.Background())
	if snap.MetricsSummary == nil || snap.MetricsSummary.TotalNodes != 0 {
		t.Errorf("空列表应得到全零概览而不是 nil，得到 %+v", snap.MetricsSummary)
	}
	if snap.SLOSummary == nil || snap.SLOSummary.TotalServices != 0 {
		t.Errorf("空列表应得到全零 SLO 概览而不是 nil，得到 %+v", snap.SLOSummary)
	}
}

// D3：dashboardTTL 从硬编码 30s 改为 AGENT_OTEL_DASHBOARD_TTL 可配（dev 可放宽到分钟级）。
func TestGetOTelSnapshot_DashboardTTLFromConfig(t *testing.T) {
	saved := config.GlobalConfig.Scheduler.OTelDashboardTTL
	defer func() { config.GlobalConfig.Scheduler.OTelDashboardTTL = saved }()

	config.GlobalConfig.Scheduler.OTelDashboardTTL = time.Hour
	var c dedupCounters
	svc := newDedupService(&c, nil, nil)
	svc.getOTelSnapshot(context.Background())
	svc.getOTelSnapshot(context.Background())
	if got := atomic.LoadInt32(&c.listNodes); got != 1 {
		t.Errorf("TTL=1h 时第二次刷新应命中缓存，ListAllNodeMetrics 实际 %d 次", got)
	}

	// 未配置（<=0）时回退 30s，行为与旧硬编码一致
	config.GlobalConfig.Scheduler.OTelDashboardTTL = 0
	if got := dashboardTTLOrDefault(); got != 30*time.Second {
		t.Errorf("未配置时应回退 30s，得到 %v", got)
	}
}
