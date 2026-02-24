// atlhyper_master_v2/database/repo/slo.go
// SLORepository 实现（目标配置 + 路由映射）
// 时序数据（raw/hourly）已迁移至 OTelSnapshot + ClickHouse
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloRepo struct {
	db      *sql.DB
	dialect database.SLODialect
}

func newSLORepo(db *sql.DB, dialect database.SLODialect) *sloRepo {
	return &sloRepo{db: db, dialect: dialect}
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
	return nil, nil
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
