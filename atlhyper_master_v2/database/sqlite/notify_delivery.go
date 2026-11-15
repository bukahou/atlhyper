// atlhyper_master_v2/database/sqlite/notify_delivery.go
// SQLite NotifyDeliveryDialect 实现
package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyDeliveryDialect struct{}

const notifyDeliveryColumns = `id, batch_id, fingerprint, alert_title, receiver_id, receiver_name, receiver_type,
	kind, status, attempts, last_error, next_attempt_at, created_at, updated_at`

func (d *notifyDeliveryDialect) Insert(delivery *database.NotifyDelivery) (string, []any) {
	query := `INSERT INTO notify_deliveries (batch_id, fingerprint, alert_title, receiver_id, receiver_name, receiver_type,
		kind, status, attempts, last_error, next_attempt_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return query, []any{
		delivery.BatchID, delivery.Fingerprint, delivery.AlertTitle,
		delivery.ReceiverID, delivery.ReceiverName, delivery.ReceiverType,
		delivery.Kind, delivery.Status, delivery.Attempts, delivery.LastError,
		nullableStamp(delivery.NextAttemptAt), utcStamp(delivery.CreatedAt), utcStamp(delivery.UpdatedAt),
	}
}

func (d *notifyDeliveryDialect) UpdateBatch(batchID string, result database.NotifyDeliveryResult) (string, []any) {
	return `UPDATE notify_deliveries SET status = ?, attempts = ?, last_error = ?, next_attempt_at = ?, updated_at = ?
		WHERE batch_id = ?`,
		[]any{result.Status, result.Attempts, result.LastError, nullableStamp(result.NextAttemptAt), utcStamp(result.UpdatedAt), batchID}
}

func (d *notifyDeliveryDialect) SelectUnfinished() (string, []any) {
	return "SELECT " + notifyDeliveryColumns + " FROM notify_deliveries WHERE status IN (?, ?) ORDER BY id",
		[]any{database.NotifyDeliveryPending, database.NotifyDeliveryFailed}
}

func (d *notifyDeliveryDialect) List(opts database.NotifyDeliveryQueryOpts) (string, []any) {
	where, args := notifyDeliveryWhere(opts)
	page, pageArgs := pageClause(opts.Limit, opts.Offset)
	return "SELECT " + notifyDeliveryColumns + " FROM notify_deliveries" + where + " ORDER BY created_at DESC, id DESC" + page,
		append(args, pageArgs...)
}

func (d *notifyDeliveryDialect) Count(opts database.NotifyDeliveryQueryOpts) (string, []any) {
	where, args := notifyDeliveryWhere(opts)
	return "SELECT COUNT(*) FROM notify_deliveries" + where, args
}

func notifyDeliveryWhere(opts database.NotifyDeliveryQueryOpts) (string, []any) {
	var conditions []string
	var args []any
	if opts.Fingerprint != "" {
		conditions = append(conditions, "fingerprint = ?")
		args = append(args, opts.Fingerprint)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (d *notifyDeliveryDialect) DeleteBefore(before time.Time) (string, []any) {
	return "DELETE FROM notify_deliveries WHERE created_at < ?", []any{utcStamp(before)}
}

func (d *notifyDeliveryDialect) ScanRow(rows *sql.Rows) (*database.NotifyDelivery, error) {
	delivery := &database.NotifyDelivery{}
	var nextAttemptAt sql.NullString
	var createdAt, updatedAt string
	err := rows.Scan(&delivery.ID, &delivery.BatchID, &delivery.Fingerprint, &delivery.AlertTitle,
		&delivery.ReceiverID, &delivery.ReceiverName, &delivery.ReceiverType,
		&delivery.Kind, &delivery.Status, &delivery.Attempts, &delivery.LastError,
		&nextAttemptAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	delivery.NextAttemptAt = parseNullableStamp(nextAttemptAt)
	delivery.CreatedAt = parseStamp(createdAt)
	delivery.UpdatedAt = parseStamp(updatedAt)
	return delivery, nil
}

var _ database.NotifyDeliveryDialect = (*notifyDeliveryDialect)(nil)
