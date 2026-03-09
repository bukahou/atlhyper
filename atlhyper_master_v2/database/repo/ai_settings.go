// atlhyper_master_v2/database/repo/ai_settings.go
// AI Settings Repository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiSettingsRepo struct {
	db      *sql.DB
	dialect database.AISettingsDialect
}

func newAISettingsRepo(db *sql.DB, dialect database.AISettingsDialect) *aiSettingsRepo {
	return &aiSettingsRepo{db: db, dialect: dialect}
}

func (r *aiSettingsRepo) Get(ctx context.Context) (*database.AISettings, error) {
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

func (r *aiSettingsRepo) Update(ctx context.Context, cfg *database.AISettings) error {
	query, args := r.dialect.Update(cfg)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

var _ database.AISettingsRepository = (*aiSettingsRepo)(nil)
