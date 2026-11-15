// atlhyper_master_v2/database/sqlite/notify_alert.go
// SQLite NotifyAlertDialect 实现
package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyAlertDialect struct{}

const notifyAlertColumns = `fingerprint, source, cluster_id, severity, status, kind, title, summary, labels, details,
	starts_at, ends_at, updated_at, last_notified_at, last_notified_kind, last_notified_severity,
	notify_count, suppressed_by, silenced_by`

func (d *notifyAlertDialect) Upsert(alert *database.NotifyAlert) (string, []any) {
	query := `INSERT INTO alerts (` + notifyAlertColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(fingerprint) DO UPDATE SET
			source = excluded.source, cluster_id = excluded.cluster_id, severity = excluded.severity,
			status = excluded.status, kind = excluded.kind, title = excluded.title, summary = excluded.summary,
			labels = excluded.labels, details = excluded.details,
			starts_at = excluded.starts_at, ends_at = excluded.ends_at, updated_at = excluded.updated_at,
			last_notified_at = excluded.last_notified_at, last_notified_kind = excluded.last_notified_kind,
			last_notified_severity = excluded.last_notified_severity, notify_count = excluded.notify_count,
			suppressed_by = excluded.suppressed_by, silenced_by = excluded.silenced_by`
	labels, details := alert.Labels, alert.Details
	if labels == nil {
		labels = map[string]string{}
	}
	if details == nil {
		details = map[string]string{}
	}
	return query, []any{
		alert.Fingerprint, alert.Source, alert.ClusterID, alert.Severity, alert.Status, alert.Kind,
		alert.Title, alert.Summary, encodeJSON(labels), encodeJSON(details),
		utcStamp(alert.StartsAt), nullableStamp(alert.EndsAt), utcStamp(alert.UpdatedAt),
		nullableStamp(alert.LastNotifiedAt), alert.LastNotifiedKind, alert.LastNotifiedSeverity,
		alert.NotifyCount, alert.SuppressedBy, alert.SilencedBy,
	}
}

func (d *notifyAlertDialect) SelectByFingerprint(fingerprint string) (string, []any) {
	return "SELECT " + notifyAlertColumns + " FROM alerts WHERE fingerprint = ?", []any{fingerprint}
}

func (d *notifyAlertDialect) SelectFiring() (string, []any) {
	return "SELECT " + notifyAlertColumns + " FROM alerts WHERE status = 'firing' ORDER BY starts_at, fingerprint", nil
}

func (d *notifyAlertDialect) List(opts database.NotifyAlertQueryOpts) (string, []any) {
	where, args := notifyAlertWhere(opts)
	page, pageArgs := pageClause(opts.Limit, opts.Offset)
	return "SELECT " + notifyAlertColumns + " FROM alerts" + where + " ORDER BY starts_at DESC, fingerprint" + page,
		append(args, pageArgs...)
}

func (d *notifyAlertDialect) Count(opts database.NotifyAlertQueryOpts) (string, []any) {
	where, args := notifyAlertWhere(opts)
	return "SELECT COUNT(*) FROM alerts" + where, args
}

func notifyAlertWhere(opts database.NotifyAlertQueryOpts) (string, []any) {
	var conditions []string
	var args []any
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.ClusterID != "" {
		conditions = append(conditions, "cluster_id = ?")
		args = append(args, opts.ClusterID)
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (d *notifyAlertDialect) DeleteResolvedBefore(before time.Time) (string, []any) {
	return "DELETE FROM alerts WHERE status = 'resolved' AND ends_at IS NOT NULL AND ends_at < ?", []any{utcStamp(before)}
}

func (d *notifyAlertDialect) ScanRow(rows *sql.Rows) (*database.NotifyAlert, error) {
	alert := &database.NotifyAlert{}
	var labels, details, startsAt, updatedAt string
	var endsAt, lastNotifiedAt sql.NullString
	err := rows.Scan(&alert.Fingerprint, &alert.Source, &alert.ClusterID, &alert.Severity, &alert.Status, &alert.Kind,
		&alert.Title, &alert.Summary, &labels, &details,
		&startsAt, &endsAt, &updatedAt, &lastNotifiedAt, &alert.LastNotifiedKind, &alert.LastNotifiedSeverity,
		&alert.NotifyCount, &alert.SuppressedBy, &alert.SilencedBy)
	if err != nil {
		return nil, err
	}
	alert.Labels = decodeStringMap(labels)
	alert.Details = decodeStringMap(details)
	alert.StartsAt = parseStamp(startsAt)
	alert.EndsAt = parseNullableStamp(endsAt)
	alert.UpdatedAt = parseStamp(updatedAt)
	alert.LastNotifiedAt = parseNullableStamp(lastNotifiedAt)
	return alert, nil
}

var _ database.NotifyAlertDialect = (*notifyAlertDialect)(nil)
