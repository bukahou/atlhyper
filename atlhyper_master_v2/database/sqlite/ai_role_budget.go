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
		(role,
		 daily_input_token_limit, daily_output_token_limit, daily_call_limit,
		 daily_input_tokens_used, daily_output_tokens_used, daily_calls_used, daily_reset_at,
		 monthly_input_token_limit, monthly_output_token_limit, monthly_call_limit,
		 monthly_input_tokens_used, monthly_output_tokens_used, monthly_calls_used, monthly_reset_at,
		 fallback_provider_id, auto_trigger_min_severity, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	var dailyResetAt, monthlyResetAt sql.NullString
	if b.DailyResetAt != nil {
		dailyResetAt = sql.NullString{String: b.DailyResetAt.Format(time.RFC3339), Valid: true}
	}
	if b.MonthlyResetAt != nil {
		monthlyResetAt = sql.NullString{String: b.MonthlyResetAt.Format(time.RFC3339), Valid: true}
	}
	args := []any{
		b.Role,
		b.DailyInputTokenLimit, b.DailyOutputTokenLimit, b.DailyCallLimit,
		b.DailyInputTokensUsed, b.DailyOutputTokensUsed, b.DailyCallsUsed, dailyResetAt,
		b.MonthlyInputTokenLimit, b.MonthlyOutputTokenLimit, b.MonthlyCallLimit,
		b.MonthlyInputTokensUsed, b.MonthlyOutputTokensUsed, b.MonthlyCallsUsed, monthlyResetAt,
		b.FallbackProviderID, b.AutoTriggerMinSeverity, b.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

const selectBudgetCols = `role,
	daily_input_token_limit, daily_output_token_limit, daily_call_limit,
	daily_input_tokens_used, daily_output_tokens_used, daily_calls_used, daily_reset_at,
	monthly_input_token_limit, monthly_output_token_limit, monthly_call_limit,
	monthly_input_tokens_used, monthly_output_tokens_used, monthly_calls_used, monthly_reset_at,
	fallback_provider_id, auto_trigger_min_severity, updated_at`

func (d *aiRoleBudgetDialect) SelectByRole(role string) (string, []any) {
	return `SELECT ` + selectBudgetCols + ` FROM ai_role_budget WHERE role = ?`, []any{role}
}

func (d *aiRoleBudgetDialect) SelectAll() (string, []any) {
	return `SELECT ` + selectBudgetCols + ` FROM ai_role_budget ORDER BY role`, nil
}

func (d *aiRoleBudgetDialect) Delete(role string) (string, []any) {
	return `DELETE FROM ai_role_budget WHERE role = ?`, []any{role}
}

func (d *aiRoleBudgetDialect) IncrementUsage(role string, inputTokens, outputTokens int) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `UPDATE ai_role_budget SET
		daily_input_tokens_used = daily_input_tokens_used + ?,
		daily_output_tokens_used = daily_output_tokens_used + ?,
		daily_calls_used = daily_calls_used + 1,
		monthly_input_tokens_used = monthly_input_tokens_used + ?,
		monthly_output_tokens_used = monthly_output_tokens_used + ?,
		monthly_calls_used = monthly_calls_used + 1,
		updated_at = ?
		WHERE role = ?`,
		[]any{inputTokens, outputTokens, inputTokens, outputTokens, now, role}
}

func (d *aiRoleBudgetDialect) ResetDailyUsage(role string) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `UPDATE ai_role_budget SET
		daily_input_tokens_used = 0, daily_output_tokens_used = 0, daily_calls_used = 0,
		daily_reset_at = ?, updated_at = ?
		WHERE role = ?`,
		[]any{now, now, role}
}

func (d *aiRoleBudgetDialect) ResetMonthlyUsage(role string) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `UPDATE ai_role_budget SET
		monthly_input_tokens_used = 0, monthly_output_tokens_used = 0, monthly_calls_used = 0,
		monthly_reset_at = ?, updated_at = ?
		WHERE role = ?`,
		[]any{now, now, role}
}

func (d *aiRoleBudgetDialect) ScanRow(rows *sql.Rows) (*database.AIRoleBudget, error) {
	b := &database.AIRoleBudget{}
	var fallbackID sql.NullInt64
	var dailyResetAt, monthlyResetAt, updatedAt sql.NullString
	var autoTrigger sql.NullString

	err := rows.Scan(
		&b.Role,
		&b.DailyInputTokenLimit, &b.DailyOutputTokenLimit, &b.DailyCallLimit,
		&b.DailyInputTokensUsed, &b.DailyOutputTokensUsed, &b.DailyCallsUsed, &dailyResetAt,
		&b.MonthlyInputTokenLimit, &b.MonthlyOutputTokenLimit, &b.MonthlyCallLimit,
		&b.MonthlyInputTokensUsed, &b.MonthlyOutputTokensUsed, &b.MonthlyCallsUsed, &monthlyResetAt,
		&fallbackID, &autoTrigger, &updatedAt,
	)
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
	if dailyResetAt.Valid {
		t, _ := time.Parse(time.RFC3339, dailyResetAt.String)
		b.DailyResetAt = &t
	}
	if monthlyResetAt.Valid {
		t, _ := time.Parse(time.RFC3339, monthlyResetAt.String)
		b.MonthlyResetAt = &t
	}
	if updatedAt.Valid {
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}

	return b, nil
}

var _ database.AIRoleBudgetDialect = (*aiRoleBudgetDialect)(nil)
