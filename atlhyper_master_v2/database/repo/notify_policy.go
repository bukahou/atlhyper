// atlhyper_master_v2/database/repo/notify_policy.go
// NotifyPolicyRepository 实现（全局通知策略，单行）
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyPolicyRepo struct {
	db      *sql.DB
	dialect database.NotifyPolicyDialect
}

func newNotifyPolicyRepo(db *sql.DB, dialect database.NotifyPolicyDialect) *notifyPolicyRepo {
	return &notifyPolicyRepo{db: db, dialect: dialect}
}

func (r *notifyPolicyRepo) Get(ctx context.Context) (*database.NotifyPolicy, error) {
	query, args := r.dialect.Select()
	policy, found, err := queryNotifyRow(ctx, r.db, r.dialect.ScanRow, query, args...)
	if err != nil {
		return nil, err
	}
	if !found {
		return database.DefaultNotifyPolicy(), nil
	}
	return policy, nil
}

func (r *notifyPolicyRepo) Save(ctx context.Context, policy *database.NotifyPolicy) error {
	now := time.Now()
	query, args := r.dialect.Upsert(policy, now)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	policy.UpdatedAt = now.UTC().Truncate(time.Second)
	return nil
}

var _ database.NotifyPolicyRepository = (*notifyPolicyRepo)(nil)
