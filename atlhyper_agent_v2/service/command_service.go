// Package service 业务逻辑层
//
// command_service.go - 指令执行服务
//
// 本文件实现 CommandService 接口，负责执行 Master 下发的指令。
//
// 支持的指令类型 (Action):
//   - scale: 扩缩容 Deployment
//   - restart: 重启 Deployment (滚动重启)
//   - update_image: 更新容器镜像
//   - delete: 删除资源 (Pod 或通用资源)
//   - get_logs: 获取 Pod 日志
//   - cordon: 封锁节点
//   - uncordon: 解封节点
//   - dynamic: 动态 API 调用 (AI 只读查询)
//
// 执行流程:
//  1. 根据 Action 分发到对应的 handler
//  2. 解析 Params 中的参数
//  3. 调用 Repository 执行操作
//  4. 封装 Result 返回
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
)

// commandService 指令执行服务实现
//
// 核心依赖:
//   - podRepo: Pod 查询 (日志获取)
//   - genericRepo: 所有写操作 + 动态查询
type commandService struct {
	podRepo     repository.PodRepository
	genericRepo repository.GenericRepository
}

// NewCommandService 创建指令服务
func NewCommandService(
	podRepo repository.PodRepository,
	genericRepo repository.GenericRepository,
) CommandService {
	return &commandService{
		podRepo:     podRepo,
		genericRepo: genericRepo,
	}
}

// Execute 执行指令
//
// 根据 Command.Action 分发到对应的处理函数。
// 无论成功与否，都返回 Result，不返回 error。
//
// 返回值:
//   - Success=true: 执行成功，Data 包含返回数据 (如日志内容)
//   - Success=false: 执行失败，Error 包含错误信息
func (s *commandService) Execute(ctx context.Context, cmd *model.Command) *model.Result {
	result := &model.Result{
		CommandID:  cmd.ID,
		ExecutedAt: time.Now(),
	}

	var err error
	var data any

	switch cmd.Action {
	case model.ActionScale:
		err = s.handleScale(ctx, cmd)
	case model.ActionRestart:
		err = s.handleRestart(ctx, cmd)
	case model.ActionUpdateImage:
		err = s.handleUpdateImage(ctx, cmd)
	case model.ActionGetLogs:
		data, err = s.handleGetLogs(ctx, cmd)
	case model.ActionGetConfigMap:
		data, err = s.handleGetConfigMap(ctx, cmd)
	case model.ActionGetSecret:
		data, err = s.handleGetSecret(ctx, cmd)
	case model.ActionDynamic:
		data, err = s.handleDynamic(ctx, cmd)
	case model.ActionDelete:
		err = s.handleDelete(ctx, cmd)
	case model.ActionCordon:
		err = s.handleCordon(ctx, cmd)
	case model.ActionUncordon:
		err = s.handleUncordon(ctx, cmd)
	default:
		err = fmt.Errorf("unknown action: %s", cmd.Action)
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		if data != nil {
			// 将返回数据转换为字符串
			switch v := data.(type) {
			case string:
				result.Output = v
			default:
				// 其他类型 JSON 序列化
				if b, e := json.Marshal(v); e == nil {
					result.Output = string(b)
				}
			}
		}
	}

	return result
}

// handleScale 处理扩缩容指令
func (s *commandService) handleScale(ctx context.Context, cmd *model.Command) error {
	var params struct {
		Replicas int32 `json:"replicas"`
	}
	if err := s.parseParams(cmd.Params, &params); err != nil {
		return fmt.Errorf("invalid scale params: %w", err)
	}

	return s.genericRepo.ScaleDeployment(ctx, cmd.Namespace, cmd.Name, params.Replicas)
}

// handleRestart 处理重启指令
func (s *commandService) handleRestart(ctx context.Context, cmd *model.Command) error {
	return s.genericRepo.RestartDeployment(ctx, cmd.Namespace, cmd.Name)
}

