package command

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/command"
)

// =============================================================================
// ClickHouse 查询指令处理
// =============================================================================

// parseQuery 解析并校验查询参数：类型不对、缺必填项、取值越界都直接报错，⛔ 不按零值继续
func parseQuery[P command.QueryParams](cmd *command.Command, into P) error {
	if err := command.ParseParams(cmd.Params, into); err != nil {
		return fmt.Errorf("%s 参数格式不对: %w", cmd.Action, err)
	}
	if err := into.Validate(); err != nil {
		return fmt.Errorf("%s 参数不合法: %w", cmd.Action, err)
	}
	return nil
}

// handleQueryTraces 处理 Trace 查询指令
func (s *commandService) handleQueryTraces(ctx context.Context, cmd *command.Command) (any, error) {
	if s.traceQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}
	var p command.QueryTracesParams
	if err := parseQuery(cmd, &p); err != nil {
		return nil, err
	}
	start, end := p.StartTime, p.EndTime

	switch p.Action() {
	case command.TraceListTraces:
		limit := p.Limit
		if limit == 0 {
			limit = 50
		}
		return s.traceQueryRepo.ListTraces(ctx, p.Service, p.Operation, p.MinDurationMs, limit,
			p.SinceOr(5*time.Minute), p.Sort, start, end, p.StatusCode, p.Method)

	case command.TraceListServices:
		return s.traceQueryRepo.ListServices(ctx, p.SinceOr(15*time.Minute), start, end)

	case command.TraceCorrelations:
		mode := p.Mode
		if mode == "" {
			mode = apm.CorrelationModeFailure
		}
		return s.traceQueryRepo.GetTraceCorrelations(ctx, p.Service, p.Operation, mode, p.SinceOr(60*time.Minute), start, end)

	case command.TraceTopology:
		return s.traceQueryRepo.GetTopology(ctx, p.SinceOr(15*time.Minute), start, end)

	case command.TraceListOperations:
		return s.traceQueryRepo.ListOperations(ctx, p.SinceOr(15*time.Minute), start, end)

	case command.TraceHTTPStats:
		return s.traceQueryRepo.GetHTTPStats(ctx, p.Service, p.SinceOr(15*time.Minute), start, end)

	case command.TraceDBStats:
		return s.traceQueryRepo.GetDBStats(ctx, p.Service, p.SinceOr(15*time.Minute), start, end)

	case command.TraceServiceSeries:
		points, err := s.traceQueryRepo.GetServiceTimeSeries(ctx, p.Service, p.SinceOr(60*time.Minute))
		if err != nil {
			return nil, err
		}
		// 包装为与 Concentrator 路径一致的格式: { service, points }
		return map[string]interface{}{
			"service": p.Service,
			"points":  points,
		}, nil
	}
	return nil, fmt.Errorf("unknown trace sub_action: %s", p.SubAction) // Validate 已拦下，走不到
}

// handleQueryTraceDetail 处理 Trace 详情查询指令
func (s *commandService) handleQueryTraceDetail(ctx context.Context, cmd *command.Command) (any, error) {
	if s.traceQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}
	var p command.QueryTraceDetailParams
	if err := parseQuery(cmd, &p); err != nil {
		return nil, err
	}
	return s.traceQueryRepo.GetTraceDetail(ctx, p.TraceID)
}

// handleQueryLogs 处理日志查询指令（通过 sub_action 区分列表/直方图）
func (s *commandService) handleQueryLogs(ctx context.Context, cmd *command.Command) (any, error) {
	if s.logQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}
	var p command.QueryLogsParams
	if err := parseQuery(cmd, &p); err != nil {
		return nil, err
	}
	opts := repository.LogQueryOptions{
		Query:     p.Query,
		Service:   p.Service,
		Level:     p.Level,
		Scope:     p.Scope,
		Since:     p.SinceOr(15 * time.Minute),
		StartTime: p.StartTime,
		EndTime:   p.EndTime,
	}
	if p.SubAction == command.LogsHistogram {
		return s.logQueryRepo.QueryHistogram(ctx, opts)
	}
	opts.TraceId = p.TraceID
	opts.SpanId = p.SpanID
	opts.Limit = p.Limit
	if opts.Limit == 0 {
		opts.Limit = 50
	}
	opts.Offset = p.Offset
	return s.logQueryRepo.QueryLogs(ctx, opts)
}

// handleQueryMetrics 处理指标查询指令
func (s *commandService) handleQueryMetrics(ctx context.Context, cmd *command.Command) (any, error) {
	if s.metricsQueryRepo == nil {
		return nil, fmt.Errorf("ClickHouse not configured")
	}
	var p command.QueryMetricsParams
	if err := parseQuery(cmd, &p); err != nil {
		return nil, err
	}

	switch p.Action() {
	case command.MetricsListAll:
		return s.metricsQueryRepo.ListAllNodeMetrics(ctx)
	case command.MetricsGetNode:
		return s.metricsQueryRepo.GetNodeMetrics(ctx, p.NodeName)
	case command.MetricsGetSeries:
		return s.metricsQueryRepo.GetNodeMetricsSeries(ctx, p.NodeName, p.Metric, p.SinceOr(30*time.Minute))
	case command.MetricsGetHistory:
		return s.metricsQueryRepo.GetNodeMetricsHistory(ctx, p.NodeName, p.SinceOr(24*time.Hour))
	case command.MetricsSummary:
		return s.metricsQueryRepo.GetMetricsSummary(ctx)
	}
	return nil, fmt.Errorf("unknown metrics sub_action: %s", p.SubAction) // Validate 已拦下，走不到
}
