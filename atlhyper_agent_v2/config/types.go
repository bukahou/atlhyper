// atlhyper_agent_v2/config/types.go
// 配置结构体定义
package config

import "time"

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	SnapshotInterval    time.Duration // 快照采集间隔
	CommandPollInterval time.Duration // 指令轮询间隔
	HeartbeatInterval   time.Duration // 心跳间隔
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	HTTPClient      time.Duration // HTTP 客户端超时 (含长轮询)
	SnapshotCollect time.Duration // 快照采集操作超时
	CommandPoll     time.Duration // 指令轮询操作超时
	Heartbeat       time.Duration // 心跳操作超时
}

// MasterConfig Master 通信配置
type MasterConfig struct {
	URL string // Master 服务地址
}

// KubernetesConfig Kubernetes 连接配置
type KubernetesConfig struct {
	KubeConfig string // kubeconfig 文件路径，空则使用 InCluster 模式
}

// AgentConfig Agent 基础配置
type AgentConfig struct {
	ClusterID string // 集群唯一标识
}

// AppConfig Agent 顶层配置结构体
type AppConfig struct {
	Agent      AgentConfig
	Master     MasterConfig
	Kubernetes KubernetesConfig
	Scheduler  SchedulerConfig
	Timeout    TimeoutConfig
}

// GlobalConfig 全局配置实例
var GlobalConfig AppConfig
