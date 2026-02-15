// atlhyper_master_v2/database/repo/aiops_baseline.go
// AIOps 基线 Repository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

// aiopsBaselineRepo AIOps 基线 Repository 实现
type aiopsBaselineRepo struct {
	db      *sql.DB
	dialect database.AIOpsBaselineDialect
}

// newAIOpsBaselineRepo 创建 AIOps 基线 Repository
func newAIOpsBaselineRepo(db *sql.DB, dialect database.AIOpsBaselineDialect) *aiopsBaselineRepo {
	return &aiopsBaselineRepo{db: db, dialect: dialect}
}

// BatchUpsert 批量插入或更新基线状态
func (r *aiopsBaselineRepo) BatchUpsert(ctx context.Context, states []*database.AIOpsBaselineState) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, state := range states {
		query, args := r.dialect.Upsert(state)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListAll 查询所有基线状态
func (r *aiopsBaselineRepo) ListAll(ctx context.Context) ([]*database.AIOpsBaselineState, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.AIOpsBaselineState
	for rows.Next() {
		s, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// ListByEntity 按实体查询基线状态
func (r *aiopsBaselineRepo) ListByEntity(ctx context.Context, entityKey string) ([]*database.AIOpsBaselineState, error) {
	query, args := r.dialect.SelectByEntity(entityKey)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.AIOpsBaselineState
	for rows.Next() {
		s, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// DeleteByEntity 按实体删除基线状态
func (r *aiopsBaselineRepo) DeleteByEntity(ctx context.Context, entityKey string) error {
	query, args := r.dialect.DeleteByEntity(entityKey)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// 确保实现了接口
var _ database.AIOpsBaselineRepository = (*aiopsBaselineRepo)(nil)
