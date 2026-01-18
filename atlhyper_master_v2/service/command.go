// atlhyper_master_v2/service/command.go
// CommandService 指令写入服务
// 负责接收 Web/AI 的指令请求，校验后写入 DataHub MQ
package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/model"
)

// CommandService 指令服务接口
type CommandService interface {
	// CreateCommand 创建指令（写入 DataHub MQ）
	CreateCommand(req *CreateCommandRequest) (*CreateCommandResponse, error)
}

// commandServiceImpl CommandService 实现
type commandServiceImpl struct {
	datahub datahub.DataHub
}

// NewCommandService 创建 CommandService
func NewCommandService(dh datahub.DataHub) CommandService {
	return &commandServiceImpl{
		datahub: dh,
	}
}

// CreateCommandRequest 创建指令请求
type CreateCommandRequest struct {
	ClusterID       string                 `json:"cluster_id"`
	Action          string                 `json:"action"` // scale / restart / delete_pod / exec ...
	TargetKind      string                 `json:"target_kind,omitempty"`
	TargetNamespace string                 `json:"target_namespace,omitempty"`
	TargetName      string                 `json:"target_name,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
	Source          string                 `json:"source,omitempty"` // web / ai
}

// CreateCommandResponse 创建指令响应
type CreateCommandResponse struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
}

// CreateCommand 创建指令
func (s *commandServiceImpl) CreateCommand(req *CreateCommandRequest) (*CreateCommandResponse, error) {
	// 1. 校验
	if err := s.validateRequest(req); err != nil {
		return nil, fmt.Errorf("validate request: %w", err)
	}

	// 2. 生成指令 ID
	commandID := uuid.New().String()

	// 3. 构建 Command
	cmd := &model.Command{
		ID:              commandID,
		ClusterID:       req.ClusterID,
		Action:          req.Action,
		TargetKind:      req.TargetKind,
		TargetNamespace: req.TargetNamespace,
		TargetName:      req.TargetName,
		Params:          req.Params,
		CreatedAt:       time.Now(),
	}

	// 4. 写入 DataHub MQ
	if err := s.datahub.EnqueueCommand(req.ClusterID, cmd); err != nil {
		return nil, fmt.Errorf("enqueue command: %w", err)
	}

	return &CreateCommandResponse{
		CommandID: commandID,
		Status:    "pending",
	}, nil
}

// validateRequest 校验请求
func (s *commandServiceImpl) validateRequest(req *CreateCommandRequest) error {
	if req.ClusterID == "" {
		return fmt.Errorf("cluster_id required")
	}
	if req.Action == "" {
		return fmt.Errorf("action required")
	}

	// 校验 Action 类型（使用 model 中定义的有效动作）
	if !model.ValidActions[req.Action] {
		return fmt.Errorf("invalid action: %s", req.Action)
	}

	// 某些操作需要目标信息
	needsTarget := map[string]bool{
		model.ActionScale:       true,
		model.ActionRestart:     true,
		model.ActionDelete:      true,
		model.ActionDeletePod:   true,
		model.ActionCordon:      true,
		model.ActionUncordon:    true,
		model.ActionUpdateImage: true,
		model.ActionGetLogs:     true,
	}
	if needsTarget[req.Action] {
		if req.TargetKind == "" || req.TargetName == "" {
			return fmt.Errorf("target_kind and target_name required for action: %s", req.Action)
		}
	}

	return nil
}
