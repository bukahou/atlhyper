// atlhyper_master_v2/database/repo/ai_active.go
// AI Active Config Repository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiActiveConfigRepo struct {
	db      *sql.DB
	dialect database.AIActiveConfigDialect
}

func newAIActiveConfigRepo(db *sql.DB, dialect database.AIActiveConfigDialect) *aiActiveConfigRepo {
	return &aiActiveConfigRepo{db: db, dialect: dialect}
}

func (r *aiActiveConfigRepo) Get(ctx context.Context) (*database.AIActiveConfig, error) {
	query, args := r.dialect.Select()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return r.dialect.ScanRow(rows)
	}
	return nil, nil
}

func (r *aiActiveConfigRepo) Update(ctx context.Context, cfg *database.AIActiveConfig) error {
	query, args := r.dialect.Update(cfg)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiActiveConfigRepo) SwitchProvider(ctx context.Context, providerID int64, updatedBy int64) error {
	query, args := r.dialect.SwitchProvider(providerID, updatedBy)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiActiveConfigRepo) SetEnabled(ctx context.Context, enabled bool, updatedBy int64) error {
	query, args := r.dialect.SetEnabled(enabled, updatedBy)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

var _ database.AIActiveConfigRepository = (*aiActiveConfigRepo)(nil)
