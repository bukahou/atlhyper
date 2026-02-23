// Package query ClickHouse 按需查询仓库实现
package query

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/apm"
)

// traceRepository Trace 查询仓库
type traceRepository struct {
	client sdk.ClickHouseClient
}

// NewTraceQueryRepository 创建 Trace 查询仓库
func NewTraceQueryRepository(client sdk.ClickHouseClient) repository.TraceQueryRepository {
	return &traceRepository{client: client}
}

// ListTraces 查询 Trace 列表（按 TraceId 聚合）
func (r *traceRepository) ListTraces(ctx context.Context, service string, minDurationMs float64, limit int, since time.Duration) ([]apm.TraceSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 2000 {
		limit = 2000
	}

	sec := sinceSeconds(since)

	// 构建 WHERE
	conditions := []string{fmt.Sprintf("Timestamp >= now() - INTERVAL %d SECOND", sec)}
	var args []any

	if service != "" {
		conditions = append(conditions, "ServiceName = ?")
		args = append(args, service)
	}
	if minDurationMs > 0 {
		conditions = append(conditions, "Duration >= ?")
		args = append(args, int64(minDurationMs*1e6)) // ms → ns
	}

	where := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT TraceId,
		       min(Timestamp) AS ts,
		       argMinIf(ServiceName, Timestamp, ParentSpanId = '') AS rootSvc,
		       argMinIf(SpanName, Timestamp, ParentSpanId = '') AS rootOp,
		       max(Duration) / 1e6 AS durationMs,
		       count() AS spanCount,
		       count(DISTINCT ServiceName) AS serviceCount,
		       countIf(StatusCode = 'STATUS_CODE_ERROR') > 0 AS hasError
		FROM otel_traces
		WHERE %s
		GROUP BY TraceId
		ORDER BY ts DESC
		LIMIT %d
	`, where, limit)

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()

	var result []apm.TraceSummary
	for rows.Next() {
		var t apm.TraceSummary
		var hasErr uint8
		if err := rows.Scan(
			&t.TraceId, &t.Timestamp, &t.RootService, &t.RootOperation,
			&t.DurationMs, &t.SpanCount, &t.ServiceCount, &hasErr,
		); err != nil {
			continue
		}
		t.HasError = hasErr > 0
		t.DurationMs = roundTo(t.DurationMs, 2)
		result = append(result, t)
	}
	if result == nil {
		result = []apm.TraceSummary{}
	}
	return result, rows.Err()
}

// GetTraceDetail 查询 Trace 详情（所有 Span）
func (r *traceRepository) GetTraceDetail(ctx context.Context, traceID string) (*apm.TraceDetail, error) {
	if traceID == "" {
		return nil, fmt.Errorf("traceID is required")
	}

	query := `
		SELECT Timestamp, TraceId, SpanId, ParentSpanId, SpanName, SpanKind,
		       ServiceName, Duration, StatusCode, StatusMessage,
		       SpanAttributes, ResourceAttributes, Events.Timestamp, Events.Name, Events.Attributes
		FROM otel_traces
		WHERE TraceId = ?
		ORDER BY Timestamp
	`

	rows, err := r.client.Query(ctx, query, traceID)
	if err != nil {
		return nil, fmt.Errorf("get trace detail: %w", err)
	}
	defer rows.Close()

	var spans []apm.Span
	serviceSet := make(map[string]bool)

	for rows.Next() {
		var s apm.Span
		var spanAttrs map[string]string
		var resAttrs map[string]string
		var eventTimestamps []time.Time
		var eventNames []string
		var eventAttrs []map[string]string

		if err := rows.Scan(
			&s.Timestamp, &s.TraceId, &s.SpanId, &s.ParentSpanId,
			&s.SpanName, &s.SpanKind, &s.ServiceName, &s.Duration,
			&s.StatusCode, &s.StatusMessage,
			&spanAttrs, &resAttrs,
			&eventTimestamps, &eventNames, &eventAttrs,
		); err != nil {
			continue
		}

		s.DurationMs = parseDurationNanos(s.Duration)
		serviceSet[s.ServiceName] = true

		// 解析 HTTP 属性
		if method := spanAttrs["http.method"]; method != "" {
			s.HTTP = &apm.SpanHTTP{
				Method: method,
				Route:  spanAttrs["http.route"],
				URL:    spanAttrs["http.url"],
				Server: spanAttrs["server.address"],
			}
			if code := spanAttrs["http.status_code"]; code != "" {
				fmt.Sscanf(code, "%d", &s.HTTP.StatusCode)
			}
			if port := spanAttrs["server.port"]; port != "" {
				fmt.Sscanf(port, "%d", &s.HTTP.ServerPort)
			}
		}

		// 解析 DB 属性
		if sys := spanAttrs["db.system"]; sys != "" {
			s.DB = &apm.SpanDB{
				System:    sys,
				Name:      spanAttrs["db.name"],
				Operation: spanAttrs["db.operation"],
				Table:     spanAttrs["db.sql.table"],
				Statement: spanAttrs["db.statement"],
			}
		}

		// 解析 Resource 属性
		s.Resource = apm.SpanResource{
			ServiceVersion: resAttrs["service.version"],
			InstanceId:     resAttrs["service.instance.id"],
			PodName:        resAttrs["k8s.pod.name"],
			ClusterName:    resAttrs["k8s.cluster.name"],
		}

		// 解析 Events
		for i := 0; i < len(eventNames) && i < len(eventTimestamps); i++ {
			evt := apm.SpanEvent{
				Timestamp: eventTimestamps[i],
				Name:      eventNames[i],
			}
			if i < len(eventAttrs) {
				evt.Attributes = eventAttrs[i]
			}
			s.Events = append(s.Events, evt)
		}

		spans = append(spans, s)
	}

	if len(spans) == 0 {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	// 计算整体 duration
	var maxDuration float64
	for _, s := range spans {
		if s.DurationMs > maxDuration {
			maxDuration = s.DurationMs
		}
	}

	return &apm.TraceDetail{
		TraceId:      traceID,
		DurationMs:   roundTo(maxDuration, 2),
		ServiceCount: len(serviceSet),
		SpanCount:    len(spans),
		Spans:        spans,
	}, rows.Err()
}

// ListServices 查询服务列表（聚合统计）
func (r *traceRepository) ListServices(ctx context.Context) ([]apm.APMService, error) {
	query := `
		SELECT ServiceName,
		       ResourceAttributes['service.namespace'] AS ns,
		       count() AS spanCount,
		       countIf(StatusCode = 'STATUS_CODE_ERROR') AS errorCount,
		       1 - countIf(StatusCode = 'STATUS_CODE_ERROR') / count() AS successRate,
		       avg(Duration) / 1e6 AS avgMs,
		       quantile(0.50)(Duration) / 1e6 AS p50Ms,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms,
		       count() / 900 AS rps
		FROM otel_traces
		WHERE SpanKind = 'SPAN_KIND_SERVER'
		  AND Timestamp >= now() - INTERVAL 15 MINUTE
		GROUP BY ServiceName, ns
	`

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	var result []apm.APMService
	for rows.Next() {
		var s apm.APMService
		if err := rows.Scan(
			&s.Name, &s.Namespace, &s.SpanCount, &s.ErrorCount,
			&s.SuccessRate, &s.AvgDurationMs, &s.P50Ms, &s.P99Ms, &s.RPS,
		); err != nil {
			continue
		}
		if s.Namespace == "" {
			s.Namespace = "default"
		}
		s.SuccessRate = roundTo(s.SuccessRate, 4) // 保持 0-1 比例，前端负责 ×100 显示
		s.AvgDurationMs = roundTo(s.AvgDurationMs, 2)
		s.P50Ms = roundTo(s.P50Ms, 2)
		s.P99Ms = roundTo(s.P99Ms, 2)
		s.RPS = roundTo(s.RPS, 3)
		result = append(result, s)
	}
	if result == nil {
		result = []apm.APMService{}
	}
	return result, rows.Err()
}

// ListOperations 查询操作级聚合统计（GROUP BY ServiceName, SpanName）
func (r *traceRepository) ListOperations(ctx context.Context) ([]apm.OperationStats, error) {
	query := `
		SELECT ServiceName,
		       SpanName AS operation,
		       count() AS spanCount,
		       countIf(StatusCode = 'STATUS_CODE_ERROR') AS errorCount,
		       1 - countIf(StatusCode = 'STATUS_CODE_ERROR') / count() AS successRate,
		       avg(Duration) / 1e6 AS avgMs,
		       quantile(0.50)(Duration) / 1e6 AS p50Ms,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms,
		       count() / 900 AS rps
		FROM otel_traces
		WHERE SpanKind = 'SPAN_KIND_SERVER'
		  AND Timestamp >= now() - INTERVAL 15 MINUTE
		GROUP BY ServiceName, SpanName
		ORDER BY spanCount DESC
	`

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list operations: %w", err)
	}
	defer rows.Close()

	var result []apm.OperationStats
	for rows.Next() {
		var o apm.OperationStats
		if err := rows.Scan(
			&o.ServiceName, &o.OperationName, &o.SpanCount, &o.ErrorCount,
			&o.SuccessRate, &o.AvgDurationMs, &o.P50Ms, &o.P99Ms, &o.RPS,
		); err != nil {
			continue
		}
		o.SuccessRate = roundTo(o.SuccessRate, 4)
		o.AvgDurationMs = roundTo(o.AvgDurationMs, 2)
		o.P50Ms = roundTo(o.P50Ms, 2)
		o.P99Ms = roundTo(o.P99Ms, 2)
		o.RPS = roundTo(o.RPS, 3)
		result = append(result, o)
	}
	if result == nil {
		result = []apm.OperationStats{}
	}
	return result, rows.Err()
}

// GetTopology 获取服务拓扑图
func (r *traceRepository) GetTopology(ctx context.Context) (*apm.Topology, error) {
	// 节点: 所有服务
	serviceQuery := `
		SELECT ServiceName,
		       ResourceAttributes['service.namespace'] AS ns,
		       count() / 900 AS rps,
		       1 - countIf(StatusCode = 'STATUS_CODE_ERROR') / count() AS successRate,
		       quantile(0.99)(Duration) / 1e6 AS p99Ms
		FROM otel_traces
		WHERE SpanKind = 'SPAN_KIND_SERVER'
		  AND Timestamp >= now() - INTERVAL 15 MINUTE
		GROUP BY ServiceName, ns
	`
	svcRows, err := r.client.Query(ctx, serviceQuery)
	if err != nil {
		return nil, fmt.Errorf("topology nodes: %w", err)
	}
	defer svcRows.Close()

	var nodes []apm.TopologyNode
	nodeSet := make(map[string]bool)

	for svcRows.Next() {
		var name, ns string
		var rps, successRate, p99Ms float64
		if err := svcRows.Scan(&name, &ns, &rps, &successRate, &p99Ms); err != nil {
			continue
		}
		if ns == "" {
			ns = "default"
		}
		id := ns + "/" + name
		nodeSet[id] = true
		node := apm.TopologyNode{
			Id:          id,
			Name:        name,
			Namespace:   ns,
			Type:        "service",
			RPS:         roundTo(rps, 3),
			SuccessRate: roundTo(successRate, 4), // 保持 0-1 比例
			P99Ms:       roundTo(p99Ms, 2),
		}
		nodes = append(nodes, node)
	}

	// 边: 跨服务调用（parent-child 关系）
	// source/target 使用 ns/name 格式与节点 ID 保持一致
	edgeQuery := `
		SELECT concat(if(t1.ResourceAttributes['service.namespace'] = '', 'default', t1.ResourceAttributes['service.namespace']), '/', t1.ServiceName) AS source,
		       concat(if(t2.ResourceAttributes['service.namespace'] = '', 'default', t2.ResourceAttributes['service.namespace']), '/', t2.ServiceName) AS target,
		       count() AS callCount,
		       avg(t2.Duration) / 1e6 AS avgMs,
		       countIf(t2.StatusCode = 'STATUS_CODE_ERROR') / count() AS errorRate
		FROM otel_traces t1
		JOIN otel_traces t2 ON t1.SpanId = t2.ParentSpanId AND t1.TraceId = t2.TraceId
		WHERE t1.ServiceName != t2.ServiceName
		  AND t1.Timestamp >= now() - INTERVAL 15 MINUTE
		  AND t2.Timestamp >= now() - INTERVAL 15 MINUTE
		GROUP BY source, target
	`

	edgeRows, err := r.client.Query(ctx, edgeQuery)
	if err != nil {
		return &apm.Topology{Nodes: nodes, Edges: []apm.TopologyEdge{}}, nil
	}
	defer edgeRows.Close()

	var edges []apm.TopologyEdge
	for edgeRows.Next() {
		var e apm.TopologyEdge
		var source, target string
		if err := edgeRows.Scan(&source, &target, &e.CallCount, &e.AvgMs, &e.ErrorRate); err != nil {
			continue
		}
		// 只保留两端节点都存在的边
		if !nodeSet[source] || !nodeSet[target] {
			continue
		}
		e.Source = source
		e.Target = target
		e.AvgMs = roundTo(e.AvgMs, 2)
		e.ErrorRate = roundTo(e.ErrorRate, 4) // 保持 0-1 比例
		edges = append(edges, e)
	}
	if edges == nil {
		edges = []apm.TopologyEdge{}
	}

	// 排序节点
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	return &apm.Topology{Nodes: nodes, Edges: edges}, nil
}
