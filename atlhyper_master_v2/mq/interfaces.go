// atlhyper_master_v2/mq/interfaces.go
// CommandBus 接口定义
// 按调用方拆分为 Producer (上层) 和 Consumer (AgentSDK)
package mq

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
)

// Topic 常量
// 每个 Cluster 有多个 Topic 队列，互不阻塞
const (
	TopicOps = "ops" // 系统操作（Web UI 发起的 scale, restart 等）
	TopicAI  = "ai"  // AI 指令（AI 发起的查询/操作）
)

// Producer 指令发送端 (上层: Gateway/Service 使用)
type Producer interface {
	// EnqueueCommand 入队指令到指定 topic
	EnqueueCommand(clusterID, topic string, cmd *model.Command) error

	// GetCommandStatus 获取指令状态
	GetCommandStatus(cmdID string) (*model.CommandStatus, error)

	// WaitCommandResult 等待指令执行完成（同步等待）
	// 阻塞直到 Agent 上报结果、超时、或 ctx 取消
	WaitCommandResult(ctx context.Context, cmdID string, timeout time.Duration) (*model.CommandResult, error)
}

// Consumer 指令消费端 (下层: AgentSDK 使用)
type Consumer interface {
	// WaitCommand 等待指定 topic 的指令（长轮询）
	// Agent 为每个 topic 开独立 goroutine 轮询
	WaitCommand(ctx context.Context, clusterID, topic string, timeout time.Duration) (*model.Command, error)

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
