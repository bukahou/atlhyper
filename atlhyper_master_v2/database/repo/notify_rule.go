// atlhyper_master_v2/database/repo/notify_rule.go
// NotifyRuleRepository 实现
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type notifyRuleRepo struct {
	db      *sql.DB
	dialect database.NotifyRuleDialect
}

func newNotifyRuleRepo(db *sql.DB, dialect database.NotifyRuleDialect) *notifyRuleRepo {
	return &notifyRuleRepo{db: db, dialect: dialect}
}

// Create 新规则排在最后。取最大位置与插入放在同一事务里，并发创建不会拿到相同的位置
func (r *notifyRuleRepo) Create(ctx context.Context, rule *database.NotifyRule) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var maxPosition int
	query, args := r.dialect.SelectMaxPosition()
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&maxPosition); err != nil {
		return err
	}

	now := time.Now()
	query, args = r.dialect.Insert(rule, maxPosition+1, now)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	rule.ID = id
	rule.Position = maxPosition + 1
	rule.CreatedAt = now.UTC().Truncate(time.Second)
	rule.UpdatedAt = rule.CreatedAt
	return nil
}

func (r *notifyRuleRepo) Update(ctx context.Context, rule *database.NotifyRule) error {
	now := time.Now()
	query, args := r.dialect.Update(rule, now)
	affected, err := execNotifyAffected(ctx, r.db, query, args...)
	if err != nil {
		return err
	}
	if affected == 0 {
		return database.ErrNotifyRuleNotFound
	}
	rule.UpdatedAt = now.UTC().Truncate(time.Second)
	return nil
}

func (r *notifyRuleRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	affected, err := execNotifyAffected(ctx, r.db, query, args...)
	if err != nil {
		return err
	}
	if affected == 0 {
		return database.ErrNotifyRuleNotFound
	}
	return nil
}

func (r *notifyRuleRepo) GetByID(ctx context.Context, id int64) (*database.NotifyRule, error) {
	query, args := r.dialect.SelectByID(id)
	rule, found, err := queryNotifyRow(ctx, r.db, r.dialect.ScanRow, query, args...)
	if err != nil || !found {
		return nil, err
	}
	return rule, nil
}

func (r *notifyRuleRepo) List(ctx context.Context) ([]*database.NotifyRule, error) {
	query, args := r.dialect.SelectAll()
	return queryNotifyRows(ctx, r.db, r.dialect.ScanRow, query, args...)
}

// Reorder 按给定顺序把位置改写为 1、2、3…。校验与改写在同一事务里：
// 校验通过之后、改写完成之前，规则集合不会被别的写入改变
func (r *notifyRuleRepo) Reorder(ctx context.Context, ids []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	existing := map[int64]bool{}
	rows, err := tx.QueryContext(ctx, "SELECT id FROM notify_rules")
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if len(ids) != len(existing) {
		return database.ErrNotifyRuleOrderMismatch
	}
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if !existing[id] || seen[id] {
			return database.ErrNotifyRuleOrderMismatch
		}
		seen[id] = true
	}

	for index, id := range ids {
		query, args := r.dialect.UpdatePosition(id, index+1)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

var _ database.NotifyRuleRepository = (*notifyRuleRepo)(nil)
