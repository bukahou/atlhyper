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

// DeleteEntities 删除某个集群里若干实体的全部基线行，一个事务内完成
func (r *aiopsBaselineRepo) DeleteEntities(ctx context.Context, clusterID string, entityKeys []string) error {
	if len(entityKeys) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, entityKey := range entityKeys {
		query, args := r.dialect.DeleteEntity(clusterID, entityKey)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// 确保实现了接口
var _ database.AIOpsBaselineRepository = (*aiopsBaselineRepo)(nil)
