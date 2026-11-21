package budget

import (
	"context"
	"errors"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// fakeRepo 只实现 Get / ResetDailyUsage / ResetMonthlyUsage
type fakeRepo struct {
	database.AIRoleBudgetRepository
	budget       *database.AIRoleBudget
	getErr       error
	dailyResets  int
	monthlyReset int
}

func (f *fakeRepo) Get(context.Context, string) (*database.AIRoleBudget, error) {
	if f.budget == nil {
		return nil, f.getErr
	}
	copied := *f.budget
	return &copied, f.getErr
}

func (f *fakeRepo) ResetDailyUsage(context.Context, string) error {
	f.dailyResets++
	return nil
}

func (f *fakeRepo) ResetMonthlyUsage(context.Context, string) error {
	f.monthlyReset++
	return nil
}

func at(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func usedUp() *database.AIRoleBudget {
	return &database.AIRoleBudget{
		Role:           "chat",
		DailyCallLimit: 100, DailyCallsUsed: 100,
		MonthlyCallLimit: 1000, MonthlyCallsUsed: 1000,
	}
}

func TestCurrent_Rollover(t *testing.T) {
	now := time.Date(2026, 11, 19, 3, 0, 0, 0, time.UTC)
	cases := []struct {
		name           string
		dailyResetAt   *time.Time
		monthlyResetAt *time.Time
		wantDaily      bool
		wantMonthly    bool
	}{
		// 从没重置过（种子数据、新建的预算）：视为到期 —— 否则用满一次就永远不可用
		{"从没重置过", nil, nil, true, true},
		{"同一天（UTC）", at("2026-11-19T00:00:01Z"), at("2026-11-01T00:00:00Z"), false, false},
		{"前一天（UTC）", at("2026-11-18T23:59:59Z"), at("2026-11-01T00:00:00Z"), true, false},
		// 存的是 +09:00：按 UTC 比较，东京的 11-19 08:00 = UTC 11-18 23:00，是前一天
		{"带时区偏移的前一天", at("2026-11-19T08:00:00+09:00"), at("2026-11-02T00:00:00+09:00"), true, false},
		{"上个月", at("2026-10-31T12:00:00Z"), at("2026-10-01T00:00:00Z"), true, true},
		{"去年同月", at("2025-11-19T12:00:00Z"), at("2025-11-01T00:00:00Z"), true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := usedUp()
			b.DailyResetAt, b.MonthlyResetAt = c.dailyResetAt, c.monthlyResetAt
			repo := &fakeRepo{budget: b}
			ledger := &Ledger{repo: repo, now: func() time.Time { return now }}

			got, err := ledger.Current(context.Background(), "chat")
			if err != nil {
				t.Fatal(err)
			}
			if (repo.dailyResets == 1) != c.wantDaily || (repo.monthlyReset == 1) != c.wantMonthly {
				t.Fatalf("重置 日=%d 月=%d，want 日=%v 月=%v", repo.dailyResets, repo.monthlyReset, c.wantDaily, c.wantMonthly)
			}
			if c.wantDaily && got.DailyCallsUsed != 0 {
				t.Error("日重置后返回的用量应为 0")
			}
			if c.wantMonthly && got.MonthlyCallsUsed != 0 {
				t.Error("月重置后返回的用量应为 0")
			}
			if HasRemaining(got) != (c.wantDaily && c.wantMonthly) {
				t.Errorf("HasRemaining = %v", HasRemaining(got))
			}
		})
	}
}

func TestCurrent_NoBudgetRecord(t *testing.T) {
	ledger := NewLedger(&fakeRepo{})
	got, err := ledger.Current(context.Background(), "chat")
	if err != nil || got != nil {
		t.Fatalf("没有预算记录应返回 (nil, nil)，实际 (%v, %v)", got, err)
	}
	if !HasRemaining(got) {
		t.Error("没有预算记录 = 不限额")
	}
}

// 读不到预算不能当成「有余额」：错误原样返回，由调用方决定
func TestCurrent_ReadError(t *testing.T) {
	ledger := NewLedger(&fakeRepo{getErr: errors.New("db locked")})
	if _, err := ledger.Current(context.Background(), "chat"); err == nil {
		t.Fatal("读预算失败应返回错误")
	}
}

func TestHasRemaining(t *testing.T) {
	cases := []struct {
		name string
		b    database.AIRoleBudget
		want bool
	}{
		{"不限额", database.AIRoleBudget{DailyCallsUsed: 999}, true},
		{"日次数用满", database.AIRoleBudget{DailyCallLimit: 10, DailyCallsUsed: 10}, false},
		{"日输入 token 用满", database.AIRoleBudget{DailyInputTokenLimit: 100, DailyInputTokensUsed: 100}, false},
		{"日输出 token 用满", database.AIRoleBudget{DailyOutputTokenLimit: 100, DailyOutputTokensUsed: 101}, false},
		{"月次数用满", database.AIRoleBudget{MonthlyCallLimit: 10, MonthlyCallsUsed: 10}, false},
		{"月输入 token 用满", database.AIRoleBudget{MonthlyInputTokenLimit: 1, MonthlyInputTokensUsed: 1}, false},
		{"月输出 token 用满", database.AIRoleBudget{MonthlyOutputTokenLimit: 1, MonthlyOutputTokensUsed: 1}, false},
		{"都没满", database.AIRoleBudget{DailyCallLimit: 10, DailyCallsUsed: 9, MonthlyCallLimit: 100, MonthlyCallsUsed: 50}, true},
	}
	for _, c := range cases {
		if got := HasRemaining(&c.b); got != c.want {
			t.Errorf("%s: HasRemaining = %v, want %v", c.name, got, c.want)
		}
	}
}
