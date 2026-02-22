package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/log"
)

// logRepository Log 查询仓库
type logRepository struct {
	client sdk.ClickHouseClient
}

// NewLogQueryRepository 创建 Log 查询仓库
func NewLogQueryRepository(client sdk.ClickHouseClient) repository.LogQueryRepository {
	return &logRepository{client: client}
}

// QueryLogs 查询日志（主查询 + 总数 + Facets）
func (r *logRepository) QueryLogs(ctx context.Context, opts repository.LogQueryOptions) (*log.QueryResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 500 {
		opts.Limit = 500
	}

	since := sinceSeconds(opts.Since)

	// 构建 WHERE 条件
	where, args := r.buildWhere(since, opts)

	// 并行执行 3 类查询
	type logsResult struct {
		entries []log.Entry
		err     error
	}
	type countResult struct {
		total int64
		err   error
	}
	type facetsResult struct {
		facets log.Facets
		err    error
	}

	logsCh := make(chan logsResult, 1)
	countCh := make(chan countResult, 1)
	facetsCh := make(chan facetsResult, 1)

	// 主查询
	go func() {
		entries, err := r.queryEntries(ctx, where, args, opts.Limit, opts.Offset)
		logsCh <- logsResult{entries, err}
	}()

	// 总数
	go func() {
		total, err := r.queryCount(ctx, where, args)
		countCh <- countResult{total, err}
	}()

	// Facets（仅基于时间范围，不受搜索条件限制）
	go func() {
		facets, err := r.queryFacets(ctx, since)
		facetsCh <- facetsResult{facets, err}
	}()

	lr := <-logsCh
	cr := <-countCh
	fr := <-facetsCh

	if lr.err != nil {
		return nil, fmt.Errorf("query logs: %w", lr.err)
	}
	if cr.err != nil {
		return nil, fmt.Errorf("query count: %w", cr.err)
	}

	result := &log.QueryResult{
		Logs:  lr.entries,
		Total: cr.total,
	}
	if fr.err == nil {
		result.Facets = fr.facets
	}

	return result, nil
}

// buildWhere 构建 WHERE 子句和参数
func (r *logRepository) buildWhere(since int64, opts repository.LogQueryOptions) (string, []any) {
	conditions := []string{fmt.Sprintf("Timestamp >= now() - INTERVAL %d SECOND", since)}
	var args []any

	if opts.Query != "" {
		conditions = append(conditions, "Body LIKE ?")
		args = append(args, "%"+opts.Query+"%")
	}
	if opts.Service != "" {
		conditions = append(conditions, "ServiceName = ?")
		args = append(args, opts.Service)
	}
	if opts.Level != "" {
		conditions = append(conditions, "SeverityText = ?")
		args = append(args, opts.Level)
	}
	if opts.Scope != "" {
		conditions = append(conditions, "ScopeName = ?")
		args = append(args, opts.Scope)
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// queryEntries 查询日志条目
func (r *logRepository) queryEntries(ctx context.Context, where string, args []any, limit, offset int) ([]log.Entry, error) {
	query := fmt.Sprintf(`
		SELECT Timestamp, TraceId, SpanId, SeverityText, SeverityNumber,
		       ServiceName, Body, ScopeName,
		       LogAttributes, ResourceAttributes
		FROM otel_logs
		%s
		ORDER BY Timestamp DESC
		LIMIT %d OFFSET %d
	`, where, limit, offset)

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []log.Entry
	for rows.Next() {
		var e log.Entry
		var attrs, resource map[string]string
		if err := rows.Scan(
			&e.Timestamp, &e.TraceId, &e.SpanId,
			&e.Severity, &e.SeverityNum,
			&e.ServiceName, &e.Body, &e.ScopeName,
			&attrs, &resource,
		); err != nil {
			return nil, fmt.Errorf("scan log entry: %w", err)
		}
		e.Attributes = attrs
		e.Resource = resource
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []log.Entry{}
	}
	return entries, rows.Err()
}

// queryCount 查询总数
func (r *logRepository) queryCount(ctx context.Context, where string, args []any) (int64, error) {
	query := fmt.Sprintf("SELECT count() FROM otel_logs %s", where)
	var total int64
	err := r.client.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// queryFacets 查询分面统计（仅基于时间范围）
func (r *logRepository) queryFacets(ctx context.Context, since int64) (log.Facets, error) {
	timeWhere := fmt.Sprintf("WHERE Timestamp >= now() - INTERVAL %d SECOND", since)

	var facets log.Facets

	// Services
	services, err := r.queryFacet(ctx, "ServiceName", timeWhere)
	if err != nil {
		return facets, err
	}
	facets.Services = services

	// Severities
	severities, err := r.queryFacet(ctx, "SeverityText", timeWhere)
	if err != nil {
		return facets, err
	}
	facets.Severities = severities

	// Scopes
	scopes, err := r.queryFacet(ctx, "ScopeName", timeWhere)
	if err != nil {
		return facets, err
	}
	facets.Scopes = scopes

	return facets, nil
}

// queryFacet 查询单个分面
func (r *logRepository) queryFacet(ctx context.Context, column, where string) ([]log.Facet, error) {
	query := fmt.Sprintf(`
		SELECT %s AS value, count() AS cnt
		FROM otel_logs %s
		GROUP BY value
		ORDER BY cnt DESC
		LIMIT 50
	`, column, where)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facets []log.Facet
	for rows.Next() {
		var f log.Facet
		if err := rows.Scan(&f.Value, &f.Count); err != nil {
			return nil, err
		}
		facets = append(facets, f)
	}
	if facets == nil {
		facets = []log.Facet{}
	}
	return facets, rows.Err()
}

// scanFacets 从 rows 扫描 facet 列表（复用）
func scanFacets(rows *sql.Rows) ([]log.Facet, error) {
	var facets []log.Facet
	for rows.Next() {
		var f log.Facet
		if err := rows.Scan(&f.Value, &f.Count); err != nil {
			return nil, err
		}
		facets = append(facets, f)
	}
	if facets == nil {
		facets = []log.Facet{}
	}
	return facets, rows.Err()
}
