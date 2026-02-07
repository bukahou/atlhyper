// atlhyper_master_v2/database/sqlite/slo_edge.go
// SQLite SLO Edge Dialect 实现
// 拓扑边表 SQL: slo_edge_raw + slo_edge_hourly
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloEdgeDialect struct{}

// ==================== Edge Raw ====================

func (d *sloEdgeDialect) InsertEdgeRaw(m *database.SLOEdgeRaw) (string, []any) {
	query := `INSERT INTO slo_edge_raw (cluster_id, src_namespace, src_name, dst_namespace, dst_name, timestamp,
		request_delta, failure_delta, latency_sum, latency_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		m.ClusterID, m.SrcNamespace, m.SrcName, m.DstNamespace, m.DstName,
		m.Timestamp.Format(time.RFC3339),
		m.RequestDelta, m.FailureDelta, m.LatencySum, m.LatencyCount,
	}
	return query, args
}

func (d *sloEdgeDialect) SelectEdgeRaw(clusterID string, start, end time.Time) (string, []any) {
	return `SELECT id, cluster_id, src_namespace, src_name, dst_namespace, dst_name, timestamp,
		request_delta, failure_delta, latency_sum, latency_count
		FROM slo_edge_raw
		WHERE cluster_id = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp ASC`,
		[]any{clusterID, start.Format(time.RFC3339), end.Format(time.RFC3339)}
}

func (d *sloEdgeDialect) DeleteEdgeRawBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_edge_raw WHERE timestamp < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloEdgeDialect) ScanEdgeRaw(rows *sql.Rows) (*database.SLOEdgeRaw, error) {
	m := &database.SLOEdgeRaw{}
	var timestamp string
	err := rows.Scan(&m.ID, &m.ClusterID, &m.SrcNamespace, &m.SrcName,
		&m.DstNamespace, &m.DstName, &timestamp,
		&m.RequestDelta, &m.FailureDelta, &m.LatencySum, &m.LatencyCount)
	if err != nil {
		return nil, err
	}
	m.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
	return m, nil
}

// ==================== Edge Hourly ====================

func (d *sloEdgeDialect) UpsertEdgeHourly(m *database.SLOEdgeHourly) (string, []any) {
	query := `INSERT INTO slo_edge_hourly (cluster_id, src_namespace, src_name, dst_namespace, dst_name, hour_start,
		total_requests, error_requests, avg_latency_ms, avg_rps, error_rate,
		sample_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, src_namespace, src_name, dst_namespace, dst_name, hour_start) DO UPDATE SET
		total_requests = excluded.total_requests,
		error_requests = excluded.error_requests,
		avg_latency_ms = excluded.avg_latency_ms,
		avg_rps = excluded.avg_rps,
		error_rate = excluded.error_rate,
		sample_count = excluded.sample_count`
	args := []any{
		m.ClusterID, m.SrcNamespace, m.SrcName, m.DstNamespace, m.DstName,
		m.HourStart.Format(time.RFC3339),
		m.TotalRequests, m.ErrorRequests,
		m.AvgLatencyMs, m.AvgRPS, m.ErrorRate,
		m.SampleCount, m.CreatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloEdgeDialect) SelectEdgeHourly(clusterID string, start, end time.Time) (string, []any) {
	return `SELECT id, cluster_id, src_namespace, src_name, dst_namespace, dst_name, hour_start,
		total_requests, error_requests, avg_latency_ms, avg_rps, error_rate,
		sample_count, created_at
		FROM slo_edge_hourly
		WHERE cluster_id = ? AND hour_start >= ? AND hour_start < ?
		ORDER BY hour_start ASC`,
		[]any{clusterID, start.Format(time.RFC3339), end.Format(time.RFC3339)}
}

func (d *sloEdgeDialect) DeleteEdgeHourlyBefore(before time.Time) (string, []any) {
	return `DELETE FROM slo_edge_hourly WHERE hour_start < ?`, []any{before.Format(time.RFC3339)}
}

func (d *sloEdgeDialect) ScanEdgeHourly(rows *sql.Rows) (*database.SLOEdgeHourly, error) {
	m := &database.SLOEdgeHourly{}
	var hourStart, createdAt string
	var avgLatency sql.NullInt64
	var avgRPS, errorRate sql.NullFloat64
	err := rows.Scan(&m.ID, &m.ClusterID, &m.SrcNamespace, &m.SrcName,
		&m.DstNamespace, &m.DstName, &hourStart,
		&m.TotalRequests, &m.ErrorRequests,
		&avgLatency, &avgRPS, &errorRate,
		&m.SampleCount, &createdAt)
	if err != nil {
		return nil, err
	}
	m.HourStart, _ = time.Parse(time.RFC3339, hourStart)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if avgLatency.Valid {
		m.AvgLatencyMs = int(avgLatency.Int64)
	}
	if avgRPS.Valid {
		m.AvgRPS = avgRPS.Float64
	}
	if errorRate.Valid {
		m.ErrorRate = errorRate.Float64
	}
	return m, nil
}

var _ database.SLOEdgeDialect = (*sloEdgeDialect)(nil)
