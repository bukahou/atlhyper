// atlhyper_master_v2/mq/interfaces.go
// CommandBus 接口定义
// 按调用方拆分为 Producer (上层) 和 Consumer (AgentSDK)
package mq

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
)

// Producer 指令发送端 (上层: Gateway/Service 使用)
type Producer interface {
	// EnqueueCommand 入队指令
	EnqueueCommand(clusterID string, cmd *model.Command) error

	// GetCommandStatus 获取指令状态
	GetCommandStatus(cmdID string) (*model.CommandStatus, error)

	// WaitCommandResult 等待指令执行完成（同步等待）
	// 阻塞直到 Agent 上报结果或超时
	WaitCommandResult(cmdID string, timeout time.Duration) (*model.CommandResult, error)
}

// Consumer 指令消费端 (下层: AgentSDK 使用)
type Consumer interface {
	// WaitCommand 等待指令（长轮询）
	// 阻塞等待直到有指令或超时
	WaitCommand(ctx context.Context, clusterID string, timeout time.Duration) (*model.Command, error)

	// AckCommand 确认指令完成
	AckCommand(cmdID string, result *model.CommandResult) error
}

// CommandBus 完整接口 (工厂创建 + 生命周期管理)
type CommandBus interface {
	Producer
	Consumer

	// Start 启动 CommandBus
	Start() error

	// Stop 停止 CommandBus
	Stop() error
}
