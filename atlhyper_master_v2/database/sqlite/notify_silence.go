// atlhyper_master_v2/database/sqlite/notify_silence.go
// SQLite NotifySilenceDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifySilenceDialect struct{}

const notifySilenceColumns = "id, cluster_id, namespace, subject, starts_at, ends_at, comment, created_by, created_at"

func (d *notifySilenceDialect) Insert(silence *database.NotifySilence, now time.Time) (string, []any) {
	return `INSERT INTO notify_silences (cluster_id, namespace, subject, starts_at, ends_at, comment, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{silence.ClusterID, silence.Namespace, silence.Subject, utcStamp(silence.StartsAt), utcStamp(silence.EndsAt),
			silence.Comment, silence.CreatedBy, utcStamp(now)}
}

func (d *notifySilenceDialect) UpdateEndsAt(id int64, endsAt time.Time) (string, []any) {
	return "UPDATE notify_silences SET ends_at = ? WHERE id = ?", []any{utcStamp(endsAt), id}
}

func (d *notifySilenceDialect) SelectByID(id int64) (string, []any) {
	return "SELECT " + notifySilenceColumns + " FROM notify_silences WHERE id = ?", []any{id}
}

func (d *notifySilenceDialect) SelectAll() (string, []any) {
	return "SELECT " + notifySilenceColumns + " FROM notify_silences ORDER BY starts_at DESC, id DESC", nil
}

func (d *notifySilenceDialect) DeleteEndedBefore(before time.Time) (string, []any) {
	return "DELETE FROM notify_silences WHERE ends_at < ?", []any{utcStamp(before)}
}

func (d *notifySilenceDialect) ScanRow(rows *sql.Rows) (*database.NotifySilence, error) {
	silence := &database.NotifySilence{}
	var startsAt, endsAt, createdAt string
	err := rows.Scan(&silence.ID, &silence.ClusterID, &silence.Namespace, &silence.Subject,
		&startsAt, &endsAt, &silence.Comment, &silence.CreatedBy, &createdAt)
	if err != nil {
		return nil, err
	}
	silence.StartsAt = parseStamp(startsAt)
	silence.EndsAt = parseStamp(endsAt)
	silence.CreatedAt = parseStamp(createdAt)
	return silence, nil
}

var _ database.NotifySilenceDialect = (*notifySilenceDialect)(nil)
