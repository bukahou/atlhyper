// atlhyper_master_v2/database/sqlite/notify.go
// SQLite NotifyDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyDialect struct{}

func (d *notifyDialect) Insert(ch *database.NotifyChannel) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return "INSERT INTO notify_channels (type, name, enabled, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		[]any{ch.Type, ch.Name, ch.Enabled, ch.Config, now, now}
}

func (d *notifyDialect) Update(ch *database.NotifyChannel) (string, []any) {
	return "UPDATE notify_channels SET name = ?, enabled = ?, config = ?, updated_at = ? WHERE id = ?",
		[]any{ch.Name, ch.Enabled, ch.Config, time.Now().Format(time.RFC3339), ch.ID}
}

func (d *notifyDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM notify_channels WHERE id = ?", []any{id}
}

func (d *notifyDialect) SelectByID(id int64) (string, []any) {
	return "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels WHERE id = ?", []any{id}
}

func (d *notifyDialect) SelectByType(channelType string) (string, []any) {
	return "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels WHERE type = ?", []any{channelType}
}

func (d *notifyDialect) SelectAll() (string, []any) {
	return "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels", nil
}

func (d *notifyDialect) SelectEnabled() (string, []any) {
	return "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels WHERE enabled = 1", nil
}

func (d *notifyDialect) ScanRow(rows *sql.Rows) (*database.NotifyChannel, error) {
	var ch database.NotifyChannel
	var enabled int
	var createdAt, updatedAt string
	err := rows.Scan(
		&ch.ID, &ch.Type, &ch.Name, &enabled, &ch.Config, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	ch.Enabled = enabled == 1
	ch.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	ch.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &ch, nil
}

var _ database.NotifyDialect = (*notifyDialect)(nil)
