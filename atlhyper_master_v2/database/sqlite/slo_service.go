// atlhyper_master_v2/database/sqlite/slo_service.go
// SQLite SLO Service Dialect 实现
// 服务网格表 SQL: slo_service_raw + slo_service_hourly
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloServiceDialect struct{}

// ==================== Service Raw ====================

func (d *sloServiceDialect) InsertServiceRaw(m *database.SLOServiceRaw) (string, []any) {
	query := `INSERT INTO slo_service_raw (cluster_id, namespace, name, timestamp,
		total_requests, error_requests, status_2xx, status_3xx, status_4xx, status_5xx,
		latency_sum, latency_count, latency_buckets,
		tls_request_delta, total_request_delta)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		m.ClusterID, m.Namespace, m.Name, m.Timestamp.Format(time.RFC3339),
		m.TotalRequests, m.ErrorRequests,
		m.Status2xx, m.Status3xx, m.Status4xx, m.Status5xx,
		m.LatencySum, m.LatencyCount, m.LatencyBuckets,
		m.TLSRequestDelta, m.TotalRequestDelta,
	}
	return query, args
}

func (d *sloServiceDialect) SelectServiceRaw(clusterID, namespace, name string, start, end time.Time) (string, []any) {
	if namespace == "" && name == "" {
		return `SELECT id, cluster_id, namespace, name, timestamp,
			total_requests, error_requests, status_2xx, status_3xx, status_4xx, status_5xx,
			latency_sum, latency_count, latency_buckets,
			tls_request_delta, total_request_delta
			FROM slo_service_raw
			WHERE cluster_id = ? AND timestamp >= ? AND timestamp < ?
			ORDER BY timestamp ASC`,
			[]any{clusterID, start.Format(time.RFC3339), end.Format(time.RFC3339)}
	}
	return `SELECT id, cluster_id, namespace, name, timestamp,
		total_requests, error_requests, status_2xx, status_3xx, status_4xx, status_5xx,
		latency_sum, latency_count, latency_buckets,
		tls_request_delta, total_request_delta
		FROM slo_service_raw
		WHERE cluster_id = ? AND namespace = ? AND name = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp ASC`,
		[]any{clusterID, namespace, name, start.Format(time.RFC3339), end.Format(time.RFC3339)}
}

func (d *sloServiceDialect) DeleteServiceRawBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_service_raw WHERE timestamp < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloServiceDialect) ScanServiceRaw(rows *sql.Rows) (*database.SLOServiceRaw, error) {
	m := &database.SLOServiceRaw{}
	var timestamp string
	var latencyBuckets sql.NullString
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Namespace, &m.Name, &timestamp,
		&m.TotalRequests, &m.ErrorRequests,
		&m.Status2xx, &m.Status3xx, &m.Status4xx, &m.Status5xx,
		&m.LatencySum, &m.LatencyCount, &latencyBuckets,
		&m.TLSRequestDelta, &m.TotalRequestDelta)
	if err != nil {
		return nil, err
	}
	m.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
	m.LatencyBuckets = latencyBuckets.String
	return m, nil
}

// ==================== Service Hourly ====================

func (d *sloServiceDialect) UpsertServiceHourly(m *database.SLOServiceHourly) (string, []any) {
	query := `INSERT INTO slo_service_hourly (cluster_id, namespace, name, hour_start,
		total_requests, error_requests, availability,
		p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, avg_rps,
		status_2xx, status_3xx, status_4xx, status_5xx,
		latency_buckets, mtls_percent, sample_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, namespace, name, hour_start) DO UPDATE SET
		total_requests = excluded.total_requests,
		error_requests = excluded.error_requests,
		availability = excluded.availability,
		p50_latency_ms = excluded.p50_latency_ms,
		p95_latency_ms = excluded.p95_latency_ms,
		p99_latency_ms = excluded.p99_latency_ms,
		avg_latency_ms = excluded.avg_latency_ms,
		avg_rps = excluded.avg_rps,
		status_2xx = excluded.status_2xx,
		status_3xx = excluded.status_3xx,
		status_4xx = excluded.status_4xx,
		status_5xx = excluded.status_5xx,
		latency_buckets = excluded.latency_buckets,
		mtls_percent = excluded.mtls_percent,
		sample_count = excluded.sample_count`
	args := []any{
		m.ClusterID, m.Namespace, m.Name, m.HourStart.Format(time.RFC3339),
		m.TotalRequests, m.ErrorRequests, m.Availability,
		m.P50LatencyMs, m.P95LatencyMs, m.P99LatencyMs, m.AvgLatencyMs, m.AvgRPS,
		m.Status2xx, m.Status3xx, m.Status4xx, m.Status5xx,
		m.LatencyBuckets, m.MtlsPercent, m.SampleCount, m.CreatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloServiceDialect) SelectServiceHourly(clusterID, namespace, name string, start, end time.Time) (string, []any) {
	if namespace == "" && name == "" {
		return `SELECT id, cluster_id, namespace, name, hour_start,
			total_requests, error_requests, availability,
			p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, avg_rps,
			status_2xx, status_3xx, status_4xx, status_5xx,
			latency_buckets, mtls_percent, sample_count, created_at
			FROM slo_service_hourly
			WHERE cluster_id = ? AND hour_start >= ? AND hour_start < ?
			ORDER BY hour_start ASC`,
			[]any{clusterID, start.Format(time.RFC3339), end.Format(time.RFC3339)}
	}
	return `SELECT id, cluster_id, namespace, name, hour_start,
		total_requests, error_requests, availability,
		p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, avg_rps,
		status_2xx, status_3xx, status_4xx, status_5xx,
		latency_buckets, mtls_percent, sample_count, created_at
		FROM slo_service_hourly
		WHERE cluster_id = ? AND namespace = ? AND name = ? AND hour_start >= ? AND hour_start < ?
		ORDER BY hour_start ASC`,
		[]any{clusterID, namespace, name, start.Format(time.RFC3339), end.Format(time.RFC3339)}
}

func (d *sloServiceDialect) DeleteServiceHourlyBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_service_hourly WHERE hour_start < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloServiceDialect) ScanServiceHourly(rows *sql.Rows) (*database.SLOServiceHourly, error) {
	m := &database.SLOServiceHourly{}
	var hourStart, createdAt string
	var latencyBuckets sql.NullString
	var availability, mtlsPercent sql.NullFloat64
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Namespace, &m.Name, &hourStart,
		&m.TotalRequests, &m.ErrorRequests, &availability,
		&m.P50LatencyMs, &m.P95LatencyMs, &m.P99LatencyMs, &m.AvgLatencyMs, &m.AvgRPS,
		&m.Status2xx, &m.Status3xx, &m.Status4xx, &m.Status5xx,
		&latencyBuckets, &mtlsPercent, &m.SampleCount, &createdAt)
	if err != nil {
		return nil, err
	}
	m.HourStart, _ = time.Parse(time.RFC3339, hourStart)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.LatencyBuckets = latencyBuckets.String
	if availability.Valid {
		m.Availability = availability.Float64
	}
	if mtlsPercent.Valid {
		m.MtlsPercent = mtlsPercent.Float64
	}
	return m, nil
}

var _ database.SLOServiceDialect = (*sloServiceDialect)(nil)
