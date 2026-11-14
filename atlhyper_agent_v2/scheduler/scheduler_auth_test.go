package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/command"
)

// rejectedByMaster 模拟 Master 以 401 拒绝请求时网关返回的错误
func rejectedByMaster() error {
	return fmt.Errorf("%w: HTTP 401, body: valid cluster token required", gateway.ErrMasterRejectedIdentity)
}

// 日志文案要让人一眼分清「身份被拒」与「网络故障」。
func TestMasterFailureMessage(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		wantIdentity bool
	}{
		{"身份被拒", rejectedByMaster(), true},
		{"身份被拒（再包一层）", fmt.Errorf("push: %w", rejectedByMaster()), true},
		{"网络故障", errors.New("dial tcp: connection refused"), false},
		{"服务端错误", errors.New("unexpected status code: 500, body: Internal error"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := masterFailureMessage("推送快照失败", c.err)
			if !strings.HasPrefix(got, "推送快照失败") {
				t.Errorf("文案应以动作开头：%q", got)
			}
			if mentionsIdentity := strings.Contains(got, "身份"); mentionsIdentity != c.wantIdentity {
				t.Errorf("文案 %q 提到身份 = %v, want %v", got, mentionsIdentity, c.wantIdentity)
			}
		})
	}
}

// 拉指令被拒时把错误交回循环（由它限频记录并退避），不执行任何指令。
func TestPollAndExecuteCommands_IdentityRejected(t *testing.T) {
	var executeCalls atomic.Int32
	commandSvc := &mock.CommandService{
		ExecuteFn: func(ctx context.Context, cmd *command.Command) *command.Result {
			executeCalls.Add(1)
			return &command.Result{CommandID: cmd.ID}
		},
	}
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			return nil, rejectedByMaster()
		},
	}
	s := newTestScheduler(&mock.SnapshotService{}, commandSvc, gw)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	hasCommands, err := s.pollAndExecuteCommands("ops")

	if hasCommands || !errors.Is(err, gateway.ErrMasterRejectedIdentity) {
		t.Errorf("got (%v, %v), want (false, ErrMasterRejectedIdentity)", hasCommands, err)
	}
	if executeCalls.Load() != 0 || gw.ReportResultCalls != 0 {
		t.Errorf("被拒时不应执行或回报：execute=%d report=%d", executeCalls.Load(), gw.ReportResultCalls)
	}
}

// 身份被拒不让 Agent 退出：四条循环照常运转、持续重试，Stop 能正常收尾。
func TestScheduler_KeepsRunningWhenIdentityRejected(t *testing.T) {
	var pollAttempts, heartbeatAttempts atomic.Int32
	gw := &mock.MasterGateway{
		PollCommandsFn: func(ctx context.Context, topic string) ([]command.Command, error) {
			pollAttempts.Add(1)
			return nil, rejectedByMaster()
		},
		HeartbeatFn: func(ctx context.Context) error {
			heartbeatAttempts.Add(1)
			return rejectedByMaster()
		},
	}
	s := newTestScheduler(&mock.SnapshotService{}, &mock.CommandService{}, gw)
	s.config.SnapshotInterval = 5 * time.Millisecond
	s.config.HeartbeatInterval = 5 * time.Millisecond
	s.config.CommandPollInterval = time.Millisecond

	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && (pollAttempts.Load() < 3 || heartbeatAttempts.Load() < 2) {
		time.Sleep(5 * time.Millisecond)
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}

	if pollAttempts.Load() < 3 {
		t.Errorf("拉指令只尝试了 %d 次：被拒后应继续重试", pollAttempts.Load())
	}
	if heartbeatAttempts.Load() < 2 {
		t.Errorf("心跳只尝试了 %d 次：被拒后应继续发送", heartbeatAttempts.Load())
	}
}
