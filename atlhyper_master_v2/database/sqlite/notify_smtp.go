// atlhyper_master_v2/database/sqlite/notify_smtp.go
// SQLite NotifySMTPDialect 实现（单行表，id 恒为 1；配置列是密文）
package sqlite

import (
	"database/sql"
	"time"
)

type notifySMTPDialect struct{}

func (d *notifySMTPDialect) Select() (string, []any) {
	return "SELECT config, updated_at FROM notify_smtp WHERE id = 1", nil
}

func (d *notifySMTPDialect) Upsert(encryptedConfig string, now time.Time) (string, []any) {
	query := `INSERT INTO notify_smtp (id, config, updated_at) VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET config = excluded.config, updated_at = excluded.updated_at`
	return query, []any{encryptedConfig, utcStamp(now)}
}

func (d *notifySMTPDialect) ScanRow(rows *sql.Rows) (string, time.Time, error) {
	var encryptedConfig, updatedAt string
	if err := rows.Scan(&encryptedConfig, &updatedAt); err != nil {
		return "", time.Time{}, err
	}
	return encryptedConfig, parseStamp(updatedAt), nil
}
