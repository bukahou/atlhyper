// atlhyper_master_v2/database/sqlite/impl/settings.go
// SettingsRepository SQLite 实现
package impl

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Get(ctx context.Context, key string) (*repository.Setting, error) {
	row := r.db.QueryRowContext(ctx, "SELECT * FROM settings WHERE key = ?", key)
	s := &repository.Setting{}
	var updatedAt string
	err := row.Scan(&s.Key, &s.Value, &s.Description, &updatedAt, &s.UpdatedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return s, nil
}

func (r *SettingsRepository) Set(ctx context.Context, s *repository.Setting) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO settings (key, value, description, updated_at, updated_by)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, description = excluded.description,
			updated_at = excluded.updated_at, updated_by = excluded.updated_by`,
		s.Key, s.Value, s.Description, time.Now().Format(time.RFC3339), s.UpdatedBy,
	)
	return err
}

func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM settings WHERE key = ?", key)
	return err
}

func (r *SettingsRepository) List(ctx context.Context) ([]*repository.Setting, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT * FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*repository.Setting
	for rows.Next() {
		s := &repository.Setting{}
		var updatedAt string
		if err := rows.Scan(&s.Key, &s.Value, &s.Description, &updatedAt, &s.UpdatedBy); err != nil {
			return nil, err
		}
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		settings = append(settings, s)
	}
	return settings, rows.Err()
}

var _ repository.SettingsRepository = (*SettingsRepository)(nil)
