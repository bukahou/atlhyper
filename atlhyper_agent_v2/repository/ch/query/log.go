package query

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	where, args, err := r.buildWhere(since, opts)
	if err != nil {
		return nil, err
	}

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
// 起止时间解析失败时返回错误：不能退化成没有时间条件的全表扫描
func (r *logRepository) buildWhere(since int64, opts repository.LogQueryOptions) (string, []any, error) {
	var conditions []string
	var args []any

	// 绝对时间范围（brush 选区）优先于相对时间
	// 前端传 ISO 8601 字符串，需解析为 time.Time 以兼容 ClickHouse DateTime64(9)
	if opts.StartTime != "" && opts.EndTime != "" {
		startT, err := time.Parse(time.RFC3339Nano, opts.StartTime)
		if err != nil {
			return "", nil, fmt.Errorf("start_time 不是 RFC 3339 时间: %w", err)
		}
		endT, err := time.Parse(time.RFC3339Nano, opts.EndTime)
		if err != nil {
			return "", nil, fmt.Errorf("end_time 不是 RFC 3339 时间: %w", err)
		}
		conditions = append(conditions, "Timestamp >= ?", "Timestamp <= ?")
		args = append(args, startT, endT)
	} else if opts.TraceId == "" {
		// 有 TraceId 时跳过时间条件（TraceId 是精确匹配，不需要时间窗口限制）
		conditions = append(conditions, fmt.Sprintf("Timestamp >= now() - INTERVAL %d SECOND", since))
	}

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
	if opts.TraceId != "" {
		conditions = append(conditions, "TraceId = ?")
		args = append(args, opts.TraceId)
	}
	if opts.SpanId != "" {
		conditions = append(conditions, "SpanId = ?")
		args = append(args, opts.SpanId)
	}

	if len(conditions) == 0 {
		return "", args, nil
	}
	return "WHERE " + strings.Join(conditions, " AND "), args, nil
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

// QueryHistogram 查询直方图（服务端聚合，~30 桶）
func (r *logRepository) QueryHistogram(ctx context.Context, opts repository.LogQueryOptions) (*log.HistogramResult, error) {
	since := sinceSeconds(opts.Since)
	where, args, err := r.buildWhere(since, opts)
	if err != nil {
		return nil, err
	}

	// 绝对时间时基于实际跨度计算桶间隔
	spanSeconds := since
	if opts.StartTime != "" && opts.EndTime != "" {
		if startT, err := time.Parse(time.RFC3339Nano, opts.StartTime); err == nil {
			if endT, err := time.Parse(time.RFC3339Nano, opts.EndTime); err == nil {
				s := int64(endT.Sub(startT).Seconds())
				if s > 0 {
					spanSeconds = s
				}
			}
		}
	}
	interval := histogramBucketSeconds(spanSeconds)

	query := fmt.Sprintf(`
		SELECT
			toStartOfInterval(Timestamp, INTERVAL %d SECOND) AS bucket,
			SeverityText AS severity,
			count() AS cnt
		FROM otel_logs
		%s
		GROUP BY bucket, severity
		ORDER BY bucket ASC
	`, interval, where)

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query histogram: %w", err)
	}
	defer rows.Close()

	var buckets []log.HistogramBucket
	for rows.Next() {
		var b log.HistogramBucket
		if err := rows.Scan(&b.Timestamp, &b.Severity, &b.Count); err != nil {
			return nil, fmt.Errorf("scan histogram bucket: %w", err)
		}
		buckets = append(buckets, b)
	}
	if buckets == nil {
		buckets = []log.HistogramBucket{}
	}

	return &log.HistogramResult{
		Buckets:    buckets,
		IntervalMs: interval * 1000,
	}, rows.Err()
}

// histogramBucketSeconds 自适应桶间隔：目标 ~30 桶，最小 10 秒
func histogramBucketSeconds(sinceSeconds int64) int64 {
	interval := sinceSeconds / 30
	if interval < 10 {
		interval = 10
	}
	return interval
}

// 日志概要的统计口径：时间窗与「什么算 ERROR」。总的 ERROR 计数（SeverityCounts["ERROR"]）与按服务的
// 错误数都由这两段拼出，同一批日志在两处的数才对得上
const (
	logSummaryWindow  = "Timestamp >= now() - INTERVAL 5 MINUTE"
	logErrorCondition = "SeverityText IN ('ERROR', 'SEVERE')"
)

