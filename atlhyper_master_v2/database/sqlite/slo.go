// atlhyper_master_v2/database/sqlite/slo.go
// SQLite SLO Dialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloDialect struct{}

// ==================== Counter Snapshot ====================

func (d *sloDialect) SelectCounterSnapshot(clusterID, host string) (string, []any) {
	if host == "" {
		return `SELECT id, cluster_id, host, ingress_name, ingress_class, namespace, service, tls, method, status, counter_value, prev_value, updated_at
			FROM ingress_counter_snapshot WHERE cluster_id = ?`, []any{clusterID}
	}
	return `SELECT id, cluster_id, host, ingress_name, ingress_class, namespace, service, tls, method, status, counter_value, prev_value, updated_at
		FROM ingress_counter_snapshot WHERE cluster_id = ? AND host = ?`, []any{clusterID, host}
}

func (d *sloDialect) UpsertCounterSnapshot(s *database.IngressCounterSnapshot) (string, []any) {
	query := `INSERT INTO ingress_counter_snapshot (cluster_id, host, ingress_name, ingress_class, namespace, service, tls, method, status, counter_value, prev_value, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, host, method, status) DO UPDATE SET
		ingress_name = excluded.ingress_name,
		ingress_class = excluded.ingress_class,
		namespace = excluded.namespace,
		service = excluded.service,
		tls = excluded.tls,
		prev_value = ingress_counter_snapshot.counter_value,
		counter_value = excluded.counter_value,
		updated_at = excluded.updated_at`
	args := []any{
		s.ClusterID, s.Host, s.IngressName, s.IngressClass, s.Namespace, s.Service,
		boolToInt(s.TLS), s.Method, s.Status, s.CounterValue, s.PrevValue,
		s.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) ScanCounterSnapshot(rows *sql.Rows) (*database.IngressCounterSnapshot, error) {
	s := &database.IngressCounterSnapshot{}
	var tls int
	var updatedAt string
	err := rows.Scan(&s.ID, &s.ClusterID, &s.Host, &s.IngressName, &s.IngressClass,
		&s.Namespace, &s.Service, &tls, &s.Method, &s.Status,
		&s.CounterValue, &s.PrevValue, &updatedAt)
	if err != nil {
		return nil, err
	}
	s.TLS = tls == 1
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return s, nil
}

// ==================== Histogram Snapshot ====================

func (d *sloDialect) SelectHistogramSnapshot(clusterID, host string) (string, []any) {
	if host == "" {
		return `SELECT id, cluster_id, host, ingress_name, namespace, le, bucket_value, prev_value, sum_value, count_value, updated_at
			FROM ingress_histogram_snapshot WHERE cluster_id = ?`, []any{clusterID}
	}
	return `SELECT id, cluster_id, host, ingress_name, namespace, le, bucket_value, prev_value, sum_value, count_value, updated_at
		FROM ingress_histogram_snapshot WHERE cluster_id = ? AND host = ?`, []any{clusterID, host}
}

func (d *sloDialect) UpsertHistogramSnapshot(s *database.IngressHistogramSnapshot) (string, []any) {
	query := `INSERT INTO ingress_histogram_snapshot (cluster_id, host, ingress_name, namespace, le, bucket_value, prev_value, sum_value, count_value, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, host, le) DO UPDATE SET
		ingress_name = excluded.ingress_name,
		namespace = excluded.namespace,
		prev_value = ingress_histogram_snapshot.bucket_value,
		bucket_value = excluded.bucket_value,
		sum_value = excluded.sum_value,
		count_value = excluded.count_value,
		updated_at = excluded.updated_at`
	args := []any{
		s.ClusterID, s.Host, s.IngressName, s.Namespace, s.LE,
		s.BucketValue, s.PrevValue, s.SumValue, s.CountValue,
		s.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) ScanHistogramSnapshot(rows *sql.Rows) (*database.IngressHistogramSnapshot, error) {
	s := &database.IngressHistogramSnapshot{}
	var updatedAt string
	var sumValue sql.NullFloat64
	var countValueInt sql.NullInt64
	err := rows.Scan(&s.ID, &s.ClusterID, &s.Host, &s.IngressName, &s.Namespace,
		&s.LE, &s.BucketValue, &s.PrevValue, &sumValue, &countValueInt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if sumValue.Valid {
		s.SumValue = &sumValue.Float64
	}
	if countValueInt.Valid {
		cv := countValueInt.Int64
		s.CountValue = &cv
	}
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return s, nil
}

// ==================== Raw Metrics ====================

func (d *sloDialect) InsertRawMetrics(m *database.SLOMetricsRaw) (string, []any) {
	query := `INSERT INTO slo_metrics_raw (cluster_id, host, domain, path_prefix, timestamp, total_requests, error_requests, sum_latency_ms,
		bucket_5ms, bucket_10ms, bucket_25ms, bucket_50ms, bucket_100ms, bucket_250ms, bucket_500ms,
		bucket_1s, bucket_2500ms, bucket_5s, bucket_10s, bucket_inf, is_missing)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		m.ClusterID, m.Host, m.Domain, m.PathPrefix, m.Timestamp.Format(time.RFC3339),
		m.TotalRequests, m.ErrorRequests, m.SumLatencyMs,
		m.Bucket5ms, m.Bucket10ms, m.Bucket25ms, m.Bucket50ms,
		m.Bucket100ms, m.Bucket250ms, m.Bucket500ms, m.Bucket1s,
		m.Bucket2500ms, m.Bucket5s, m.Bucket10s, m.BucketInf,
		boolToInt(m.IsMissing),
	}
	return query, args
}

func (d *sloDialect) SelectRawMetrics(clusterID, host string, start, end time.Time) (string, []any) {
	return `SELECT id, cluster_id, host, domain, path_prefix, timestamp, total_requests, error_requests, sum_latency_ms,
		bucket_5ms, bucket_10ms, bucket_25ms, bucket_50ms, bucket_100ms, bucket_250ms, bucket_500ms,
		bucket_1s, bucket_2500ms, bucket_5s, bucket_10s, bucket_inf, is_missing
		FROM slo_metrics_raw
		WHERE cluster_id = ? AND host = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp ASC`,
		[]any{clusterID, host, start.Format(time.RFC3339), end.Format(time.RFC3339)}
}

func (d *sloDialect) UpdateRawMetricsBuckets(m *database.SLOMetricsRaw) (string, []any) {
	query := `UPDATE slo_metrics_raw SET
		bucket_5ms = ?, bucket_10ms = ?, bucket_25ms = ?, bucket_50ms = ?,
		bucket_100ms = ?, bucket_250ms = ?, bucket_500ms = ?, bucket_1s = ?,
		bucket_2500ms = ?, bucket_5s = ?, bucket_10s = ?, bucket_inf = ?
		WHERE id = ?`
	args := []any{
		m.Bucket5ms, m.Bucket10ms, m.Bucket25ms, m.Bucket50ms,
		m.Bucket100ms, m.Bucket250ms, m.Bucket500ms, m.Bucket1s,
		m.Bucket2500ms, m.Bucket5s, m.Bucket10s, m.BucketInf,
		m.ID,
	}
	return query, args
}

func (d *sloDialect) DeleteRawMetricsBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_metrics_raw WHERE timestamp < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloDialect) ScanRawMetrics(rows *sql.Rows) (*database.SLOMetricsRaw, error) {
	m := &database.SLOMetricsRaw{}
	var timestamp string
	var isMissing int
	var domain, pathPrefix sql.NullString
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Host, &domain, &pathPrefix, &timestamp,
		&m.TotalRequests, &m.ErrorRequests, &m.SumLatencyMs,
		&m.Bucket5ms, &m.Bucket10ms, &m.Bucket25ms, &m.Bucket50ms,
		&m.Bucket100ms, &m.Bucket250ms, &m.Bucket500ms, &m.Bucket1s,
		&m.Bucket2500ms, &m.Bucket5s, &m.Bucket10s, &m.BucketInf,
		&isMissing)
	if err != nil {
		return nil, err
	}
	m.Domain = domain.String
	m.PathPrefix = pathPrefix.String
	if m.PathPrefix == "" {
		m.PathPrefix = "/"
	}
	m.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
	m.IsMissing = isMissing == 1
	return m, nil
}

// ==================== Hourly Metrics ====================

func (d *sloDialect) UpsertHourlyMetrics(m *database.SLOMetricsHourly) (string, []any) {
	query := `INSERT INTO slo_metrics_hourly (cluster_id, host, domain, path_prefix, hour_start, total_requests, error_requests, availability,
		p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, avg_rps,
		bucket_5ms, bucket_10ms, bucket_25ms, bucket_50ms, bucket_100ms, bucket_250ms, bucket_500ms,
		bucket_1s, bucket_2500ms, bucket_5s, bucket_10s, bucket_inf, sample_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, host, hour_start) DO UPDATE SET
		domain = excluded.domain,
		path_prefix = excluded.path_prefix,
		total_requests = excluded.total_requests,
		error_requests = excluded.error_requests,
		availability = excluded.availability,
		p50_latency_ms = excluded.p50_latency_ms,
		p95_latency_ms = excluded.p95_latency_ms,
		p99_latency_ms = excluded.p99_latency_ms,
		avg_latency_ms = excluded.avg_latency_ms,
		avg_rps = excluded.avg_rps,
		bucket_5ms = excluded.bucket_5ms,
		bucket_10ms = excluded.bucket_10ms,
		bucket_25ms = excluded.bucket_25ms,
		bucket_50ms = excluded.bucket_50ms,
		bucket_100ms = excluded.bucket_100ms,
		bucket_250ms = excluded.bucket_250ms,
		bucket_500ms = excluded.bucket_500ms,
		bucket_1s = excluded.bucket_1s,
		bucket_2500ms = excluded.bucket_2500ms,
		bucket_5s = excluded.bucket_5s,
		bucket_10s = excluded.bucket_10s,
		bucket_inf = excluded.bucket_inf,
		sample_count = excluded.sample_count`
	args := []any{
		m.ClusterID, m.Host, m.Domain, m.PathPrefix, m.HourStart.Format(time.RFC3339),
		m.TotalRequests, m.ErrorRequests, m.Availability,
		m.P50LatencyMs, m.P95LatencyMs, m.P99LatencyMs, m.AvgLatencyMs, m.AvgRPS,
		m.Bucket5ms, m.Bucket10ms, m.Bucket25ms, m.Bucket50ms,
		m.Bucket100ms, m.Bucket250ms, m.Bucket500ms, m.Bucket1s,
		m.Bucket2500ms, m.Bucket5s, m.Bucket10s, m.BucketInf,
		m.SampleCount, m.CreatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) SelectHourlyMetrics(clusterID, host string, start, end time.Time) (string, []any) {
	return `SELECT id, cluster_id, host, domain, path_prefix, hour_start, total_requests, error_requests, availability,
		p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, avg_rps,
		bucket_5ms, bucket_10ms, bucket_25ms, bucket_50ms, bucket_100ms, bucket_250ms, bucket_500ms,
		bucket_1s, bucket_2500ms, bucket_5s, bucket_10s, bucket_inf, sample_count, created_at
		FROM slo_metrics_hourly
		WHERE cluster_id = ? AND host = ? AND hour_start >= ? AND hour_start < ?
		ORDER BY hour_start ASC`,
		[]any{clusterID, host, start.Format(time.RFC3339), end.Format(time.RFC3339)}
}

func (d *sloDialect) DeleteHourlyMetricsBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_metrics_hourly WHERE hour_start < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloDialect) ScanHourlyMetrics(rows *sql.Rows) (*database.SLOMetricsHourly, error) {
	m := &database.SLOMetricsHourly{}
	var hourStart, createdAt string
	var domain, pathPrefix sql.NullString
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Host, &domain, &pathPrefix, &hourStart,
		&m.TotalRequests, &m.ErrorRequests, &m.Availability,
		&m.P50LatencyMs, &m.P95LatencyMs, &m.P99LatencyMs, &m.AvgLatencyMs, &m.AvgRPS,
		&m.Bucket5ms, &m.Bucket10ms, &m.Bucket25ms, &m.Bucket50ms,
		&m.Bucket100ms, &m.Bucket250ms, &m.Bucket500ms, &m.Bucket1s,
		&m.Bucket2500ms, &m.Bucket5s, &m.Bucket10s, &m.BucketInf,
		&m.SampleCount, &createdAt)
	if err != nil {
		return nil, err
	}
	m.Domain = domain.String
	m.PathPrefix = pathPrefix.String
	if m.PathPrefix == "" {
		m.PathPrefix = "/"
	}
	m.HourStart, _ = time.Parse(time.RFC3339, hourStart)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return m, nil
}

// ==================== Targets ====================

func (d *sloDialect) SelectTargets(clusterID string) (string, []any) {
	return `SELECT id, cluster_id, host, ingress_name, ingress_class, namespace, tls, time_range, availability_target, p95_latency_target, created_at, updated_at
		FROM slo_targets WHERE cluster_id = ? ORDER BY host, time_range`, []any{clusterID}
}

func (d *sloDialect) SelectTargetsByHost(clusterID, host string) (string, []any) {
	return `SELECT id, cluster_id, host, ingress_name, ingress_class, namespace, tls, time_range, availability_target, p95_latency_target, created_at, updated_at
		FROM slo_targets WHERE cluster_id = ? AND host = ? ORDER BY time_range`, []any{clusterID, host}
}

func (d *sloDialect) UpsertTarget(t *database.SLOTarget) (string, []any) {
	query := `INSERT INTO slo_targets (cluster_id, host, ingress_name, ingress_class, namespace, tls, time_range, availability_target, p95_latency_target, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, host, time_range) DO UPDATE SET
		ingress_name = excluded.ingress_name,
		ingress_class = excluded.ingress_class,
		namespace = excluded.namespace,
		tls = excluded.tls,
		availability_target = excluded.availability_target,
		p95_latency_target = excluded.p95_latency_target,
		updated_at = excluded.updated_at`
	args := []any{
		t.ClusterID, t.Host, t.IngressName, t.IngressClass, t.Namespace,
		boolToInt(t.TLS), t.TimeRange, t.AvailabilityTarget, t.P95LatencyTarget,
		t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) DeleteTarget(clusterID, host, timeRange string) (string, []any) {
	return `DELETE FROM slo_targets WHERE cluster_id = ? AND host = ? AND time_range = ?`,
		[]any{clusterID, host, timeRange}
}

func (d *sloDialect) ScanTarget(rows *sql.Rows) (*database.SLOTarget, error) {
	t := &database.SLOTarget{}
	var tls int
	var createdAt, updatedAt string
	err := rows.Scan(&t.ID, &t.ClusterID, &t.Host, &t.IngressName, &t.IngressClass,
		&t.Namespace, &tls, &t.TimeRange, &t.AvailabilityTarget, &t.P95LatencyTarget,
		&createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	t.TLS = tls == 1
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return t, nil
}

// ==================== Status History ====================

func (d *sloDialect) InsertStatusHistory(h *database.SLOStatusHistory) (string, []any) {
	query := `INSERT INTO slo_status_history (cluster_id, host, time_range, old_status, new_status, availability, p95_latency, error_budget_remaining, changed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		h.ClusterID, h.Host, h.TimeRange, h.OldStatus, h.NewStatus,
		h.Availability, h.P95Latency, h.ErrorBudgetRemaining,
		h.ChangedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) SelectStatusHistory(clusterID, host string, limit int) (string, []any) {
	if host == "" {
		return `SELECT id, cluster_id, host, time_range, old_status, new_status, availability, p95_latency, error_budget_remaining, changed_at
			FROM slo_status_history WHERE cluster_id = ? ORDER BY changed_at DESC LIMIT ?`,
			[]any{clusterID, limit}
	}
	return `SELECT id, cluster_id, host, time_range, old_status, new_status, availability, p95_latency, error_budget_remaining, changed_at
		FROM slo_status_history WHERE cluster_id = ? AND host = ? ORDER BY changed_at DESC LIMIT ?`,
		[]any{clusterID, host, limit}
}

func (d *sloDialect) DeleteStatusHistoryBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_status_history WHERE changed_at < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloDialect) ScanStatusHistory(rows *sql.Rows) (*database.SLOStatusHistory, error) {
	h := &database.SLOStatusHistory{}
	var changedAt string
	err := rows.Scan(&h.ID, &h.ClusterID, &h.Host, &h.TimeRange,
		&h.OldStatus, &h.NewStatus, &h.Availability, &h.P95Latency,
		&h.ErrorBudgetRemaining, &changedAt)
	if err != nil {
		return nil, err
	}
	h.ChangedAt, _ = time.Parse(time.RFC3339, changedAt)
	return h, nil
}

// ==================== Domain List ====================

func (d *sloDialect) SelectAllHosts(clusterID string) (string, []any) {
	return `SELECT DISTINCT host FROM ingress_counter_snapshot WHERE cluster_id = ? ORDER BY host`,
		[]any{clusterID}
}

func (d *sloDialect) SelectAllClusterIDs() (string, []any) {
	return `SELECT DISTINCT cluster_id FROM slo_metrics_raw ORDER BY cluster_id`, nil
}

// ==================== Route Mapping ====================

func (d *sloDialect) UpsertRouteMapping(m *database.SLORouteMapping) (string, []any) {
	// 唯一约束基于 cluster_id + domain + path_prefix
	// 同一个 service 可能服务于多个路径
	query := `INSERT INTO slo_route_mapping (cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, domain, path_prefix) DO UPDATE SET
		ingress_name = excluded.ingress_name,
		namespace = excluded.namespace,
		tls = excluded.tls,
		service_key = excluded.service_key,
		service_name = excluded.service_name,
		service_port = excluded.service_port,
		updated_at = excluded.updated_at`
	args := []any{
		m.ClusterID, m.Domain, m.PathPrefix, m.IngressName, m.Namespace,
		boolToInt(m.TLS), m.ServiceKey, m.ServiceName, m.ServicePort,
		m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) SelectRouteMappingByServiceKey(clusterID, serviceKey string) (string, []any) {
	return `SELECT id, cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at
		FROM slo_route_mapping WHERE cluster_id = ? AND service_key = ?`,
		[]any{clusterID, serviceKey}
}

func (d *sloDialect) SelectRouteMappingsByDomain(clusterID, domain string) (string, []any) {
	return `SELECT id, cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at
		FROM slo_route_mapping WHERE cluster_id = ? AND domain = ? ORDER BY path_prefix`,
		[]any{clusterID, domain}
}

func (d *sloDialect) SelectAllRouteMappings(clusterID string) (string, []any) {
	return `SELECT id, cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at
		FROM slo_route_mapping WHERE cluster_id = ? ORDER BY domain, path_prefix`,
		[]any{clusterID}
}

func (d *sloDialect) SelectAllDomains(clusterID string) (string, []any) {
	return `SELECT DISTINCT domain FROM slo_route_mapping WHERE cluster_id = ? ORDER BY domain`,
		[]any{clusterID}
}

func (d *sloDialect) DeleteRouteMapping(clusterID, serviceKey string) (string, []any) {
	return `DELETE FROM slo_route_mapping WHERE cluster_id = ? AND service_key = ?`,
		[]any{clusterID, serviceKey}
}

func (d *sloDialect) ScanRouteMapping(rows *sql.Rows) (*database.SLORouteMapping, error) {
	m := &database.SLORouteMapping{}
	var tls int
	var createdAt, updatedAt string
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Domain, &m.PathPrefix, &m.IngressName,
		&m.Namespace, &tls, &m.ServiceKey, &m.ServiceName, &m.ServicePort,
		&createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	m.TLS = tls == 1
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return m, nil
}

// ==================== Helper ====================

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

var _ database.SLODialect = (*sloDialect)(nil)
