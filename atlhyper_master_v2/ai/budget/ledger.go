// Package budget AI 角色预算：「取预算 → 跨日 / 跨月清零 → 判断余额」只有这一处。
// 对话、深度分析、后台自动分析都经它判断，不各写一份。
package budget

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var log = logger.Module("AI-Budget")

// Ledger 角色预算账本
type Ledger struct {
	repo database.AIRoleBudgetRepository
	now  func() time.Time
}

// NewLedger 创建账本
func NewLedger(repo database.AIRoleBudgetRepository) *Ledger {
	return &Ledger{repo: repo, now: time.Now}
}

// Current 取角色当前周期的预算：跨了自然日 / 自然月（按 UTC）的先把用量清零。
// 从没重置过的（种子数据、新建的预算）视为已到期。没有预算记录返回 (nil, nil)，表示不限额。
func (l *Ledger) Current(ctx context.Context, role string) (*database.AIRoleBudget, error) {
	b, err := l.repo.Get(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("读取角色 %s 的预算: %w", role, err)
	}
	if b == nil {
		return nil, nil
	}
	now := l.now().UTC()
	if dailyRolloverDue(b.DailyResetAt, now) {
		if err := l.repo.ResetDailyUsage(ctx, role); err != nil {
			return nil, fmt.Errorf("重置角色 %s 的日用量: %w", role, err)
		}
		b.DailyInputTokensUsed, b.DailyOutputTokensUsed, b.DailyCallsUsed = 0, 0, 0
		log.Info("角色预算跨日清零", "role", role)
	}
	if monthlyRolloverDue(b.MonthlyResetAt, now) {
		if err := l.repo.ResetMonthlyUsage(ctx, role); err != nil {
			return nil, fmt.Errorf("重置角色 %s 的月用量: %w", role, err)
		}
		b.MonthlyInputTokensUsed, b.MonthlyOutputTokensUsed, b.MonthlyCallsUsed = 0, 0, 0
		log.Info("角色预算跨月清零", "role", role)
	}
	return b, nil
}

// HasRemaining 当前周期是否还有余额（各项限额为 0 表示不限）；nil = 不限额
func HasRemaining(b *database.AIRoleBudget) bool {
	if b == nil {
		return true
	}
	limits := []struct{ limit, used int }{
		{b.DailyInputTokenLimit, b.DailyInputTokensUsed},
		{b.DailyOutputTokenLimit, b.DailyOutputTokensUsed},
		{b.DailyCallLimit, b.DailyCallsUsed},
		{b.MonthlyInputTokenLimit, b.MonthlyInputTokensUsed},
		{b.MonthlyOutputTokenLimit, b.MonthlyOutputTokensUsed},
		{b.MonthlyCallLimit, b.MonthlyCallsUsed},
	}
	for _, l := range limits {
		if l.limit > 0 && l.used >= l.limit {
			return false
		}
	}
	return true
}

func dailyRolloverDue(resetAt *time.Time, now time.Time) bool {
	if resetAt == nil {
		return true
	}
	ry, rm, rd := resetAt.UTC().Date()
	ny, nm, nd := now.Date()
	return time.Date(ny, nm, nd, 0, 0, 0, 0, time.UTC).After(time.Date(ry, rm, rd, 0, 0, 0, 0, time.UTC))
}

func monthlyRolloverDue(resetAt *time.Time, now time.Time) bool {
	if resetAt == nil {
		return true
	}
	ry, rm, _ := resetAt.UTC().Date()
	ny, nm, _ := now.Date()
	return time.Date(ny, nm, 1, 0, 0, 0, 0, time.UTC).After(time.Date(ry, rm, 1, 0, 0, 0, 0, time.UTC))
}
