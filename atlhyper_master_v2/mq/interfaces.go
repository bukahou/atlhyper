// atlhyper_master_v2/mq/interfaces.go
// CommandBus 接口定义
// 消息队列，负责指令的入队、等待、确认和结果获取
package mq

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
)

// CommandBus 消息队列接口
// 抽象层，底层可替换为 MemoryBus / RedisBus
type CommandBus interface {
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

	// Start 启动 CommandBus
	Start() error

	// Stop 停止 CommandBus
	Stop() error
}
