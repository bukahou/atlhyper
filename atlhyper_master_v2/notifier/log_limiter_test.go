package notifier

import (
	"context"
	"fmt"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/template"
)

// fakeClock 可拨动的时钟
type fakeClock struct{ current time.Time }

func (c *fakeClock) now() time.Time          { return c.current }
func (c *fakeClock) advance(d time.Duration) { c.current = c.current.Add(d) }

func newFakeClock() *fakeClock {
	return &fakeClock{current: time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)}
}

func TestRepeatLogLimiter(t *testing.T) {
	type step struct {
		advance     time.Duration // 这一步之前时钟前进多少
		key         string
		wantAllow   bool
		wantDropped int
	}
	cases := []struct {
		name  string
		steps []step
	}{
		{
			name: "同一个键在时间窗内只放行第一次",
			steps: []step{
				{0, "a", true, 0},
				{time.Minute, "a", false, 0},
				{8 * time.Minute, "a", false, 0},
			},
		},
		{
			name: "时间窗过后再次放行，并报出期间压下的条数",
			steps: []step{
				{0, "a", true, 0},
				{time.Minute, "a", false, 0},
				{time.Minute, "a", false, 0},
				{time.Minute, "a", false, 0},
				{7 * time.Minute, "a", true, 3}, // 恰好满 10 分钟
				{time.Minute, "a", false, 0},
				{10 * time.Minute, "a", true, 1}, // 计数随放行清零，只算上一个时间窗的
			},
		},
		{
			name: "差一点不到时间窗仍不放行",
			steps: []step{
				{0, "a", true, 0},
				{10*time.Minute - time.Nanosecond, "a", false, 0},
			},
		},
		{
			name: "不同的键互不影响",
			steps: []step{
				{0, "a", true, 0},
				{time.Minute, "b", true, 0},
				{time.Minute, "a", false, 0},
				{time.Minute, "b", false, 0},
				{7 * time.Minute, "a", true, 1}, // a 满 10 分钟
				{0, "b", false, 0},              // b 晚一分钟开始，此刻才 9 分钟
				{time.Minute, "b", true, 2},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clock := newFakeClock()
			limiter := newRepeatLogLimiter(10*time.Minute, clock.now)
			for i, s := range c.steps {
				clock.advance(s.advance)
				allow, dropped := limiter.allow(s.key)
				if allow != s.wantAllow || dropped != s.wantDropped {
					t.Fatalf("第 %d 步（键 %q）：got (%v, %d), want (%v, %d)", i, s.key, allow, dropped, s.wantAllow, s.wantDropped)
				}
			}
		})
	}
}

// 键的数量有上限：告警标题里带资源名，种类不受控，记录表不能无限增长。
func TestRepeatLogLimiter_BoundedKeys(t *testing.T) {
	clock := newFakeClock()
	limiter := newRepeatLogLimiter(10*time.Minute, clock.now)

	for i := 0; i < repeatLogLimiterMaxKeys*3; i++ {
		limiter.allow(fmt.Sprintf("title-%d", i))
	}
	if got := len(limiter.entries); got > repeatLogLimiterMaxKeys {
		t.Errorf("记录了 %d 个键，上限 %d", got, repeatLogLimiterMaxKeys)
	}
}

// 过期的键在表满时先被清掉，仍在时间窗内的键不受影响。
func TestRepeatLogLimiter_EvictsExpiredFirst(t *testing.T) {
	clock := newFakeClock()
	limiter := newRepeatLogLimiter(10*time.Minute, clock.now)

	for i := 0; i < repeatLogLimiterMaxKeys-1; i++ {
		limiter.allow(fmt.Sprintf("old-%d", i))
	}
	clock.advance(11 * time.Minute) // 以上全部过期
	limiter.allow("recent")         // 表恰好满
	limiter.allow("recent")         // 压下一条
	limiter.allow("newcomer")       // 触发清理

	if _, kept := limiter.entries["recent"]; !kept {
		t.Fatal("仍在时间窗内的键不应被清掉")
	}
	if allow, _ := limiter.allow("recent"); allow {
		t.Error("仍在时间窗内的键应继续被限频")
	}
	if got := len(limiter.entries); got != 2 {
		t.Errorf("清理后应只剩 2 个键，实际 %d", got)
	}
}

// ==================== Manager：没有启用任何渠道时的限频 ====================

type noChannelRepo struct {
	database.NotifyChannelRepository
	listCalls int
}

func (r *noChannelRepo) ListEnabled(context.Context) ([]*database.NotifyChannel, error) {
	r.listCalls++
	return nil, nil
}

func TestSendWithTemplate_NoChannel_WarnsOncePerTitlePerWindow(t *testing.T) {
	repo := &noChannelRepo{}
	manager, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	clock := newFakeClock()
	manager.noChannelWarnings = newRepeatLogLimiter(noChannelWarnWindow, clock.now)

	send := func(title string) {
		t.Helper()
		if err := manager.SendWithTemplate("heartbeat_offline", &template.AlertData{Title: title}); err != nil {
			t.Fatalf("没有渠道不算错误，实际 %v", err)
		}
	}

	// 7 分钟里同一标题来了 22 条（测试环境里见到的量级），另一标题 1 条
	for i := 0; i < 22; i++ {
		send("Agent 离线")
		clock.advance(19 * time.Second)
	}
	send("Pod 重启")

	if repo.listCalls != 23 {
		t.Errorf("限频只影响日志，不应跳过渠道查询：查了 %d 次", repo.listCalls)
	}
	if got := manager.noChannelWarnings.entries["Agent 离线"].dropped; got != 21 {
		t.Errorf("「Agent 离线」22 条里应记 1 条、压下 21 条，实际压下 %d", got)
	}
	if got := manager.noChannelWarnings.entries["Pod 重启"].dropped; got != 0 {
		t.Errorf("另一个标题的第一条应照常记录，实际压下 %d", got)
	}

	// 时间窗过后的下一条重新记录，并带出期间压下的条数
	clock.advance(noChannelWarnWindow)
	if allow, dropped := manager.noChannelWarnings.allow("Agent 离线"); !allow || dropped != 21 {
		t.Errorf("时间窗过后 got (%v, %d), want (true, 21)", allow, dropped)
	}
}
