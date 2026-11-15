// atlhyper_master_v2/database/repo/notify_delivery.go
// NotifyDeliveryRepository 实现
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyDeliveryRepo struct {
	db      *sql.DB
	dialect database.NotifyDeliveryDialect
}

func newNotifyDeliveryRepo(db *sql.DB, dialect database.NotifyDeliveryDialect) *notifyDeliveryRepo {
	return &notifyDeliveryRepo{db: db, dialect: dialect}
}

func (r *notifyDeliveryRepo) Create(ctx context.Context, delivery *database.NotifyDelivery) error {
	query, args := r.dialect.Insert(delivery)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	delivery.ID = id
	return nil
}

func (r *notifyDeliveryRepo) UpdateBatch(ctx context.Context, batchID string, result database.NotifyDeliveryResult) error {
	query, args := r.dialect.UpdateBatch(batchID, result)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *notifyDeliveryRepo) ListUnfinished(ctx context.Context) ([]*database.NotifyDelivery, error) {
	query, args := r.dialect.SelectUnfinished()
	return queryNotifyRows(ctx, r.db, r.dialect.ScanRow, query, args...)
}

func (r *notifyDeliveryRepo) List(ctx context.Context, opts database.NotifyDeliveryQueryOpts) ([]*database.NotifyDelivery, error) {
	query, args := r.dialect.List(opts)
	return queryNotifyRows(ctx, r.db, r.dialect.ScanRow, query, args...)
}

func (r *notifyDeliveryRepo) Count(ctx context.Context, opts database.NotifyDeliveryQueryOpts) (int, error) {
	query, args := r.dialect.Count(opts)
	return queryNotifyCount(ctx, r.db, query, args...)
}

func (r *notifyDeliveryRepo) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteBefore(before)
	return execNotifyAffected(ctx, r.db, query, args...)
}

var _ database.NotifyDeliveryRepository = (*notifyDeliveryRepo)(nil)
