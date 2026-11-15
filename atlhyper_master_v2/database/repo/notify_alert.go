// atlhyper_master_v2/database/repo/notify_alert.go
// NotifyAlertRepository 实现
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyAlertRepo struct {
	db      *sql.DB
	dialect database.NotifyAlertDialect
}

func newNotifyAlertRepo(db *sql.DB, dialect database.NotifyAlertDialect) *notifyAlertRepo {
	return &notifyAlertRepo{db: db, dialect: dialect}
}

func (r *notifyAlertRepo) Upsert(ctx context.Context, alert *database.NotifyAlert) error {
	query, args := r.dialect.Upsert(alert)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *notifyAlertRepo) Get(ctx context.Context, fingerprint string) (*database.NotifyAlert, error) {
	query, args := r.dialect.SelectByFingerprint(fingerprint)
	alert, found, err := queryNotifyRow(ctx, r.db, r.dialect.ScanRow, query, args...)
	if err != nil || !found {
		return nil, err
	}
	return alert, nil
}

func (r *notifyAlertRepo) ListFiring(ctx context.Context) ([]*database.NotifyAlert, error) {
	query, args := r.dialect.SelectFiring()
	return queryNotifyRows(ctx, r.db, r.dialect.ScanRow, query, args...)
}

func (r *notifyAlertRepo) List(ctx context.Context, opts database.NotifyAlertQueryOpts) ([]*database.NotifyAlert, error) {
	query, args := r.dialect.List(opts)
	return queryNotifyRows(ctx, r.db, r.dialect.ScanRow, query, args...)
}

func (r *notifyAlertRepo) Count(ctx context.Context, opts database.NotifyAlertQueryOpts) (int, error) {
	query, args := r.dialect.Count(opts)
	return queryNotifyCount(ctx, r.db, query, args...)
}

func (r *notifyAlertRepo) DeleteResolvedBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteResolvedBefore(before)
	return execNotifyAffected(ctx, r.db, query, args...)
}

var _ database.NotifyAlertRepository = (*notifyAlertRepo)(nil)
