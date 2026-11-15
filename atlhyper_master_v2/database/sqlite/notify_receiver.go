// atlhyper_master_v2/database/sqlite/notify_receiver.go
// SQLite NotifyReceiverDialect 实现。config 列是密文，本层不接触明文配置
package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyReceiverDialect struct{}

const notifyReceiverColumns = "id, name, type, enabled, config, created_at, updated_at"

func (d *notifyReceiverDialect) Insert(receiver *database.NotifyReceiver, encryptedConfig string, now time.Time) (string, []any) {
	stamp := utcStamp(now)
	return `INSERT INTO notify_receivers (name, type, enabled, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		[]any{receiver.Name, receiver.Type, boolToInt(receiver.Enabled), encryptedConfig, stamp, stamp}
}

func (d *notifyReceiverDialect) Update(receiver *database.NotifyReceiver, encryptedConfig string, now time.Time) (string, []any) {
	return `UPDATE notify_receivers SET name = ?, type = ?, enabled = ?, config = ?, updated_at = ? WHERE id = ?`,
		[]any{receiver.Name, receiver.Type, boolToInt(receiver.Enabled), encryptedConfig, utcStamp(now), receiver.ID}
}

func (d *notifyReceiverDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM notify_receivers WHERE id = ?", []any{id}
}

func (d *notifyReceiverDialect) SelectByID(id int64) (string, []any) {
	return "SELECT " + notifyReceiverColumns + " FROM notify_receivers WHERE id = ?", []any{id}
}

func (d *notifyReceiverDialect) SelectAll() (string, []any) {
	return "SELECT " + notifyReceiverColumns + " FROM notify_receivers ORDER BY id", nil
}

func (d *notifyReceiverDialect) ScanRow(rows *sql.Rows) (*database.NotifyReceiver, string, error) {
	receiver := &database.NotifyReceiver{}
	var enabled int
	var encryptedConfig, createdAt, updatedAt string
	if err := rows.Scan(&receiver.ID, &receiver.Name, &receiver.Type, &enabled, &encryptedConfig, &createdAt, &updatedAt); err != nil {
		return nil, "", err
	}
	receiver.Enabled = enabled == 1
	receiver.CreatedAt = parseStamp(createdAt)
	receiver.UpdatedAt = parseStamp(updatedAt)
	return receiver, encryptedConfig, nil
}

// IsDuplicateName SQLite 的唯一约束冲突文案形如 "UNIQUE constraint failed: notify_receivers.name"
func (d *notifyReceiverDialect) IsDuplicateName(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: notify_receivers.name")
}

var _ database.NotifyReceiverDialect = (*notifyReceiverDialect)(nil)
