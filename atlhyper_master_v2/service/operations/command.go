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
	bus      mq.Producer
	cmdRepo  database.CommandHistoryRepository
	presence ClusterPresence
}

// ClusterPresence 判断集群是否有在线的 Agent
type ClusterPresence interface {
	IsClusterOnline(clusterID string) bool
}

// NewCommandService 创建 CommandService
func NewCommandService(bus mq.Producer, cmdRepo database.CommandHistoryRepository, presence ClusterPresence) *CommandService {
	return &CommandService{
		bus:      bus,
		cmdRepo:  cmdRepo,
		presence: presence,
	}
}

// CreateCommand 创建指令
func (s *CommandService) CreateCommand(req *model.CreateCommandRequest) (*model.CreateCommandResponse, error) {
	// 1. 校验
	if err := s.validateRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %w", model.ErrInvalidCommand, err)
	}
	// 没有在线 Agent 的集群不建指令：发了也没人取，任意集群 ID 也不会在队列里留下东西
	if !s.presence.IsClusterOnline(req.ClusterID) {
		return nil, fmt.Errorf("%w: %s", model.ErrClusterOffline, req.ClusterID)
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

	// 4. 先持久化指令历史（只写需要留痕的指令，见 command.RecordsHistory）。
	// 指令一入队 Agent 就可能立刻执行并回报结果，回报时要找得到这条历史；
	// 若先入队后写历史，结果先到就对不上，历史会永远停在 pending。
	var history *database.CommandHistory
	var historyErr error
	if command.RecordsHistory(req.Action, req.Source) {
		paramsJSON, _ := json.Marshal(req.Params)
		history = &database.CommandHistory{
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
		historyErr = s.cmdRepo.Create(context.Background(), history)
		if historyErr != nil {
			log.Error("指令历史持久化失败", "err", historyErr)
		}
	}

	// 5. 写入 MQ（按来源路由 topic）
	topic := mq.TopicOps
	if req.Source == command.SourceAI {
		topic = mq.TopicAI
	}
	if err := s.bus.EnqueueCommand(req.ClusterID, topic, cmd); err != nil {
		if history != nil && historyErr == nil {
			s.markHistoryFailed(history, err)
		}
		return nil, fmt.Errorf("%w: %w", model.ErrEnqueueFailed, err)
	}

	return &model.CreateCommandResponse{
		CommandID: commandID,
		Status:    "pending",
	}, nil
}

// PurgeHistoryBefore 删除早于 before 的指令历史，返回删除的行数
func (s *CommandService) PurgeHistoryBefore(ctx context.Context, before time.Time) (int64, error) {
	return s.cmdRepo.DeleteBefore(ctx, before)
}

// markHistoryFailed 指令没能入队时，把已写下的历史标成失败
func (s *CommandService) markHistoryFailed(history *database.CommandHistory, cause error) {
	now := time.Now()
	history.Status = command.StatusFailed
	history.ErrorMessage = "指令未能入队: " + cause.Error()
	history.FinishedAt = &now
	if err := s.cmdRepo.Update(context.Background(), history); err != nil {
		log.Error("更新指令历史失败", "cmd", history.CommandID, "err", err)
	}
}

// validateRequest 校验请求
func (s *CommandService) validateRequest(req *model.CreateCommandRequest) error {
	if req.ClusterID == "" {
		return fmt.Errorf("cluster_id required")
	}
	if req.Action == "" {
		return fmt.Errorf("action required")
	}

	// 「来源 × 动作」必须在允许名单里（Agent 执行前按同一份名单再核对一次）
	if err := command.CheckSource(req.Action, req.Source); err != nil {
		return err
	}

	// 目标的命名空间 / 名字必须是合法的 K8s 资源名
	if err := command.ValidateTarget(req.TargetNamespace, req.TargetName); err != nil {
		return err
	}

	// 某些操作需要目标信息
	needsTarget := map[string]bool{
		command.ActionScale:       true,
		command.ActionRestart:     true,
		command.ActionDelete:      true,
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
		return nil, err
	}
	result, err := s.bus.WaitCommandResult(ctx, resp.CommandID, timeout)
	if err != nil {
		return nil, fmt.Errorf("wait command %s: %w", resp.CommandID, err)
	}
	if result == nil {
		// 总线约定 err 为 nil 时一定有结果；守住这个约定，调用方不必再判空
		return nil, fmt.Errorf("wait command %s: %w", resp.CommandID, mq.ErrWaitTimeout)
	}
	return result, nil
}
