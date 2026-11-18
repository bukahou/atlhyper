package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// newTestScheduler 创建用于测试的 Scheduler 实例
func newTestScheduler(snapshotSvc *mock.SnapshotService, commandSvc *mock.CommandService, gw *mock.MasterGateway) *Scheduler {
	return &Scheduler{
		config: Config{
			SnapshotInterval:    10 * time.Second,
			CommandPollInterval: 1 * time.Second,
			HeartbeatInterval:   15 * time.Second,
			SnapshotTimeout:     5 * time.Second,
			CommandPollTimeout:  5 * time.Second,
			HeartbeatTimeout:    5 * time.Second,
		},
		snapshotSvc: snapshotSvc,
		commandSvc:  commandSvc,
		masterGw:    gw,
	}
}

// =============================================================================
// 生命周期测试
// =============================================================================

func TestScheduler_StartStop(t *testing.T) {
	snapshotSvc := &mock.SnapshotService{}
	commandSvc := &mock.CommandService{}
	gw := &mock.MasterGateway{}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}

// =============================================================================
// collectAndPushSnapshot 测试
// =============================================================================

func TestCollectAndPushSnapshot_Success(t *testing.T) {
	snapshot := &cluster.ClusterSnapshot{
		Pods:  []cluster.Pod{{}, {}},
		Nodes: []cluster.Node{{}},
	}

	snapshotSvc := &mock.SnapshotService{
		CollectFn: func(ctx context.Context) (*cluster.ClusterSnapshot, error) {
			return snapshot, nil
		},
	}
	commandSvc := &mock.CommandService{}
	gw := &mock.MasterGateway{}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	s.collectAndPushSnapshot()

	if gw.PushSnapshotCalls != 1 {
		t.Errorf("PushSnapshotCalls = %d, want 1", gw.PushSnapshotCalls)
	}
	if gw.LastSnapshot == nil {
		t.Error("LastSnapshot is nil, want non-nil")
	}
}

func TestCollectAndPushSnapshot_CollectError(t *testing.T) {
	snapshotSvc := &mock.SnapshotService{
		CollectFn: func(ctx context.Context) (*cluster.ClusterSnapshot, error) {
			return nil, errors.New("collect failed")
		},
	}
	commandSvc := &mock.CommandService{}
	gw := &mock.MasterGateway{}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	s.collectAndPushSnapshot()

	if gw.PushSnapshotCalls != 0 {
		t.Errorf("PushSnapshotCalls = %d, want 0 (should NOT push on collect error)", gw.PushSnapshotCalls)
	}
}

func TestCollectAndPushSnapshot_PushError(t *testing.T) {
	snapshot := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{{}},
	}

	snapshotSvc := &mock.SnapshotService{
		CollectFn: func(ctx context.Context) (*cluster.ClusterSnapshot, error) {
			return snapshot, nil
		},
	}
	commandSvc := &mock.CommandService{}
	gw := &mock.MasterGateway{
		PushSnapshotFn: func(ctx context.Context, snapshot *cluster.ClusterSnapshot) error {
			return errors.New("push failed")
		},
	}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	// Should not panic
	s.collectAndPushSnapshot()

	if gw.PushSnapshotCalls != 1 {
		t.Errorf("PushSnapshotCalls = %d, want 1 (push was attempted)", gw.PushSnapshotCalls)
	}
}

// =============================================================================
// pollAndExecuteCommands 测试
// =============================================================================

func TestPollAndExecuteCommands_WithCommands(t *testing.T) {
	commands := []command.Command{
		{ID: "cmd-1", Action: "scale"},
		{ID: "cmd-2", Action: "restart"},
	}

	var executeCalls atomic.Int32
	snapshotSvc := &mock.SnapshotService{}
	commandSvc := &mock.CommandService{
		ExecuteFn: func(ctx context.Context, cmd *command.Command) *command.Result {
			executeCalls.Add(1)
			return &command.Result{CommandID: cmd.ID, Success: true}
		},
	}
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			return commands, nil
		},
	}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	s.pollAndExecuteCommands("ops")

	if got := executeCalls.Load(); got != 2 {
		t.Errorf("Execute called %d times, want 2", got)
	}
	if gw.ReportResultCalls != 2 {
		t.Errorf("ReportResultCalls = %d, want 2", gw.ReportResultCalls)
	}
}