// handleUpdateImage 处理更新镜像指令
func (s *commandService) handleUpdateImage(ctx context.Context, cmd *model.Command) error {
	var params struct {
		Container string `json:"container,omitempty"`
		Image     string `json:"image"`
	}
	if err := s.parseParams(cmd.Params, &params); err != nil {
		return fmt.Errorf("invalid update_image params: %w", err)
	}
	if params.Image == "" {
		return fmt.Errorf("image is required")
	}

	return s.genericRepo.UpdateDeploymentImage(ctx, cmd.Namespace, cmd.Name, params.Container, params.Image)
}

// handleCordon 处理封锁节点指令
func (s *commandService) handleCordon(ctx context.Context, cmd *model.Command) error {
	return s.genericRepo.CordonNode(ctx, cmd.Name)
}

// handleUncordon 处理解封节点指令
func (s *commandService) handleUncordon(ctx context.Context, cmd *model.Command) error {
	return s.genericRepo.UncordonNode(ctx, cmd.Name)
}

// handleGetLogs 处理获取日志指令
func (s *commandService) handleGetLogs(ctx context.Context, cmd *model.Command) (string, error) {
	var params struct {
		Container    string `json:"container,omitempty"`
		TailLines    int64  `json:"tailLines,omitempty"`
		SinceSeconds int64  `json:"sinceSeconds,omitempty"`
		Timestamps   bool   `json:"timestamps,omitempty"`
		Previous     bool   `json:"previous,omitempty"`
	}
	if cmd.Params != nil {
		if err := s.parseParams(cmd.Params, &params); err != nil {
			return "", fmt.Errorf("invalid log params: %w", err)
		}
	}

	return s.podRepo.GetLogs(ctx, cmd.Namespace, cmd.Name, model.LogOptions{
		Container:    params.Container,
		TailLines:    params.TailLines,
		SinceSeconds: params.SinceSeconds,
		Timestamps:   params.Timestamps,
		Previous:     params.Previous,
	})
}

// handleGetConfigMap 处理获取 ConfigMap 数据指令
func (s *commandService) handleGetConfigMap(ctx context.Context, cmd *model.Command) (map[string]string, error) {
	if cmd.Namespace == "" || cmd.Name == "" {
		return nil, fmt.Errorf("namespace and name are required")
	}
	return s.genericRepo.GetConfigMapData(ctx, cmd.Namespace, cmd.Name)
}

// handleGetSecret 处理获取 Secret 数据指令
func (s *commandService) handleGetSecret(ctx context.Context, cmd *model.Command) (map[string]string, error) {
	if cmd.Namespace == "" || cmd.Name == "" {
		return nil, fmt.Errorf("namespace and name are required")
	}
	return s.genericRepo.GetSecretData(ctx, cmd.Namespace, cmd.Name)
}

// handleDynamic 处理动态请求指令
//
// 用于 AI 只读查询 K8s API (仅 GET)
func (s *commandService) handleDynamic(ctx context.Context, cmd *model.Command) (*model.DynamicResponse, error) {
	var params struct {
		Path  string            `json:"path"`
		Query map[string]string `json:"query,omitempty"`
	}
	if err := s.parseParams(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid dynamic params: %w", err)
	}

	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	req := &model.DynamicRequest{
		Path:  params.Path,
		Query: params.Query,
	}

	return s.genericRepo.Execute(ctx, req)
}

// handleDelete 处理通用删除指令
func (s *commandService) handleDelete(ctx context.Context, cmd *model.Command) error {
	var params struct {
		GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty"`
		Force              bool   `json:"force,omitempty"`
	}
	if cmd.Params != nil {
		if err := s.parseParams(cmd.Params, &params); err != nil {
			return fmt.Errorf("invalid delete params: %w", err)
		}
	}

	opts := model.DeleteOptions{
		GracePeriodSeconds: params.GracePeriodSeconds,
		Force:              params.Force,
	}

	// Pod 使用专门的删除方法
	if cmd.Kind == "Pod" {
		return s.genericRepo.DeletePod(ctx, cmd.Namespace, cmd.Name, opts)
	}

	// 其他资源使用通用删除
	return s.genericRepo.Delete(ctx, cmd.Kind, cmd.Namespace, cmd.Name, opts)
}

// parseParams 解析参数
func (s *commandService) parseParams(params any, target any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
