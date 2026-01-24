// atlhyper_master_v2/datahub/interfaces.go
// Store 接口定义
// Store 负责快照存储、Agent 状态和 Event 查询
package datahub

import (
	"time"

	"AtlHyper/model_v2"
)

// Store 数据存储接口
// 抽象层，底层可替换为 MemoryStore / RedisStore
type Store interface {
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

	// ==================== Event 查询 ====================

	// GetEvents 获取集群当前所有 Events
	GetEvents(clusterID string) ([]model_v2.Event, error)

	// ==================== 生命周期 ====================

	// Start 启动 Store
	Start() error

	// Stop 停止 Store
	Stop() error
}

// Config Store 配置
type Config struct {
	Type            string        // 类型: memory / redis
	EventRetention  time.Duration // Event 保留时间
	HeartbeatExpire time.Duration // 心跳过期时间

	// Redis 配置（Type=redis 时使用）
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}
