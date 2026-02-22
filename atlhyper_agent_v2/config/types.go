// atlhyper_agent_v2/config/types.go
// 配置结构体定义
package config

import "time"

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	SnapshotInterval    time.Duration // 快照采集间隔
	CommandPollInterval time.Duration // 指令轮询间隔
	HeartbeatInterval   time.Duration // 心跳间隔
	OTelCacheTTL        time.Duration // OTel 概览缓存 TTL (默认 5m)
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

// LogConfig 日志配置
type LogConfig struct {
	Level  string // 日志级别: debug / info / warn / error (默认 info)
	Format string // 日志格式: text / json (默认 text)
}

// ClickHouseConfig ClickHouse 连接配置
type ClickHouseConfig struct {
	Endpoint string        // ClickHouse 地址，如 "clickhouse://localhost:9000"
	Database string        // 数据库名
	Timeout  time.Duration // 连接/查询超时
}

// AppConfig Agent 顶层配置结构体
type AppConfig struct {
	Log        LogConfig
	Agent      AgentConfig
	Master     MasterConfig
	Kubernetes KubernetesConfig
	Scheduler  SchedulerConfig
	Timeout    TimeoutConfig
	ClickHouse ClickHouseConfig
}

// GlobalConfig 全局配置实例
var GlobalConfig AppConfig
