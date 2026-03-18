// atlhyper_master_v2/service/operations/command.go
// CommandService 指令写入服务
// 负责接收 Web/AI 的指令请求，校验后写入 MQ 并持久化到数据库
package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/command"
)

var log = logger.Module("CommandService")

// CommandService 指令服务
type CommandService struct {
	bus     mq.Producer
	cmdRepo database.CommandHistoryRepository
}

// NewCommandService 创建 CommandService
func NewCommandService(bus mq.Producer, cmdRepo database.CommandHistoryRepository) *CommandService {
	return &CommandService{
		bus:     bus,
		cmdRepo: cmdRepo,
	}
}

// CreateCommand 创建指令
func (s *CommandService) CreateCommand(req *model.CreateCommandRequest) (*model.CreateCommandResponse, error) {
	// 1. 校验
	if err := s.validateRequest(req); err != nil {
		return nil, fmt.Errorf("validate request: %w", err)
	}

	// 2. 生成指令 ID
	commandID := uuid.New().String()

	// 3. 构建 Command
	cmd := &command.Command{
		ID:        commandID,
		ClusterID: req.ClusterID,
		Action:    req.Action,
		Kind:      req.TargetKind,
		Namespace: req.TargetNamespace,
		Name:      req.TargetName,
		Params:    req.Params,
		Source:    req.Source,
		CreatedAt: time.Now(),
	}

	// 4. 写入 MQ（按来源路由 topic）
	topic := mq.TopicOps
	if req.Source == "ai" {
		topic = mq.TopicAI
	}
	if err := s.bus.EnqueueCommand(req.ClusterID, topic, cmd); err != nil {
		return nil, fmt.Errorf("enqueue command: %w", err)
	}

	// 5. 持久化指令历史
	paramsJSON, _ := json.Marshal(req.Params)
	history := &database.CommandHistory{
		CommandID:       commandID,
		ClusterID:       req.ClusterID,
		Source:          req.Source,
		Action:          req.Action,
		TargetKind:      req.TargetKind,
		TargetNamespace: req.TargetNamespace,
		TargetName:      req.TargetName,
		Params:          string(paramsJSON),
		Status:          command.StatusPending,
		CreatedAt:       cmd.CreatedAt,
	}
	if err := s.cmdRepo.Create(context.Background(), history); err != nil {
		log.Error("指令历史持久化失败", "err", err)
	}

	return &model.CreateCommandResponse{
		CommandID: commandID,
		Status:    "pending",
	}, nil
}

// validateRequest 校验请求
func (s *CommandService) validateRequest(req *model.CreateCommandRequest) error {
	if req.ClusterID == "" {
		return fmt.Errorf("cluster_id required")
	}
	if req.Action == "" {
		return fmt.Errorf("action required")
	}

	// 校验 Action 类型（使用 model 中定义的有效动作）
	if !command.ValidActions[req.Action] {
		return fmt.Errorf("invalid action: %s", req.Action)
	}

	// 某些操作需要目标信息
	needsTarget := map[string]bool{
		command.ActionScale:       true,
		command.ActionRestart:     true,
		command.ActionDelete:      true,
		command.ActionDeletePod:   true,
		command.ActionCordon:      true,
		command.ActionUncordon:    true,
		command.ActionUpdateImage: true,
		command.ActionGetLogs:     true,
	}
	if needsTarget[req.Action] {
		if req.TargetKind == "" || req.TargetName == "" {
			return fmt.Errorf("target_kind and target_name required for action: %s", req.Action)
		}
	}

	return nil
}

// ExecuteCommandSync 创建指令并同步等待 Agent 执行结果
func (s *CommandService) ExecuteCommandSync(ctx context.Context, req *model.CreateCommandRequest, timeout time.Duration) (*command.Result, error) {
	resp, err := s.CreateCommand(req)
	if err != nil {
		return nil, fmt.Errorf("create command: %w", err)
	}
	result, err := s.bus.WaitCommandResult(ctx, resp.CommandID, timeout)
	if err != nil {
		return nil, fmt.Errorf("wait command %s: %w", resp.CommandID, err)
	}
	return result, nil
}