func TestPollAndExecuteCommands_NoCommands(t *testing.T) {
	snapshotSvc := &mock.SnapshotService{}
	commandSvc := &mock.CommandService{}
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			return nil, nil
		},
	}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	s.pollAndExecuteCommands("ops")

	if gw.ReportResultCalls != 0 {
		t.Errorf("ReportResultCalls = %d, want 0", gw.ReportResultCalls)
	}
}

func TestPollAndExecuteCommands_PollError(t *testing.T) {
	snapshotSvc := &mock.SnapshotService{}
	commandSvc := &mock.CommandService{}
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			return nil, errors.New("poll failed")
		},
	}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	// Should not panic
	s.pollAndExecuteCommands("ops")

	if gw.ReportResultCalls != 0 {
		t.Errorf("ReportResultCalls = %d, want 0", gw.ReportResultCalls)
	}
}

func TestPollAndExecuteCommands_ReportError(t *testing.T) {
	commands := []command.Command{
		{ID: "cmd-1", Action: "scale"},
	}

	snapshotSvc := &mock.SnapshotService{}
	commandSvc := &mock.CommandService{
		ExecuteFn: func(ctx context.Context, cmd *command.Command) *command.Result {
			return &command.Result{CommandID: cmd.ID, Success: true}
		},
	}
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			return commands, nil
		},
		ReportResultFn: func(ctx context.Context, result *command.Result) error {
			return errors.New("report failed")
		},
	}

	s := newTestScheduler(snapshotSvc, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	// Should not panic (error is logged but not fatal)
	s.pollAndExecuteCommands("ops")

	if gw.ReportResultCalls != 1 {
		t.Errorf("ReportResultCalls = %d, want 1", gw.ReportResultCalls)
	}
}

// =============================================================================
// 指令各阶段的时间预算
// =============================================================================

// 长轮询等了很久才等到指令：执行与回报不能只剩轮询剩下的那点时间。
func TestPollAndExecuteCommands_ExecuteAndReportHaveOwnBudgets(t *testing.T) {
	var execCtxErr, reportCtxErr error
	commandSvc := &mock.CommandService{
		ExecuteFn: func(ctx context.Context, cmd *command.Command) *command.Result {
			time.Sleep(80 * time.Millisecond) // 执行耗时超过轮询预算剩余的部分
			execCtxErr = ctx.Err()
			return &command.Result{CommandID: cmd.ID, Success: true}
		},
	}
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			time.Sleep(70 * time.Millisecond) // 指令在长轮询快结束时才到
			return []command.Command{{ID: "cmd-1", Action: "restart"}}, nil
		},
		ReportResultFn: func(ctx context.Context, result *command.Result) error {
			reportCtxErr = ctx.Err()
			return nil
		},
	}

	s := newTestScheduler(&mock.SnapshotService{}, commandSvc, gw)
	s.config.CommandPollTimeout = 100 * time.Millisecond
	s.config.CommandExecTimeout = time.Second
	s.config.CommandReportTimeout = time.Second
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	s.pollAndExecuteCommands("ops")

	if execCtxErr != nil {
		t.Errorf("执行阶段的 ctx 被轮询预算拖着过期了: %v", execCtxErr)
	}
	if reportCtxErr != nil {
		t.Errorf("回报阶段的 ctx 被轮询预算拖着过期了: %v", reportCtxErr)
	}
}

// 执行超时后仍要把结果报回去：回报用自己的预算，不沿用已过期的执行 ctx。
func TestPollAndExecuteCommands_ReportsEvenWhenExecutionTimedOut(t *testing.T) {
	var reportCtxErr error
	reported := false
	commandSvc := &mock.CommandService{
		ExecuteFn: func(ctx context.Context, cmd *command.Command) *command.Result {
			<-ctx.Done() // 一直执行到超时
			return &command.Result{CommandID: cmd.ID, Success: false, Error: ctx.Err().Error()}
		},
	}
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			return []command.Command{{ID: "cmd-1", Action: "restart"}}, nil
		},
		ReportResultFn: func(ctx context.Context, result *command.Result) error {
			reported = true
			reportCtxErr = ctx.Err()
			return nil
		},
	}

	s := newTestScheduler(&mock.SnapshotService{}, commandSvc, gw)
	s.config.CommandExecTimeout = 50 * time.Millisecond
	s.config.CommandReportTimeout = time.Second
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	s.pollAndExecuteCommands("ops")

	if !reported {
		t.Fatal("执行超时后没有回报结果")
	}
	if reportCtxErr != nil {
		t.Errorf("回报用的 ctx 已过期: %v", reportCtxErr)
	}
}
