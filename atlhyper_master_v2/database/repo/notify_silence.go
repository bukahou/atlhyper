// atlhyper_master_v2/database/repo/notify_silence.go
// NotifySilenceRepository 实现
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifySilenceRepo struct {
	db      *sql.DB
	dialect database.NotifySilenceDialect
}

func newNotifySilenceRepo(db *sql.DB, dialect database.NotifySilenceDialect) *notifySilenceRepo {
	return &notifySilenceRepo{db: db, dialect: dialect}
}

func (r *notifySilenceRepo) Create(ctx context.Context, silence *database.NotifySilence) error {
	now := time.Now()
	query, args := r.dialect.Insert(silence, now)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	silence.ID = id
	silence.CreatedAt = now.UTC().Truncate(time.Second)
	return nil
}

func (r *notifySilenceRepo) GetByID(ctx context.Context, id int64) (*database.NotifySilence, error) {
	query, args := r.dialect.SelectByID(id)
	silence, found, err := queryNotifyRow(ctx, r.db, r.dialect.ScanRow, query, args...)
	if err != nil || !found {
		return nil, err
	}
	return silence, nil
}

func (r *notifySilenceRepo) List(ctx context.Context) ([]*database.NotifySilence, error) {
	query, args := r.dialect.SelectAll()
	return queryNotifyRows(ctx, r.db, r.dialect.ScanRow, query, args...)
}

func (r *notifySilenceRepo) SetEndsAt(ctx context.Context, id int64, endsAt time.Time) error {
	query, args := r.dialect.UpdateEndsAt(id, endsAt)
	affected, err := execNotifyAffected(ctx, r.db, query, args...)
	if err != nil {
		return err
	}
	if affected == 0 {
		return database.ErrNotifySilenceNotFound
	}
	return nil
}

func (r *notifySilenceRepo) DeleteEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteEndedBefore(before)
	return execNotifyAffected(ctx, r.db, query, args...)
}

var _ database.NotifySilenceRepository = (*notifySilenceRepo)(nil)
