// atlhyper_master_v2/database/repo/slo.go
// SLORepository 实现（入口指标 + 目标 + 状态 + 路由映射）
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloRepo struct {
	db      *sql.DB
	dialect database.SLODialect
}

func newSLORepo(db *sql.DB, dialect database.SLODialect) *sloRepo {
	return &sloRepo{db: db, dialect: dialect}
}

// ==================== Raw Metrics ====================

func (r *sloRepo) InsertRawMetrics(ctx context.Context, m *database.SLOMetricsRaw) error {
	query, args := r.dialect.InsertRawMetrics(m)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	m.ID = id
	return nil
}

func (r *sloRepo) GetRawMetrics(ctx context.Context, clusterID, host string, start, end time.Time) ([]*database.SLOMetricsRaw, error) {
	query, args := r.dialect.SelectRawMetrics(clusterID, host, start, end)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*database.SLOMetricsRaw
	for rows.Next() {
		m, err := r.dialect.ScanRawMetrics(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *sloRepo) DeleteRawMetricsBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteRawMetricsBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ==================== Hourly Metrics ====================

func (r *sloRepo) UpsertHourlyMetrics(ctx context.Context, m *database.SLOMetricsHourly) error {
	query, args := r.dialect.UpsertHourlyMetrics(m)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if m.ID == 0 {
		id, _ := result.LastInsertId()
		m.ID = id
	}
	return nil
}

func (r *sloRepo) GetHourlyMetrics(ctx context.Context, clusterID, host string, start, end time.Time) ([]*database.SLOMetricsHourly, error) {
	query, args := r.dialect.SelectHourlyMetrics(clusterID, host, start, end)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*database.SLOMetricsHourly
	for rows.Next() {
		m, err := r.dialect.ScanHourlyMetrics(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *sloRepo) DeleteHourlyMetricsBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteHourlyMetricsBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ==================== Targets ====================

func (r *sloRepo) GetTargets(ctx context.Context, clusterID string) ([]*database.SLOTarget, error) {
	query, args := r.dialect.SelectTargets(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []*database.SLOTarget
	for rows.Next() {
		t, err := r.dialect.ScanTarget(rows)
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func (r *sloRepo) GetTargetsByHost(ctx context.Context, clusterID, host string) ([]*database.SLOTarget, error) {
	query, args := r.dialect.SelectTargetsByHost(clusterID, host)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []*database.SLOTarget
	for rows.Next() {
		t, err := r.dialect.ScanTarget(rows)
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func (r *sloRepo) UpsertTarget(ctx context.Context, t *database.SLOTarget) error {
	query, args := r.dialect.UpsertTarget(t)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if t.ID == 0 {
		id, _ := result.LastInsertId()
		t.ID = id
	}
	return nil
}

func (r *sloRepo) DeleteTarget(ctx context.Context, clusterID, host, timeRange string) error {
	query, args := r.dialect.DeleteTarget(clusterID, host, timeRange)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// ==================== Status History ====================

func (r *sloRepo) InsertStatusHistory(ctx context.Context, h *database.SLOStatusHistory) error {
	query, args := r.dialect.InsertStatusHistory(h)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	h.ID = id
	return nil
}

func (r *sloRepo) GetStatusHistory(ctx context.Context, clusterID, host string, limit int) ([]*database.SLOStatusHistory, error) {
	query, args := r.dialect.SelectStatusHistory(clusterID, host, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*database.SLOStatusHistory
	for rows.Next() {
		h, err := r.dialect.ScanStatusHistory(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func (r *sloRepo) DeleteStatusHistoryBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteStatusHistoryBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ==================== Domain List ====================

func (r *sloRepo) GetAllHosts(ctx context.Context, clusterID string) ([]string, error) {
	query, args := r.dialect.SelectAllHosts(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []string
	for rows.Next() {
		var host string
		if err := rows.Scan(&host); err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	return hosts, rows.Err()
}

func (r *sloRepo) GetAllClusterIDs(ctx context.Context) ([]string, error) {
	query, args := r.dialect.SelectAllClusterIDs()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusterIDs []string
	for rows.Next() {
		var clusterID string
		if err := rows.Scan(&clusterID); err != nil {
			return nil, err
		}
		clusterIDs = append(clusterIDs, clusterID)
	}
	return clusterIDs, rows.Err()
}

// ==================== Route Mapping ====================

func (r *sloRepo) UpsertRouteMapping(ctx context.Context, m *database.SLORouteMapping) error {
	query, args := r.dialect.UpsertRouteMapping(m)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if m.ID == 0 {
		id, _ := result.LastInsertId()
		m.ID = id
	}
	return nil
}

func (r *sloRepo) GetRouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*database.SLORouteMapping, error) {
	query, args := r.dialect.SelectRouteMappingByServiceKey(clusterID, serviceKey)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return r.dialect.ScanRouteMapping(rows)
	}
	return nil, nil // 未找到返回 nil
}

func (r *sloRepo) GetRouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*database.SLORouteMapping, error) {
	query, args := r.dialect.SelectRouteMappingsByDomain(clusterID, domain)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []*database.SLORouteMapping
	for rows.Next() {
		m, err := r.dialect.ScanRouteMapping(rows)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

func (r *sloRepo) GetAllRouteMappings(ctx context.Context, clusterID string) ([]*database.SLORouteMapping, error) {
	query, args := r.dialect.SelectAllRouteMappings(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []*database.SLORouteMapping
	for rows.Next() {
		m, err := r.dialect.ScanRouteMapping(rows)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

func (r *sloRepo) GetAllDomains(ctx context.Context, clusterID string) ([]string, error) {
	query, args := r.dialect.SelectAllDomains(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, rows.Err()
}

func (r *sloRepo) DeleteRouteMapping(ctx context.Context, clusterID, serviceKey string) error {
	query, args := r.dialect.DeleteRouteMapping(clusterID, serviceKey)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

var _ database.SLORepository = (*sloRepo)(nil)
