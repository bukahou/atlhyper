// atlhyper_master_v2/database/sqlite/ai_role_budget.go
// AIRoleBudget SQLite Dialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiRoleBudgetDialect struct{}

func (d *aiRoleBudgetDialect) Upsert(b *database.AIRoleBudget) (string, []any) {
	query := `INSERT OR REPLACE INTO ai_role_budget
		(role, daily_token_limit, daily_call_limit, fallback_provider_id,
		 auto_trigger_min_severity, daily_tokens_used, daily_calls_used, daily_reset_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	var resetAt sql.NullString
	if b.DailyResetAt != nil {
		resetAt = sql.NullString{String: b.DailyResetAt.Format(time.RFC3339), Valid: true}
	}
	args := []any{
		b.Role, b.DailyTokenLimit, b.DailyCallLimit, b.FallbackProviderID,
		b.AutoTriggerMinSeverity, b.DailyTokensUsed, b.DailyCallsUsed,
		resetAt, b.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *aiRoleBudgetDialect) SelectByRole(role string) (string, []any) {
	return `SELECT role, daily_token_limit, daily_call_limit, fallback_provider_id,
		auto_trigger_min_severity, daily_tokens_used, daily_calls_used, daily_reset_at, updated_at
		FROM ai_role_budget WHERE role = ?`, []any{role}
}

func (d *aiRoleBudgetDialect) Delete(role string) (string, []any) {
	return `DELETE FROM ai_role_budget WHERE role = ?`, []any{role}
}

func (d *aiRoleBudgetDialect) IncrementUsage(role string, tokens int) (string, []any) {
	return `UPDATE ai_role_budget SET daily_tokens_used = daily_tokens_used + ?, daily_calls_used = daily_calls_used + 1, updated_at = ? WHERE role = ?`,
		[]any{tokens, time.Now().Format(time.RFC3339), role}
}

func (d *aiRoleBudgetDialect) ResetDailyUsage(role string) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `UPDATE ai_role_budget SET daily_tokens_used = 0, daily_calls_used = 0, daily_reset_at = ?, updated_at = ? WHERE role = ?`,
		[]any{now, now, role}
}

func (d *aiRoleBudgetDialect) ScanRow(rows *sql.Rows) (*database.AIRoleBudget, error) {
	b := &database.AIRoleBudget{}
	var fallbackID sql.NullInt64
	var resetAt, updatedAt sql.NullString
	var autoTrigger sql.NullString

	err := rows.Scan(&b.Role, &b.DailyTokenLimit, &b.DailyCallLimit, &fallbackID,
		&autoTrigger, &b.DailyTokensUsed, &b.DailyCallsUsed, &resetAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	if fallbackID.Valid {
		b.FallbackProviderID = &fallbackID.Int64
	}
	if autoTrigger.Valid {
		b.AutoTriggerMinSeverity = autoTrigger.String
	} else {
		b.AutoTriggerMinSeverity = "critical"
	}
	if resetAt.Valid {
		t, _ := time.Parse(time.RFC3339, resetAt.String)
		b.DailyResetAt = &t
	}
	if updatedAt.Valid {
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}

	return b, nil
}

var _ database.AIRoleBudgetDialect = (*aiRoleBudgetDialect)(nil)