// logSummaryAggQuery 日志概要的聚合：总数 + 各级别计数 + 最新时间
const logSummaryAggQuery = `
		SELECT count() AS total,
		       countIf(` + logErrorCondition + `) AS errors,
		       countIf(SeverityText IN ('WARN', 'WARNING')) AS warns,
		       countIf(SeverityText = 'INFO') AS infos,
		       countIf(SeverityText = 'DEBUG') AS debugs,
		       max(Timestamp) AS latest_at
		FROM otel_logs
		WHERE ` + logSummaryWindow + `
	`

// logTopServicesQuery 日志量最多的 10 个服务
const logTopServicesQuery = `
		SELECT ServiceName, count() AS cnt
		FROM otel_logs
		WHERE ` + logSummaryWindow + `
		GROUP BY ServiceName
		ORDER BY cnt DESC
		LIMIT 10
	`

// maxServiceErrors 日志概要里按服务的错误数最多带几个服务（错误数最多的前几名），控制快照大小
const maxServiceErrors = 50

// serviceErrorsQuery 按服务的 ERROR 日志数：错误数最多的前 maxServiceErrors 个服务，同数时按名字排（截断处稳定）。
// 命名空间与 APM 服务同一取法，主控据此拼服务实体键；没有服务名的日志归不到任何服务，不计入
var serviceErrorsQuery = `
		SELECT ServiceName, ` + serviceNamespaceExpr("") + ` AS ns, count() AS errors
		FROM otel_logs
		WHERE ` + logSummaryWindow + ` AND ` + logErrorCondition + ` AND ServiceName != ''
		GROUP BY ServiceName, ns
		ORDER BY errors DESC, ServiceName, ns
		LIMIT ` + strconv.Itoa(maxServiceErrors) + `
	`

// GetSummary 获取日志统计摘要（5 分钟窗口）
func (r *logRepository) GetSummary(ctx context.Context) (*log.Summary, error) {
	summary := &log.Summary{
		SeverityCounts: make(map[string]int64),
	}

	// 聚合查询：总数 + 各级别计数 + 最新时间
	var total, errors, warns, infos, debugs int64
	var latestAt time.Time
	err := r.client.QueryRow(ctx, logSummaryAggQuery).Scan(&total, &errors, &warns, &infos, &debugs, &latestAt)
	if err != nil {
		return nil, fmt.Errorf("query log summary: %w", err)
	}

	summary.TotalEntries = total
	summary.LatestAt = latestAt
	if errors > 0 {
		summary.SeverityCounts["ERROR"] = errors
	}
	if warns > 0 {
		summary.SeverityCounts["WARN"] = warns
	}
	if infos > 0 {
		summary.SeverityCounts["INFO"] = infos
	}
	if debugs > 0 {
		summary.SeverityCounts["DEBUG"] = debugs
	}

	// 按服务的错误数（口径同上面的 ERROR 计数）
	serviceErrors, err := r.queryServiceErrors(ctx)
	if err != nil {
		return nil, fmt.Errorf("日志按服务错误数查询: %w", err)
	}
	summary.ServiceErrors = serviceErrors

	// Top 10 服务
	rows, err := r.client.Query(ctx, logTopServicesQuery)
	if err != nil {
		return nil, fmt.Errorf("日志 Top 服务查询: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sc log.ServiceCount
		if err := rows.Scan(&sc.Service, &sc.Count); err != nil {
			break
		}
		summary.TopServices = append(summary.TopServices, sc)
	}
	if summary.TopServices == nil {
		summary.TopServices = []log.ServiceCount{}
	}

	return summary, nil
}

// queryServiceErrors 按服务的 ERROR 日志数（serviceErrorsQuery）
func (r *logRepository) queryServiceErrors(ctx context.Context) ([]log.ServiceErrorCount, error) {
	rows, err := r.client.Query(ctx, serviceErrorsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skips := newScanSkips("queryServiceErrors")
	defer skips.report()
	return scanServiceErrors(rows, skips)
}

// scanServiceErrors 逐行读出按服务的错误数，解析不了的行跳过并记入 skips。
// 查询成功但没有行时返回空切片：空 = 没有服务出错，nil 留给「没有这项数据」（旧版 Agent）
func scanServiceErrors(rows *sql.Rows, skips *scanSkips) ([]log.ServiceErrorCount, error) {
	result := []log.ServiceErrorCount{}
	for rows.Next() {
		var se log.ServiceErrorCount
		if err := rows.Scan(&se.Service, &se.Namespace, &se.Count); err != nil {
			skips.add(err)
			continue
		}
		result = append(result, se)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
