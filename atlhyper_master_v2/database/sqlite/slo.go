// atlhyper_master_v2/database/sqlite/slo.go
// SQLite SLO Dialect 实现
// 入口指标表 SQL: slo_metrics_raw + slo_metrics_hourly + targets + status + route_mapping
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloDialect struct{}

// ==================== Raw Metrics (入口) ====================

func (d *sloDialect) InsertRawMetrics(m *database.SLOMetricsRaw) (string, []any) {
	query := `INSERT INTO slo_metrics_raw (cluster_id, host, domain, path_prefix, timestamp,
		total_requests, error_requests, latency_sum, latency_count, latency_buckets,
		method_get, method_post, method_put, method_delete, method_other, is_missing)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		m.ClusterID, m.Host, m.Domain, m.PathPrefix, m.Timestamp.Format(time.RFC3339),
		m.TotalRequests, m.ErrorRequests, m.LatencySum, m.LatencyCount, m.LatencyBuckets,
		m.MethodGet, m.MethodPost, m.MethodPut, m.MethodDelete, m.MethodOther,
		boolToInt(m.IsMissing),
	}
	return query, args
}

func (d *sloDialect) SelectRawMetrics(clusterID, host string, start, end time.Time) (string, []any) {
	return `SELECT id, cluster_id, host, domain, path_prefix, timestamp,
		total_requests, error_requests, latency_sum, latency_count, latency_buckets,
		method_get, method_post, method_put, method_delete, method_other, is_missing
		FROM slo_metrics_raw
		WHERE cluster_id = ? AND host = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp ASC`,
		[]any{clusterID, host, start.Format(time.RFC3339), end.Format(time.RFC3339)}
}

func (d *sloDialect) DeleteRawMetricsBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_metrics_raw WHERE timestamp < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloDialect) ScanRawMetrics(rows *sql.Rows) (*database.SLOMetricsRaw, error) {
	m := &database.SLOMetricsRaw{}
	var timestamp string
	var isMissing int
	var domain, pathPrefix, latencyBuckets sql.NullString
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Host, &domain, &pathPrefix, &timestamp,
		&m.TotalRequests, &m.ErrorRequests, &m.LatencySum, &m.LatencyCount, &latencyBuckets,
		&m.MethodGet, &m.MethodPost, &m.MethodPut, &m.MethodDelete, &m.MethodOther,
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
	m.LatencyBuckets = latencyBuckets.String
	m.IsMissing = isMissing == 1
	return m, nil
}

// ==================== Hourly Metrics (入口) ====================

func (d *sloDialect) UpsertHourlyMetrics(m *database.SLOMetricsHourly) (string, []any) {
	query := `INSERT INTO slo_metrics_hourly (cluster_id, host, domain, path_prefix, hour_start,
		total_requests, error_requests, availability,
		p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, avg_rps,
		latency_buckets, method_get, method_post, method_put, method_delete, method_other,
		sample_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		latency_buckets = excluded.latency_buckets,
		method_get = excluded.method_get,
		method_post = excluded.method_post,
		method_put = excluded.method_put,
		method_delete = excluded.method_delete,
		method_other = excluded.method_other,
		sample_count = excluded.sample_count`
	args := []any{
		m.ClusterID, m.Host, m.Domain, m.PathPrefix, m.HourStart.Format(time.RFC3339),
		m.TotalRequests, m.ErrorRequests, m.Availability,
		m.P50LatencyMs, m.P95LatencyMs, m.P99LatencyMs, m.AvgLatencyMs, m.AvgRPS,
		m.LatencyBuckets, m.MethodGet, m.MethodPost, m.MethodPut, m.MethodDelete, m.MethodOther,
		m.SampleCount, m.CreatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) SelectHourlyMetrics(clusterID, host string, start, end time.Time) (string, []any) {
	return `SELECT id, cluster_id, host, domain, path_prefix, hour_start,
		total_requests, error_requests, availability,
		p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, avg_rps,
		latency_buckets, method_get, method_post, method_put, method_delete, method_other,
		sample_count, created_at
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
	var domain, pathPrefix, latencyBuckets sql.NullString
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Host, &domain, &pathPrefix, &hourStart,
		&m.TotalRequests, &m.ErrorRequests, &m.Availability,
		&m.P50LatencyMs, &m.P95LatencyMs, &m.P99LatencyMs, &m.AvgLatencyMs, &m.AvgRPS,
		&latencyBuckets, &m.MethodGet, &m.MethodPost, &m.MethodPut, &m.MethodDelete, &m.MethodOther,
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
	m.LatencyBuckets = latencyBuckets.String
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
	return `SELECT DISTINCT host FROM slo_metrics_raw WHERE cluster_id = ? ORDER BY host`,
		[]any{clusterID}
}

func (d *sloDialect) SelectAllClusterIDs() (string, []any) {
	return `SELECT DISTINCT cluster_id FROM slo_metrics_raw ORDER BY cluster_id`, nil
}

// ==================== Route Mapping ====================

func (d *sloDialect) UpsertRouteMapping(m *database.SLORouteMapping) (string, []any) {
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

var _ database.SLODialect = (*sloDialect)(nil)
