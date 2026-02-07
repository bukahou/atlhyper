// atlhyper_master_v2/database/repo/slo_edge.go
// SLOEdgeRepository 实现（拓扑边数据）
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloEdgeRepo struct {
	db      *sql.DB
	dialect database.SLOEdgeDialect
}

func newSLOEdgeRepo(db *sql.DB, dialect database.SLOEdgeDialect) *sloEdgeRepo {
	return &sloEdgeRepo{db: db, dialect: dialect}
}

// ==================== Edge Raw ====================

func (r *sloEdgeRepo) InsertEdgeRaw(ctx context.Context, m *database.SLOEdgeRaw) error {
	query, args := r.dialect.InsertEdgeRaw(m)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	m.ID = id
	return nil
}

func (r *sloEdgeRepo) GetEdgeRaw(ctx context.Context, clusterID string, start, end time.Time) ([]*database.SLOEdgeRaw, error) {
	query, args := r.dialect.SelectEdgeRaw(clusterID, start, end)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*database.SLOEdgeRaw
	for rows.Next() {
		m, err := r.dialect.ScanEdgeRaw(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *sloEdgeRepo) DeleteEdgeRawBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteEdgeRawBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ==================== Edge Hourly ====================

func (r *sloEdgeRepo) UpsertEdgeHourly(ctx context.Context, m *database.SLOEdgeHourly) error {
	query, args := r.dialect.UpsertEdgeHourly(m)
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

func (r *sloEdgeRepo) GetEdgeHourly(ctx context.Context, clusterID string, start, end time.Time) ([]*database.SLOEdgeHourly, error) {
	query, args := r.dialect.SelectEdgeHourly(clusterID, start, end)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*database.SLOEdgeHourly
	for rows.Next() {
		m, err := r.dialect.ScanEdgeHourly(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *sloEdgeRepo) DeleteEdgeHourlyBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteEdgeHourlyBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

var _ database.SLOEdgeRepository = (*sloEdgeRepo)(nil)
