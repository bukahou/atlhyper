// atlhyper_master_v2/database/notify.go
// NotifyChannelRepository 实现
package database

import (
	"context"
	"database/sql"
)

type notifyRepo struct {
	db      *sql.DB
	dialect NotifyDialect
}

func newNotifyRepo(db *sql.DB, dialect NotifyDialect) *notifyRepo {
	return &notifyRepo{db: db, dialect: dialect}
}

func (r *notifyRepo) Create(ctx context.Context, ch *NotifyChannel) error {
	query, args := r.dialect.Insert(ch)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	ch.ID = id
	return nil
}

func (r *notifyRepo) Update(ctx context.Context, ch *NotifyChannel) error {
	query, args := r.dialect.Update(ch)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *notifyRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *notifyRepo) GetByID(ctx context.Context, id int64) (*NotifyChannel, error) {
	query, args := r.dialect.SelectByID(id)
	return r.queryOne(ctx, query, args...)
}

func (r *notifyRepo) GetByType(ctx context.Context, channelType string) (*NotifyChannel, error) {
	query, args := r.dialect.SelectByType(channelType)
	return r.queryOne(ctx, query, args...)
}

func (r *notifyRepo) List(ctx context.Context) ([]*NotifyChannel, error) {
	query, args := r.dialect.SelectAll()
	return r.queryAll(ctx, query, args...)
}

func (r *notifyRepo) ListEnabled(ctx context.Context) ([]*NotifyChannel, error) {
	query, args := r.dialect.SelectEnabled()
	return r.queryAll(ctx, query, args...)
}

func (r *notifyRepo) queryOne(ctx context.Context, query string, args ...any) (*NotifyChannel, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	return r.dialect.ScanRow(rows)
}

func (r *notifyRepo) queryAll(ctx context.Context, query string, args ...any) ([]*NotifyChannel, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*NotifyChannel
	for rows.Next() {
		ch, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}
