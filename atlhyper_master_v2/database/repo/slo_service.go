// atlhyper_master_v2/database/repo/slo_service.go
// SLOServiceRepository 实现（服务网格数据）
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloServiceRepo struct {
	db      *sql.DB
	dialect database.SLOServiceDialect
}

func newSLOServiceRepo(db *sql.DB, dialect database.SLOServiceDialect) *sloServiceRepo {
	return &sloServiceRepo{db: db, dialect: dialect}
}

// ==================== Service Raw ====================

func (r *sloServiceRepo) InsertServiceRaw(ctx context.Context, m *database.SLOServiceRaw) error {
	query, args := r.dialect.InsertServiceRaw(m)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	m.ID = id
	return nil
}

func (r *sloServiceRepo) GetServiceRaw(ctx context.Context, clusterID, namespace, name string, start, end time.Time) ([]*database.SLOServiceRaw, error) {
	query, args := r.dialect.SelectServiceRaw(clusterID, namespace, name, start, end)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*database.SLOServiceRaw
	for rows.Next() {
		m, err := r.dialect.ScanServiceRaw(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *sloServiceRepo) DeleteServiceRawBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteServiceRawBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ==================== Service Hourly ====================

func (r *sloServiceRepo) UpsertServiceHourly(ctx context.Context, m *database.SLOServiceHourly) error {
	query, args := r.dialect.UpsertServiceHourly(m)
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

func (r *sloServiceRepo) GetServiceHourly(ctx context.Context, clusterID, namespace, name string, start, end time.Time) ([]*database.SLOServiceHourly, error) {
	query, args := r.dialect.SelectServiceHourly(clusterID, namespace, name, start, end)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*database.SLOServiceHourly
	for rows.Next() {
		m, err := r.dialect.ScanServiceHourly(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *sloServiceRepo) DeleteServiceHourlyBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteServiceHourlyBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

var _ database.SLOServiceRepository = (*sloServiceRepo)(nil)
