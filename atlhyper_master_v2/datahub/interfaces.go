// atlhyper_master_v2/datahub/interfaces.go
// DataHub 接口定义
// DataHub 是实时数据中心，负责快照存储、Agent 状态和指令队列
package datahub

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// DataHub 数据中心接口
// 抽象层，底层可替换为 MemoryHub / RedisHub
type DataHub interface {
	// ==================== 快照管理 ====================

	// SetSnapshot 存储集群快照
	SetSnapshot(clusterID string, snapshot *model_v2.ClusterSnapshot) error

	// GetSnapshot 获取集群快照
	GetSnapshot(clusterID string) (*model_v2.ClusterSnapshot, error)

	// ==================== Agent 状态 ====================

	// UpdateHeartbeat 更新 Agent 心跳
	UpdateHeartbeat(clusterID string) error

	// GetAgentStatus 获取 Agent 状态
	GetAgentStatus(clusterID string) (*model_v2.AgentStatus, error)

	// ListAgents 列出所有 Agent
	ListAgents() ([]model_v2.AgentInfo, error)

	// ==================== 指令队列（简易 MQ）====================

	// EnqueueCommand 入队指令
	EnqueueCommand(clusterID string, cmd *model.Command) error

	// WaitCommand 等待指令（长轮询）
	// 阻塞等待直到有指令或超时
	WaitCommand(ctx context.Context, clusterID string, timeout time.Duration) (*model.Command, error)

	// AckCommand 确认指令完成
	AckCommand(cmdID string, result *model.CommandResult) error

	// GetCommandStatus 获取指令状态
	GetCommandStatus(cmdID string) (*model.CommandStatus, error)

	// WaitCommandResult 等待指令执行完成（同步等待）
	// 阻塞直到 Agent 上报结果或超时
	WaitCommandResult(cmdID string, timeout time.Duration) (*model.CommandResult, error)

	// ==================== Event 查询（用于持久化）====================

	// GetEvents 获取集群当前所有 Events
	GetEvents(clusterID string) ([]model_v2.Event, error)

	// ==================== 生命周期 ====================

	// Start 启动 DataHub
	Start() error

	// Stop 停止 DataHub
	Stop() error
}
