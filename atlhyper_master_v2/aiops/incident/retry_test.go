package incident

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

var errBusy = errors.New("database is locked (5) (SQLITE_BUSY)")

// flakyRepo 按调用顺序记录成功的写入；failNext > 0 时接下来的写入失败并返回 failWith
type flakyRepo struct {
	database.AIOpsIncidentRepository
	writes   []string // 成功执行的写入，按顺序
	attempts int      // 写入尝试次数（含失败）
	failNext int
	failWith error
	timeline []*database.AIOpsIncidentTimeline
}

func (f *flakyRepo) write(name string) error {
	f.attempts++
	if f.failNext > 0 {
		f.failNext--
		return f.failWith
	}
	f.writes = append(f.writes, name)
	return nil
}

func (f *flakyRepo) CreateIncident(_ context.Context, inc *database.AIOpsIncident) error {
	return f.write("create:" + inc.ID)
}
func (f *flakyRepo) UpdateState(_ context.Context, id, state, _ string) error {
	return f.write("state:" + id + ":" + state)
}
func (f *flakyRepo) UpdatePeakRisk(_ context.Context, id string, _ float64) error {
	return f.write("peak:" + id)
}
func (f *flakyRepo) IncrementRecurrence(_ context.Context, id string) error {
	return f.write("recurrence:" + id)
}
func (f *flakyRepo) Resolve(_ context.Context, id string, _ time.Time) error {
	return f.write("resolve:" + id)
}
func (f *flakyRepo) UpdateRootCause(_ context.Context, id, _ string) error {
	return f.write("rootcause:" + id)
}
func (f *flakyRepo) AddEntity(_ context.Context, e *database.AIOpsIncidentEntity) error {
	return f.write("entity:" + e.IncidentID)
}
func (f *flakyRepo) GetEntities(context.Context, string) ([]*database.AIOpsIncidentEntity, error) {
	return nil, nil
}
func (f *flakyRepo) AddTimeline(_ context.Context, t *database.AIOpsIncidentTimeline) error {
	if err := f.write("timeline:" + t.IncidentID + ":" + t.EventType); err != nil {
		return err
	}
	f.timeline = append(f.timeline, t)
	return nil
}

func newRetryFixture() (*Store, *flakyRepo, *time.Time) {
	repo := &flakyRepo{failWith: errBusy}
	store := NewStore(repo)
	current := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return current }
	return store, repo, &current
}

func stateWrites(writes []string) []string {
	var out []string
	for _, w := range writes {
		if len(w) > 6 && (w[:6] == "state:" || w[:6] == "recurr" || w[:6] == "resolv" || w[:6] == "create") {
			out = append(out, w)
		}
	}
	return out
}

// 写锁被占住时，状态写入失败后不丢：稍后按原来的顺序补上，时间线保留原来的时刻。
func TestStore_TransientFailureIsRetriedInOrder(t *testing.T) {
	store, repo, clock := newRetryFixture()
	ctx := context.Background()
	occurredAt := *clock

	repo.failNext = 1
	store.IncrementRecurrence(ctx, "inc-1", risk("default/pod/a", 0.6, 0.6), occurredAt)
	store.UpdateState(ctx, "inc-1", aiops.StateWarning, risk("default/pod/a", 0.6, 0.6), occurredAt)

	if got := stateWrites(repo.writes); len(got) != 0 {
		t.Fatalf("写锁被占期间不应有写入成功，实际 %v", got)
	}
	if store.PendingWrites() == 0 {
		t.Fatal("失败的写入应留在待重试队列里")
	}

	*clock = clock.Add(time.Minute) // 过了退避时间
	store.FlushPending(ctx)

	want := []string{"recurrence:inc-1", "state:inc-1:warning"}
	if got := stateWrites(repo.writes); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("补写顺序 = %v, want %v", got, want)
	}
	if store.PendingWrites() != 0 {
		t.Errorf("补写完成后队列应为空，还剩 %d", store.PendingWrites())
	}
	for _, entry := range repo.timeline {
		if !entry.Timestamp.Equal(occurredAt) {
			t.Errorf("时间线应保留事情发生的时刻 %v，实际 %v", occurredAt, entry.Timestamp)
		}
	}
}

