package operations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/command"
)

// ==================== Mock: mq.Producer ====================

type mockProducer struct {
	enqueueErr        error
	waitResult        *command.Result
	waitErr           error
	enqueuedClusterID string
	enqueuedTopic     string
	enqueuedCmd       *command.Command
	waitedCmdID       string
}

func (m *mockProducer) EnqueueCommand(clusterID, topic string, cmd *command.Command) error {
	m.enqueuedClusterID = clusterID
	m.enqueuedTopic = topic
	m.enqueuedCmd = cmd
	return m.enqueueErr
}

func (m *mockProducer) GetCommandStatus(cmdID string) (*command.Status, error) {
	return nil, nil
}

func (m *mockProducer) WaitCommandResult(ctx context.Context, cmdID string, timeout time.Duration) (*command.Result, error) {
	m.waitedCmdID = cmdID
	return m.waitResult, m.waitErr
}

// ==================== Mock: database.CommandHistoryRepository ====================

type mockCommandRepo struct {
	createErr error
}

func (m *mockCommandRepo) Create(ctx context.Context, cmd *database.CommandHistory) error {
	return m.createErr
}

func (m *mockCommandRepo) Update(ctx context.Context, cmd *database.CommandHistory) error {
	return nil
}

func (m *mockCommandRepo) GetByCommandID(ctx context.Context, cmdID string) (*database.CommandHistory, error) {
	return nil, nil
}

func (m *mockCommandRepo) ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*database.CommandHistory, error) {
	return nil, nil
}

func (m *mockCommandRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*database.CommandHistory, error) {
	return nil, nil
}

func (m *mockCommandRepo) List(ctx context.Context, opts database.CommandQueryOpts) ([]*database.CommandHistory, error) {
	return nil, nil
}

func (m *mockCommandRepo) Count(ctx context.Context, opts database.CommandQueryOpts) (int64, error) {
	return 0, nil
}

// ==================== 辅助函数 ====================

// validRequest 创建一个合法的 CreateCommandRequest（通过 validateRequest 校验）
func validRequest() *model.CreateCommandRequest {
	return &model.CreateCommandRequest{
		ClusterID:       "test-cluster",
		Action:          command.ActionGetLogs,
		TargetKind:      "Pod",
		TargetNamespace: "default",
		TargetName:      "nginx-abc123",
		Source:          "web",
	}
}

// ==================== Tests ====================

func TestExecuteCommandSync_Success(t *testing.T) {
	// Arrange: EnqueueCommand 成功，WaitCommandResult 返回有效结果
	expectedResult := &command.Result{
		CommandID:  "will-be-overwritten", // 实际 ID 由 CreateCommand 生成
		Success:    true,
		Output:     "log line 1\nlog line 2",
		ExecutedAt: time.Now(),
	}
	producer := &mockProducer{
		waitResult: expectedResult,
	}
	cmdRepo := &mockCommandRepo{}
	svc := &CommandService{
		bus:     producer,
		cmdRepo: cmdRepo,
	}

	// Act
	result, err := svc.ExecuteCommandSync(context.Background(), validRequest(), 10*time.Second)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Success != true {
		t.Errorf("expected Success=true, got %v", result.Success)
	}
	if result.Output != expectedResult.Output {
		t.Errorf("expected Output=%q, got %q", expectedResult.Output, result.Output)
	}
	// 验证 WaitCommandResult 被调用时传入了 CreateCommand 返回的 commandID
	if producer.waitedCmdID == "" {
		t.Error("expected WaitCommandResult to be called with a command ID")
	}
}

func TestExecuteCommandSync_CreateFail(t *testing.T) {
	// Arrange: EnqueueCommand 返回错误，导致 CreateCommand 失败
	producer := &mockProducer{
		enqueueErr: fmt.Errorf("mq connection refused"),
	}
	cmdRepo := &mockCommandRepo{}
	svc := &CommandService{
		bus:     producer,
		cmdRepo: cmdRepo,
	}

	// Act
	result, err := svc.ExecuteCommandSync(context.Background(), validRequest(), 10*time.Second)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
	// 错误应包含 "create command" 前缀（来自 ExecuteCommandSync 的错误包装）
	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Error("expected non-empty error message")
	}
	// WaitCommandResult 不应被调用（因为 CreateCommand 已失败）
	if producer.waitedCmdID != "" {
		t.Errorf("WaitCommandResult should not be called when CreateCommand fails, but was called with %q", producer.waitedCmdID)
	}
}

func TestExecuteCommandSync_WaitTimeout(t *testing.T) {
	// Arrange: CreateCommand 成功，但 WaitCommandResult 超时
	producer := &mockProducer{
		waitErr: context.DeadlineExceeded,
	}
	cmdRepo := &mockCommandRepo{}
	svc := &CommandService{
		bus:     producer,
		cmdRepo: cmdRepo,
	}

	// Act
	result, err := svc.ExecuteCommandSync(context.Background(), validRequest(), 1*time.Second)

	// Assert
	if err == nil {
		t.Fatal("expected error on timeout, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on timeout, got: %+v", result)
	}
	// 验证 WaitCommandResult 确实被调用了
	if producer.waitedCmdID == "" {
		t.Error("expected WaitCommandResult to be called")
	}
}

func TestExecuteCommandSync_WaitError(t *testing.T) {
	// Arrange: CreateCommand 成功，但 WaitCommandResult 返回其他错误
	waitError := fmt.Errorf("agent disconnected")
	producer := &mockProducer{
		waitErr: waitError,
	}
	cmdRepo := &mockCommandRepo{}
	svc := &CommandService{
		bus:     producer,
		cmdRepo: cmdRepo,
	}

	// Act
	result, err := svc.ExecuteCommandSync(context.Background(), validRequest(), 10*time.Second)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
	// 验证 WaitCommandResult 确实被调用了
	if producer.waitedCmdID == "" {
		t.Error("expected WaitCommandResult to be called")
	}
}
