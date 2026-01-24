// atlhyper_master_v2/database/sqlite/settings.go
// SQLite SettingsDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type settingsDialect struct{}

func (d *settingsDialect) SelectByKey(key string) (string, []any) {
	return "SELECT key, value, description, updated_at, updated_by FROM settings WHERE key = ?", []any{key}
}

func (d *settingsDialect) Upsert(s *database.Setting) (string, []any) {
	query := `INSERT INTO settings (key, value, description, updated_at, updated_by)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value, description = excluded.description,
		updated_at = excluded.updated_at, updated_by = excluded.updated_by`
	args := []any{s.Key, s.Value, s.Description, time.Now().Format(time.RFC3339), s.UpdatedBy}
	return query, args
}

func (d *settingsDialect) Delete(key string) (string, []any) {
	return "DELETE FROM settings WHERE key = ?", []any{key}
}

func (d *settingsDialect) SelectAll() (string, []any) {
	return "SELECT key, value, description, updated_at, updated_by FROM settings", nil
}

func (d *settingsDialect) ScanRow(rows *sql.Rows) (*database.Setting, error) {
	s := &database.Setting{}
	var updatedAt string
	err := rows.Scan(&s.Key, &s.Value, &s.Description, &updatedAt, &s.UpdatedBy)
	if err != nil {
		return nil, err
	}
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return s, nil
}

var _ database.SettingsDialect = (*settingsDialect)(nil)
