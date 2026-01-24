// atlhyper_master_v2/database/sqlite/event.go
// SQLite EventDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type eventDialect struct{}

func (d *eventDialect) Upsert(event *database.ClusterEvent) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	query := `
		INSERT INTO cluster_events (
			dedup_key, cluster_id, namespace, name, type, reason, message,
			source_component, source_host,
			involved_kind, involved_name, involved_namespace,
			first_timestamp, last_timestamp, count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(dedup_key) DO UPDATE SET
			count = cluster_events.count + excluded.count,
			last_timestamp = excluded.last_timestamp,
			message = excluded.message,
			updated_at = excluded.updated_at
	`
	args := []any{
		event.DedupKey, event.ClusterID, event.Namespace, event.Name,
		event.Type, event.Reason, event.Message,
		event.SourceComponent, event.SourceHost,
		event.InvolvedKind, event.InvolvedName, event.InvolvedNamespace,
		event.FirstTimestamp.Format(time.RFC3339),
		event.LastTimestamp.Format(time.RFC3339),
		event.Count, now, now,
	}
	return query, args
}

func (d *eventDialect) ListByCluster(clusterID string, opts database.EventQueryOpts) (string, []any) {
	query := `
		SELECT id, dedup_key, cluster_id, namespace, name, type, reason, message,
			source_component, source_host,
			involved_kind, involved_name, involved_namespace,
			first_timestamp, last_timestamp, count, created_at, updated_at
		FROM cluster_events
		WHERE cluster_id = ?
	`
	args := []any{clusterID}

	if opts.Type != "" {
		query += " AND type = ?"
		args = append(args, opts.Type)
	}
	if opts.Reason != "" {
		query += " AND reason = ?"
		args = append(args, opts.Reason)
	}
	if !opts.Since.IsZero() {
		query += " AND last_timestamp >= ?"
		args = append(args, opts.Since.Format(time.RFC3339))
	}
	if !opts.Until.IsZero() {
		query += " AND last_timestamp <= ?"
		args = append(args, opts.Until.Format(time.RFC3339))
	}

	query += " ORDER BY last_timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	return query, args
}

func (d *eventDialect) ListByInvolvedResource(clusterID, kind, namespace, name string) (string, []any) {
	query := `
		SELECT id, dedup_key, cluster_id, namespace, name, type, reason, message,
			source_component, source_host,
			involved_kind, involved_name, involved_namespace,
			first_timestamp, last_timestamp, count, created_at, updated_at
		FROM cluster_events
		WHERE cluster_id = ? AND involved_kind = ? AND involved_namespace = ? AND involved_name = ?
		ORDER BY last_timestamp DESC
	`
	return query, []any{clusterID, kind, namespace, name}
}

func (d *eventDialect) ListByType(clusterID, eventType string, since time.Time) (string, []any) {
	query := `
		SELECT id, dedup_key, cluster_id, namespace, name, type, reason, message,
			source_component, source_host,
			involved_kind, involved_name, involved_namespace,
			first_timestamp, last_timestamp, count, created_at, updated_at
		FROM cluster_events
		WHERE cluster_id = ? AND type = ? AND last_timestamp >= ?
		ORDER BY last_timestamp DESC
	`
	return query, []any{clusterID, eventType, since.Format(time.RFC3339)}
}

func (d *eventDialect) DeleteBefore(clusterID string, before time.Time) (string, []any) {
	return "DELETE FROM cluster_events WHERE cluster_id = ? AND last_timestamp < ?",
		[]any{clusterID, before.Format(time.RFC3339)}
}

func (d *eventDialect) CountByCluster(clusterID string) (string, []any) {
	return "SELECT COUNT(*) FROM cluster_events WHERE cluster_id = ?", []any{clusterID}
}

func (d *eventDialect) CountByHour(clusterID string, since time.Time) (string, []any) {
	query := `
		SELECT
			strftime('%Y-%m-%dT%H', last_timestamp) as hour,
			SUM(CASE WHEN type = 'Warning' THEN 1 ELSE 0 END) as warning_count,
			SUM(CASE WHEN type = 'Normal' THEN 1 ELSE 0 END) as normal_count
		FROM cluster_events
		WHERE cluster_id = ? AND last_timestamp >= ?
		GROUP BY hour
		ORDER BY hour ASC
	`
	return query, []any{clusterID, since.Format(time.RFC3339)}
}

func (d *eventDialect) CountByHourAndKind(clusterID string, since time.Time) (string, []any) {
	query := `
		SELECT
			strftime('%Y-%m-%dT%H', last_timestamp) as hour,
			involved_kind,
			COUNT(*) as count
		FROM cluster_events
		WHERE cluster_id = ? AND last_timestamp >= ?
		GROUP BY hour, involved_kind
		ORDER BY hour ASC, count DESC
	`
	return query, []any{clusterID, since.Format(time.RFC3339)}
}

func (d *eventDialect) ScanRow(rows *sql.Rows) (*database.ClusterEvent, error) {
	var e database.ClusterEvent
	var firstTS, lastTS, createdAt, updatedAt string
	err := rows.Scan(
		&e.ID, &e.DedupKey, &e.ClusterID, &e.Namespace, &e.Name,
		&e.Type, &e.Reason, &e.Message,
		&e.SourceComponent, &e.SourceHost,
		&e.InvolvedKind, &e.InvolvedName, &e.InvolvedNamespace,
		&firstTS, &lastTS, &e.Count, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.FirstTimestamp, _ = time.Parse(time.RFC3339, firstTS)
	e.LastTimestamp, _ = time.Parse(time.RFC3339, lastTS)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &e, nil
}

func (d *eventDialect) ScanHourlyCount(rows *sql.Rows) (*database.HourlyEventCount, error) {
	var h database.HourlyEventCount
	if err := rows.Scan(&h.Hour, &h.WarningCount, &h.NormalCount); err != nil {
		return nil, err
	}
	return &h, nil
}

func (d *eventDialect) ScanHourlyKindCount(rows *sql.Rows) (*database.HourlyKindCount, error) {
	var h database.HourlyKindCount
	if err := rows.Scan(&h.Hour, &h.Kind, &h.Count); err != nil {
		return nil, err
	}
	return &h, nil
}

var _ database.EventDialect = (*eventDialect)(nil)
