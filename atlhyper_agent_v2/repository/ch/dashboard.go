// Package ch ClickHouse 仓库实现
package ch

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// dashboardRepository Dashboard 数据采集（组合委托现有 repos）
type dashboardRepository struct {
	metrics repository.MetricsQueryRepository
	trace   repository.TraceQueryRepository
	slo     repository.SLOQueryRepository
}

// NewDashboardRepository 创建 Dashboard 仓库
func NewDashboardRepository(
	m repository.MetricsQueryRepository,
	t repository.TraceQueryRepository,
	s repository.SLOQueryRepository,
) repository.OTelDashboardRepository {
	return &dashboardRepository{metrics: m, trace: t, slo: s}
}

func (r *dashboardRepository) GetMetricsSummary(ctx context.Context) (*metrics.Summary, error) {
	return r.metrics.GetMetricsSummary(ctx)
}

func (r *dashboardRepository) ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error) {
	return r.metrics.ListAllNodeMetrics(ctx)
}

func (r *dashboardRepository) ListAPMServices(ctx context.Context) ([]apm.APMService, error) {
	return r.trace.ListServices(ctx)
}

func (r *dashboardRepository) GetAPMTopology(ctx context.Context) (*apm.Topology, error) {
	return r.trace.GetTopology(ctx)
}

func (r *dashboardRepository) GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error) {
	return r.slo.GetSLOSummary(ctx)
}

func (r *dashboardRepository) ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	return r.slo.ListIngressSLO(ctx, since)
}

func (r *dashboardRepository) ListServiceSLO(ctx context.Context, since time.Duration) ([]slo.ServiceSLO, error) {
	return r.slo.ListServiceSLO(ctx, since)
}

func (r *dashboardRepository) ListServiceEdges(ctx context.Context, since time.Duration) ([]slo.ServiceEdge, error) {
	return r.slo.ListServiceEdges(ctx, since)
}
