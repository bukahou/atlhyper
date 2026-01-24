// atlhyper_master_v2/database/repo/notify.go
// NotifyChannelRepository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyRepo struct {
	db      *sql.DB
	dialect database.NotifyDialect
}

func newNotifyRepo(db *sql.DB, dialect database.NotifyDialect) *notifyRepo {
	return &notifyRepo{db: db, dialect: dialect}
}

func (r *notifyRepo) Create(ctx context.Context, ch *database.NotifyChannel) error {
	query, args := r.dialect.Insert(ch)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	ch.ID = id
	return nil
}

func (r *notifyRepo) Update(ctx context.Context, ch *database.NotifyChannel) error {
	query, args := r.dialect.Update(ch)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *notifyRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *notifyRepo) GetByID(ctx context.Context, id int64) (*database.NotifyChannel, error) {
	query, args := r.dialect.SelectByID(id)
	return r.queryOne(ctx, query, args...)
}

func (r *notifyRepo) GetByType(ctx context.Context, channelType string) (*database.NotifyChannel, error) {
	query, args := r.dialect.SelectByType(channelType)
	return r.queryOne(ctx, query, args...)
}

func (r *notifyRepo) List(ctx context.Context) ([]*database.NotifyChannel, error) {
	query, args := r.dialect.SelectAll()
	return r.queryAll(ctx, query, args...)
}

func (r *notifyRepo) ListEnabled(ctx context.Context) ([]*database.NotifyChannel, error) {
	query, args := r.dialect.SelectEnabled()
	return r.queryAll(ctx, query, args...)
}

func (r *notifyRepo) queryOne(ctx context.Context, query string, args ...any) (*database.NotifyChannel, error) {
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

func (r *notifyRepo) queryAll(ctx context.Context, query string, args ...any) ([]*database.NotifyChannel, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*database.NotifyChannel
	for rows.Next() {
		ch, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}
