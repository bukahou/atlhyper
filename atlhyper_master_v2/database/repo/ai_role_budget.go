// atlhyper_master_v2/database/repo/ai_role_budget.go
// AIRoleBudget Repository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiRoleBudgetRepo struct {
	db      *sql.DB
	dialect database.AIRoleBudgetDialect
}

func newAIRoleBudgetRepo(db *sql.DB, dialect database.AIRoleBudgetDialect) *aiRoleBudgetRepo {
	return &aiRoleBudgetRepo{db: db, dialect: dialect}
}

func (r *aiRoleBudgetRepo) Get(ctx context.Context, role string) (*database.AIRoleBudget, error) {
	query, args := r.dialect.SelectByRole(role)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return r.dialect.ScanRow(rows)
	}
	return nil, nil
}

func (r *aiRoleBudgetRepo) ListAll(ctx context.Context) ([]*database.AIRoleBudget, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []*database.AIRoleBudget
	for rows.Next() {
		b, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, nil
}

func (r *aiRoleBudgetRepo) Upsert(ctx context.Context, budget *database.AIRoleBudget) error {
	query, args := r.dialect.Upsert(budget)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiRoleBudgetRepo) Delete(ctx context.Context, role string) error {
	query, args := r.dialect.Delete(role)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiRoleBudgetRepo) IncrementUsage(ctx context.Context, role string, inputTokens, outputTokens int) error {
	query, args := r.dialect.IncrementUsage(role, inputTokens, outputTokens)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiRoleBudgetRepo) ResetDailyUsage(ctx context.Context, role string) error {
	query, args := r.dialect.ResetDailyUsage(role)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiRoleBudgetRepo) ResetMonthlyUsage(ctx context.Context, role string) error {
	query, args := r.dialect.ResetMonthlyUsage(role)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

var _ database.AIRoleBudgetRepository = (*aiRoleBudgetRepo)(nil)