// 一次失败之后的退避时间内，后来的写入直接排队，不再去碰数据库：
// 每次尝试都要等满写锁超时，逐条去试会把快照处理整条拖住。
func TestStore_NoWriteAttemptsDuringBackoff(t *testing.T) {
	store, repo, _ := newRetryFixture()
	ctx := context.Background()

	repo.failNext = 100
	store.UpdateState(ctx, "inc-1", aiops.StateIncident, risk("default/pod/a", 0.8, 0.8), time.Now())
	attemptsAfterFirst := repo.attempts

	for i := 0; i < 5; i++ {
		store.UpdateState(ctx, fmt.Sprintf("inc-%d", i+2), aiops.StateIncident, risk("default/pod/a", 0.8, 0.8), time.Now())
	}
	if repo.attempts != attemptsAfterFirst {
		t.Errorf("退避期间又尝试了 %d 次写入，want 0", repo.attempts-attemptsAfterFirst)
	}
	if attemptsAfterFirst != 1 {
		t.Errorf("第一次失败后应立即停手，实际尝试了 %d 次", attemptsAfterFirst)
	}
}

// 重试也不会成功的错误（约束冲突等）只记日志并丢弃，不能把后面的写入堵住。
func TestStore_PermanentErrorDoesNotBlockQueue(t *testing.T) {
	store, repo, _ := newRetryFixture()
	ctx := context.Background()

	repo.failNext = 1
	repo.failWith = errors.New("constraint failed: UNIQUE constraint failed (1555)")
	store.UpdateRootCause(ctx, "inc-1", "default/pod/a")
	store.UpdateRootCause(ctx, "inc-2", "default/pod/b")

	if fmt.Sprint(repo.writes) != fmt.Sprint([]string{"rootcause:inc-2"}) {
		t.Errorf("writes = %v, want 只有 inc-2 成功", repo.writes)
	}
	if store.PendingWrites() != 0 {
		t.Errorf("不可重试的写入不应留在队列里，还剩 %d", store.PendingWrites())
	}
}

// 创建事件时写库失败：事件号照常返回（状态机要用它关联后续变化），创建本身稍后补上，
// 而且排在这条事件后续的状态变化之前。
func TestStore_CreateReturnsIDAndIsRetriedBeforeLaterWrites(t *testing.T) {
	store, repo, clock := newRetryFixture()
	ctx := context.Background()

	repo.failNext = 1
	id := store.Create(ctx, "c1", "default/pod/a", risk("default/pod/a", 0.3, 0.3), *clock)
	if id == "" {
		t.Fatal("写库失败时也应返回事件号")
	}
	store.UpdateState(ctx, id, aiops.StateIncident, risk("default/pod/a", 0.8, 0.8), *clock)

	*clock = clock.Add(time.Minute)
	store.FlushPending(ctx)

	got := stateWrites(repo.writes)
	want := []string{"create:" + id, "state:" + id + ":incident"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("补写顺序 = %v, want %v", got, want)
	}
}

// 队列有上限：存储长时间不可用时丢最旧的，不无限占内存。
func TestStore_PendingQueueIsBounded(t *testing.T) {
	store, repo, _ := newRetryFixture()
	ctx := context.Background()

	repo.failNext = 1 << 30
	for i := 0; i < maxPendingWrites+50; i++ {
		store.UpdateRootCause(ctx, fmt.Sprintf("inc-%d", i), "default/pod/a")
	}
	if got := store.PendingWrites(); got != maxPendingWrites {
		t.Errorf("队列长度 = %d, want 上限 %d", got, maxPendingWrites)
	}
}

// 没有事件号的条目（从未建过事件的实体）关闭时不写任何东西
func TestStore_ResolveWithoutIncidentWritesNothing(t *testing.T) {
	store, repo, _ := newRetryFixture()
	store.Resolve(context.Background(), "", "default/pod/a", time.Now())
	if repo.attempts != 0 {
		t.Errorf("没有事件号时不应写库，实际尝试 %d 次: %v", repo.attempts, repo.writes)
	}
}
