// atlhyper_agent_v2/config/types.go
// 配置结构体定义：环境变量名与默认值写在字段标签上（caarlos0/env），这里是唯一出处。
package config

import "time"

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	SnapshotInterval    time.Duration `env:"AGENT_SNAPSHOT_INTERVAL" envDefault:"10s"`       // 快照采集间隔
	CommandPollInterval time.Duration `env:"AGENT_COMMAND_POLL_INTERVAL" envDefault:"100ms"` // 指令轮询间隔
	HeartbeatInterval   time.Duration `env:"AGENT_HEARTBEAT_INTERVAL" envDefault:"15s"`      // 心跳间隔
	OTelCacheTTL        time.Duration `env:"AGENT_OTEL_CACHE_TTL" envDefault:"10s"`          // OTel 概览缓存 TTL（与快照间隔一致）
	OTelDashboardTTL    time.Duration `env:"AGENT_OTEL_DASHBOARD_TTL" envDefault:"30s"`      // OTel Dashboard 列表缓存 TTL（dev 可放宽以减轻共用 ClickHouse 压力）
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	HTTPClient      time.Duration `env:"AGENT_TIMEOUT_HTTP_CLIENT" envDefault:"90s"`      // HTTP 客户端超时（含长轮询，需大于 Master 长轮询超时）
	SnapshotCollect time.Duration `env:"AGENT_TIMEOUT_SNAPSHOT_COLLECT" envDefault:"30s"` // 快照采集操作超时
	CommandPoll     time.Duration `env:"AGENT_TIMEOUT_COMMAND_POLL" envDefault:"60s"`     // 指令轮询操作超时（长轮询）
	CommandExec     time.Duration `env:"AGENT_TIMEOUT_COMMAND_EXEC" envDefault:"60s"`     // 单条指令执行超时（独立于长轮询）
	CommandReport   time.Duration `env:"AGENT_TIMEOUT_COMMAND_REPORT" envDefault:"10s"`   // 指令结果回报超时
	Heartbeat       time.Duration `env:"AGENT_TIMEOUT_HEARTBEAT" envDefault:"10s"`        // 心跳操作超时
}

// MasterConfig Master 通信配置
type MasterConfig struct {
	URL   string `env:"AGENT_MASTER_URL" envDefault:"http://localhost:8081"` // Master AgentSDK 端口（非 Gateway 端口）
	Token string `env:"AGENT_MASTER_TOKEN,required,notEmpty"`                // 集群令牌（Master 签发，必填）。⛔ 凭证，无默认值，不得写入日志
}

// KubernetesConfig Kubernetes 连接配置
type KubernetesConfig struct {
	KubeConfig string `env:"AGENT_KUBECONFIG"` // kubeconfig 文件路径；空则取 KUBECONFIG，再空则使用 InCluster 模式
}

// AgentConfig Agent 基础配置
type AgentConfig struct {
	ClusterID  string `env:"AGENT_CLUSTER_ID"`                     // 集群唯一标识；空则自动获取集群 UID
	HealthAddr string `env:"AGENT_HEALTH_ADDR" envDefault:":8082"` // 健康检查服务的监听地址（探针用）
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `env:"AGENT_LOG_LEVEL" envDefault:"info"`  // 日志级别: debug / info / warn / error
	Format string `env:"AGENT_LOG_FORMAT" envDefault:"text"` // 日志格式: text / json
}

// ClickHouseConfig ClickHouse 连接配置
type ClickHouseConfig struct {
	Endpoint string        `env:"AGENT_CLICKHOUSE_ENDPOINT"`                       // ClickHouse 地址，如 "clickhouse://localhost:9000"；为空则不连接（纯 K8s 模式）
	Database string        `env:"AGENT_CLICKHOUSE_DATABASE" envDefault:"atlhyper"` // 数据库名
	Timeout  time.Duration `env:"AGENT_CLICKHOUSE_TIMEOUT" envDefault:"10s"`       // 连接 / 查询超时
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
