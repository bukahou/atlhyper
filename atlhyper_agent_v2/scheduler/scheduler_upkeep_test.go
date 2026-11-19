package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/command"
)

func fastBackoff() wait.Backoff {
	return wait.Backoff{Duration: time.Millisecond, Factor: 1, Steps: 4}
}

func reportingScheduler(report func(context.Context, *command.Result) error) (*Scheduler, *mock.MasterGateway) {
	gw := &mock.MasterGateway{
		PollCommandsFn: func(context.Context, string) ([]command.Command, error) {
			return []command.Command{{ID: "cmd-1", Action: "scale"}}, nil
		},
		ReportResultFn: report,
	}
	commandSvc := &mock.CommandService{ExecuteFn: func(_ context.Context, cmd *command.Command) *command.Result {
		return &command.Result{CommandID: cmd.ID, Success: true}
	}}
	s := newTestScheduler(&mock.SnapshotService{}, commandSvc, gw)
	s.reportBackoff = fastBackoff()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s, gw
}

// 回报结果遇到网络类失败会重试：写操作已经在集群里生效，结果不能就此丢掉
func TestReportResult_RetriesTransientFailures(t *testing.T) {
	var calls atomic.Int32
	s, _ := reportingScheduler(func(context.Context, *command.Result) error {
		if calls.Add(1) < 3 {
			return errors.New("connection reset")
		}
		return nil
	})
	defer s.cancel()
	s.pollAndExecuteCommands("ops")
	if calls.Load() != 3 {
		t.Errorf("回报调用 %d 次，want 3（两次失败后成功）", calls.Load())
	}
}

// 身份被 Master 拒绝：重试也不会好，不重试
func TestReportResult_DoesNotRetryIdentityRejection(t *testing.T) {
	var calls atomic.Int32
	s, _ := reportingScheduler(func(context.Context, *command.Result) error {
		calls.Add(1)
		return fmt.Errorf("report: %w", gateway.ErrMasterRejectedIdentity)
	})
	defer s.cancel()
	s.pollAndExecuteCommands("ops")
	if calls.Load() != 1 {
		t.Errorf("回报调用 %d 次，want 1", calls.Load())
	}
}

// OTel 刷新有自己的循环：启动后会被调用，停止后它拿到的 ctx 被取消
func TestOTelLoop_RunsAndStopsWithScheduler(t *testing.T) {
	refreshed := make(chan context.Context, 8)
	snapshotSvc := &mock.SnapshotService{RefreshOTelFn: func(ctx context.Context) {
		select {
		case refreshed <- ctx:
		default:
		}
	}}
	s := newTestScheduler(snapshotSvc, &mock.CommandService{}, &mock.MasterGateway{})
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	var ctx context.Context
	select {
	case ctx = <-refreshed:
	case <-time.After(2 * time.Second):
		t.Fatal("OTel 刷新没有被调用")
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil {
		t.Error("调度器停止后 OTel 刷新的 ctx 应被取消")
	}
}

// 存活：快照循环在上限内完成过一轮；卡住超过上限即判不存活（推送失败不影响存活）
func TestAlive(t *testing.T) {
	s := newTestScheduler(&mock.SnapshotService{}, &mock.CommandService{}, &mock.MasterGateway{})
	start := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	s.startedAt.Store(start.UnixNano())
	limit := s.maxLoopSilence()

	if !s.Alive(start.Add(limit - time.Second)) {
		t.Error("启动后还没跑完第一轮，但在上限内：应存活")
	}
	if s.Alive(start.Add(limit + time.Second)) {
		t.Error("启动后超过上限一轮都没完成：应判不存活")
	}
	s.lastLoopDone.Store(start.Add(limit).UnixNano())
	if !s.Alive(start.Add(limit + time.Second)) {
		t.Error("刚完成一轮：应存活")
	}
}
