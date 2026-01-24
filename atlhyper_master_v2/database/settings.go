// atlhyper_master_v2/database/settings.go
// SettingsRepository 实现
package database

import (
	"context"
	"database/sql"
)

type settingsRepo struct {
	db      *sql.DB
	dialect SettingsDialect
}

func newSettingsRepo(db *sql.DB, dialect SettingsDialect) *settingsRepo {
	return &settingsRepo{db: db, dialect: dialect}
}

func (r *settingsRepo) Get(ctx context.Context, key string) (*Setting, error) {
	query, args := r.dialect.SelectByKey(key)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	return r.dialect.ScanRow(rows)
}

func (r *settingsRepo) Set(ctx context.Context, s *Setting) error {
	query, args := r.dialect.Upsert(s)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *settingsRepo) Delete(ctx context.Context, key string) error {
	query, args := r.dialect.Delete(key)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *settingsRepo) List(ctx context.Context) ([]*Setting, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*Setting
	for rows.Next() {
		s, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}
