// atlhyper_master_v2/database/event.go
// ClusterEventRepository 实现
package database

import (
	"context"
	"database/sql"
	"time"
)

type eventRepo struct {
	db      *sql.DB
	dialect EventDialect
}

func newEventRepo(db *sql.DB, dialect EventDialect) *eventRepo {
	return &eventRepo{db: db, dialect: dialect}
}

func (r *eventRepo) Upsert(ctx context.Context, event *ClusterEvent) error {
	query, args := r.dialect.Upsert(event)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *eventRepo) UpsertBatch(ctx context.Context, events []*ClusterEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 使用第一个事件获取 SQL 模板（所有 Upsert 使用相同模板）
	queryTemplate, _ := r.dialect.Upsert(events[0])
	stmt, err := tx.PrepareContext(ctx, queryTemplate)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, event := range events {
		_, args := r.dialect.Upsert(event)
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *eventRepo) ListByCluster(ctx context.Context, clusterID string, opts EventQueryOpts) ([]*ClusterEvent, error) {
	query, args := r.dialect.ListByCluster(clusterID, opts)
	return r.queryEvents(ctx, query, args...)
}

func (r *eventRepo) ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*ClusterEvent, error) {
	query, args := r.dialect.ListByInvolvedResource(clusterID, kind, namespace, name)
	return r.queryEvents(ctx, query, args...)
}

func (r *eventRepo) ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*ClusterEvent, error) {
	query, args := r.dialect.ListByType(clusterID, eventType, since)
	return r.queryEvents(ctx, query, args...)
}

func (r *eventRepo) DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteBefore(clusterID, before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *eventRepo) DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error) {
	// 先统计总数
	countQuery, countArgs := r.dialect.CountByCluster(clusterID)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return 0, err
	}

	if total <= int64(keepCount) {
		return 0, nil
	}

	deleteCount := total - int64(keepCount)
	// 使用子查询删除最旧的记录
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

func (r *eventRepo) CountByCluster(ctx context.Context, clusterID string) (int64, error) {
	query, args := r.dialect.CountByCluster(clusterID)
	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *eventRepo) CountByHour(ctx context.Context, clusterID string, hours int) ([]HourlyEventCount, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	query, args := r.dialect.CountByHour(clusterID, since)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HourlyEventCount
	for rows.Next() {
		h, err := r.dialect.ScanHourlyCount(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *h)
	}
	return results, rows.Err()
}

func (r *eventRepo) CountByHourAndKind(ctx context.Context, clusterID string, hours int) ([]HourlyKindCount, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	query, args := r.dialect.CountByHourAndKind(clusterID, since)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HourlyKindCount
	for rows.Next() {
		h, err := r.dialect.ScanHourlyKindCount(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *h)
	}
	return results, rows.Err()
}

func (r *eventRepo) queryEvents(ctx context.Context, query string, args ...any) ([]*ClusterEvent, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*ClusterEvent
	for rows.Next() {
		e, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
