package mock

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

// OTelSummaryRepository mock
type OTelSummaryRepository struct {
	GetAPMSummaryFn     func(ctx context.Context) (totalServices, healthyServices int, totalRPS, avgSuccessRate, avgP99Ms float64, err error)
	GetSLOSummaryFn     func(ctx context.Context) (ingressServices int, ingressAvgRPS float64, meshServices int, meshAvgMTLS float64, err error)
	GetMetricsSummaryFn func(ctx context.Context) (monitoredNodes int, avgCPUPct, avgMemPct, maxCPUPct, maxMemPct float64, err error)
}

func (m *OTelSummaryRepository) GetAPMSummary(ctx context.Context) (totalServices, healthyServices int, totalRPS, avgSuccessRate, avgP99Ms float64, err error) {
	if m.GetAPMSummaryFn != nil {
		return m.GetAPMSummaryFn(ctx)
	}
	return 0, 0, 0, 0, 0, nil
}

func (m *OTelSummaryRepository) GetSLOSummary(ctx context.Context) (ingressServices int, ingressAvgRPS float64, meshServices int, meshAvgMTLS float64, err error) {
	if m.GetSLOSummaryFn != nil {
		return m.GetSLOSummaryFn(ctx)
	}
	return 0, 0, 0, 0, nil
}

func (m *OTelSummaryRepository) GetMetricsSummary(ctx context.Context) (monitoredNodes int, avgCPUPct, avgMemPct, maxCPUPct, maxMemPct float64, err error) {
	if m.GetMetricsSummaryFn != nil {
		return m.GetMetricsSummaryFn(ctx)
	}
	return 0, 0, 0, 0, 0, nil
}

// TraceQueryRepository mock
type TraceQueryRepository struct {
	ListTracesFn    func(ctx context.Context, service string, minDurationMs float64, limit int, since time.Duration) ([]apm.TraceSummary, error)
	GetTraceDetailFn func(ctx context.Context, traceID string) (*apm.TraceDetail, error)
	ListServicesFn  func(ctx context.Context) ([]apm.APMService, error)
	GetTopologyFn   func(ctx context.Context) (*apm.Topology, error)
}

func (m *TraceQueryRepository) ListTraces(ctx context.Context, service string, minDurationMs float64, limit int, since time.Duration) ([]apm.TraceSummary, error) {
	if m.ListTracesFn != nil {
		return m.ListTracesFn(ctx, service, minDurationMs, limit, since)
	}
	return []apm.TraceSummary{}, nil
}

func (m *TraceQueryRepository) GetTraceDetail(ctx context.Context, traceID string) (*apm.TraceDetail, error) {
	if m.GetTraceDetailFn != nil {
		return m.GetTraceDetailFn(ctx, traceID)
	}
	return nil, nil
}

func (m *TraceQueryRepository) ListServices(ctx context.Context) ([]apm.APMService, error) {
	if m.ListServicesFn != nil {
		return m.ListServicesFn(ctx)
	}
	return []apm.APMService{}, nil
}

func (m *TraceQueryRepository) GetTopology(ctx context.Context) (*apm.Topology, error) {
	if m.GetTopologyFn != nil {
		return m.GetTopologyFn(ctx)
	}
	return nil, nil
}

// LogQueryRepository mock
type LogQueryRepository struct {
	QueryLogsFn func(ctx context.Context, opts repository.LogQueryOptions) (*log.QueryResult, error)
}

func (m *LogQueryRepository) QueryLogs(ctx context.Context, opts repository.LogQueryOptions) (*log.QueryResult, error) {
	if m.QueryLogsFn != nil {
		return m.QueryLogsFn(ctx, opts)
	}
	return &log.QueryResult{Logs: []log.Entry{}}, nil
}

// MetricsQueryRepository mock
type MetricsQueryRepository struct {
	ListAllNodeMetricsFn    func(ctx context.Context) ([]metrics.NodeMetrics, error)
	GetNodeMetricsFn        func(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error)
	GetNodeMetricsSeriesFn  func(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error)
	GetMetricsSummaryFn     func(ctx context.Context) (*metrics.Summary, error)
}

func (m *MetricsQueryRepository) ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error) {
	if m.ListAllNodeMetricsFn != nil {
		return m.ListAllNodeMetricsFn(ctx)
	}
	return []metrics.NodeMetrics{}, nil
}

func (m *MetricsQueryRepository) GetNodeMetrics(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error) {
	if m.GetNodeMetricsFn != nil {
		return m.GetNodeMetricsFn(ctx, nodeName)
	}
	return nil, nil
}

func (m *MetricsQueryRepository) GetNodeMetricsSeries(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error) {
	if m.GetNodeMetricsSeriesFn != nil {
		return m.GetNodeMetricsSeriesFn(ctx, nodeName, metric, since)
	}
	return []metrics.Point{}, nil
}

func (m *MetricsQueryRepository) GetMetricsSummary(ctx context.Context) (*metrics.Summary, error) {
	if m.GetMetricsSummaryFn != nil {
		return m.GetMetricsSummaryFn(ctx)
	}
	return nil, nil
}

// SLOQueryRepository mock
type SLOQueryRepository struct {
	ListIngressSLOFn    func(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
	ListServiceSLOFn    func(ctx context.Context, since time.Duration) ([]slo.ServiceSLO, error)
	ListServiceEdgesFn  func(ctx context.Context, since time.Duration) ([]slo.ServiceEdge, error)
	GetSLOTimeSeriesFn  func(ctx context.Context, name string, since time.Duration) (*slo.TimeSeries, error)
	GetSLOSummaryFn     func(ctx context.Context) (*slo.SLOSummary, error)
}

func (m *SLOQueryRepository) ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	if m.ListIngressSLOFn != nil {
		return m.ListIngressSLOFn(ctx, since)
	}
	return []slo.IngressSLO{}, nil
}

func (m *SLOQueryRepository) ListServiceSLO(ctx context.Context, since time.Duration) ([]slo.ServiceSLO, error) {
	if m.ListServiceSLOFn != nil {
		return m.ListServiceSLOFn(ctx, since)
	}
	return []slo.ServiceSLO{}, nil
}

func (m *SLOQueryRepository) ListServiceEdges(ctx context.Context, since time.Duration) ([]slo.ServiceEdge, error) {
	if m.ListServiceEdgesFn != nil {
		return m.ListServiceEdgesFn(ctx, since)
	}
	return []slo.ServiceEdge{}, nil
}

func (m *SLOQueryRepository) GetSLOTimeSeries(ctx context.Context, name string, since time.Duration) (*slo.TimeSeries, error) {
	if m.GetSLOTimeSeriesFn != nil {
		return m.GetSLOTimeSeriesFn(ctx, name, since)
	}
	return nil, nil
}

func (m *SLOQueryRepository) GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error) {
	if m.GetSLOSummaryFn != nil {
		return m.GetSLOSummaryFn(ctx)
	}
	return nil, nil
}
