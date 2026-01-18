// atlhyper_master_v2/database/sqlite/impl/cluster_event.go
// ClusterEventRepository SQLite 实现
package impl

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

// ClusterEventRepository SQLite 实现
type ClusterEventRepository struct {
	db *sql.DB
}

// NewClusterEventRepository 创建实例
func NewClusterEventRepository(db *sql.DB) *ClusterEventRepository {
	return &ClusterEventRepository{db: db}
}

// Upsert 插入或更新事件（基于 dedup_key 去重）
func (r *ClusterEventRepository) Upsert(ctx context.Context, event *repository.ClusterEvent) error {
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
	_, err := r.db.ExecContext(ctx, query,
		event.DedupKey, event.ClusterID, event.Namespace, event.Name,
		event.Type, event.Reason, event.Message,
		event.SourceComponent, event.SourceHost,
		event.InvolvedKind, event.InvolvedName, event.InvolvedNamespace,
		event.FirstTimestamp.Format(time.RFC3339),
		event.LastTimestamp.Format(time.RFC3339),
		event.Count, now, now,
	)
	return err
}

// UpsertBatch 批量插入或更新
func (r *ClusterEventRepository) UpsertBatch(ctx context.Context, events []*repository.ClusterEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Format(time.RFC3339)
	for _, event := range events {
		_, err := stmt.ExecContext(ctx,
			event.DedupKey, event.ClusterID, event.Namespace, event.Name,
			event.Type, event.Reason, event.Message,
			event.SourceComponent, event.SourceHost,
			event.InvolvedKind, event.InvolvedName, event.InvolvedNamespace,
			event.FirstTimestamp.Format(time.RFC3339),
			event.LastTimestamp.Format(time.RFC3339),
			event.Count, now, now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListByCluster 按集群查询
func (r *ClusterEventRepository) ListByCluster(ctx context.Context, clusterID string, opts repository.EventQueryOpts) ([]*repository.ClusterEvent, error) {
	query := `
		SELECT id, dedup_key, cluster_id, namespace, name, type, reason, message,
			source_component, source_host,
			involved_kind, involved_name, involved_namespace,
			first_timestamp, last_timestamp, count, created_at, updated_at
		FROM cluster_events
		WHERE cluster_id = ?
	`
	args := []interface{}{clusterID}

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

	return r.queryEvents(ctx, query, args...)
}

// ListByInvolvedResource 按关联资源查询
func (r *ClusterEventRepository) ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*repository.ClusterEvent, error) {
	query := `
		SELECT id, dedup_key, cluster_id, namespace, name, type, reason, message,
			source_component, source_host,
			involved_kind, involved_name, involved_namespace,
			first_timestamp, last_timestamp, count, created_at, updated_at
		FROM cluster_events
		WHERE cluster_id = ? AND involved_kind = ? AND involved_namespace = ? AND involved_name = ?
		ORDER BY last_timestamp DESC
	`
	return r.queryEvents(ctx, query, clusterID, kind, namespace, name)
}

// ListByType 按类型查询
func (r *ClusterEventRepository) ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*repository.ClusterEvent, error) {
	query := `
		SELECT id, dedup_key, cluster_id, namespace, name, type, reason, message,
			source_component, source_host,
			involved_kind, involved_name, involved_namespace,
			first_timestamp, last_timestamp, count, created_at, updated_at
		FROM cluster_events
		WHERE cluster_id = ? AND type = ? AND last_timestamp >= ?
		ORDER BY last_timestamp DESC
	`
	return r.queryEvents(ctx, query, clusterID, eventType, since.Format(time.RFC3339))
}

// DeleteBefore 删除指定时间之前的事件
func (r *ClusterEventRepository) DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM cluster_events WHERE cluster_id = ? AND last_timestamp < ?",
		clusterID, before.Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteOldest 删除最旧的事件
func (r *ClusterEventRepository) DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error) {
	// 先统计总数
	var total int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM cluster_events WHERE cluster_id = ?",
		clusterID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}

	if total <= int64(keepCount) {
		return 0, nil
	}

	deleteCount := total - int64(keepCount)
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM cluster_events WHERE id IN (
			SELECT id FROM cluster_events WHERE cluster_id = ?
			ORDER BY last_timestamp ASC LIMIT ?
		)
	`, clusterID, deleteCount)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountByCluster 统计集群事件数
func (r *ClusterEventRepository) CountByCluster(ctx context.Context, clusterID string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM cluster_events WHERE cluster_id = ?",
		clusterID,
	).Scan(&count)
	return count, err
}

// CountByHour 按小时统计事件数
func (r *ClusterEventRepository) CountByHour(ctx context.Context, clusterID string, hours int) ([]repository.HourlyEventCount, error) {
	// 计算起始时间
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

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

	rows, err := r.db.QueryContext(ctx, query, clusterID, since.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []repository.HourlyEventCount
	for rows.Next() {
		var h repository.HourlyEventCount
		if err := rows.Scan(&h.Hour, &h.WarningCount, &h.NormalCount); err != nil {
			return nil, err
		}
		results = append(results, h)
	}
	return results, rows.Err()
}

// CountByHourAndKind 按小时和资源类型统计
func (r *ClusterEventRepository) CountByHourAndKind(ctx context.Context, clusterID string, hours int) ([]repository.HourlyKindCount, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

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

	rows, err := r.db.QueryContext(ctx, query, clusterID, since.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []repository.HourlyKindCount
	for rows.Next() {
		var h repository.HourlyKindCount
		if err := rows.Scan(&h.Hour, &h.Kind, &h.Count); err != nil {
			return nil, err
		}
		results = append(results, h)
	}
	return results, rows.Err()
}

// queryEvents 通用查询
func (r *ClusterEventRepository) queryEvents(ctx context.Context, query string, args ...interface{}) ([]*repository.ClusterEvent, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*repository.ClusterEvent
	for rows.Next() {
		var e repository.ClusterEvent
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
		events = append(events, &e)
	}
	return events, rows.Err()
}

// 确保实现了接口
var _ repository.ClusterEventRepository = (*ClusterEventRepository)(nil)
