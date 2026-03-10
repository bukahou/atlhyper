// atlhyper_master_v2/config/types.go
// 配置结构体定义
package config

import "time"

// ServerConfig 服务器配置
type ServerConfig struct {
	GatewayPort  int // Gateway 端口（Web/API）
	AgentSDKPort int // AgentSDK 端口（Agent 数据上报）
	TesterPort   int // Tester 端口（测试服务）
}

// DataHubConfig DataHub 配置
type DataHubConfig struct {
	Type              string        // 类型: memory / redis
	EventRetention    time.Duration // Event 保留时间（30 分钟）
	SnapshotRetain    int           // 快照保留数量
	HeartbeatExpire   time.Duration // 心跳过期时间
	SnapshotRetention time.Duration // OTel 快照时间线保留时间（默认 15min）
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string // 类型: sqlite / mysql
	Path     string // SQLite 路径
	DSN      string // MySQL/PG 连接串
	MaxConns int    // 最大连接数
}

// EventConfig Event 持久化配置
type EventConfig struct {
	RetentionDays   int           // 保留天数
	MaxCount        int           // 单集群最大事件数
	CleanupInterval time.Duration // 清理检查间隔
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	CommandPoll time.Duration // 长轮询超时
	Heartbeat   time.Duration // 心跳超时阈值
}

// JWTConfig JWT 配置
type JWTConfig struct {
	SecretKey   string        // JWT 密钥
	TokenExpiry time.Duration // Token 有效期
}

// AdminConfig 默认管理员配置
type AdminConfig struct {
	Username    string // 管理员用户名
	Password    string // 管理员密码（明文，启动时会 bcrypt 加密）
	DisplayName string // 显示名称
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string // 地址 (host:port)
	Password string // 密码（空为无密码）
	DB       int    // 数据库编号
}

// AIConfig AI 功能配置
// ToolTimeout 用于首次启动时初始化 ai_settings 表
// 角色分配即生效：Provider 有角色就使用，无角色就不使用
type AIConfig struct {
	Enabled     bool          // 已废弃（保留用于配置兼容，不再影响逻辑）
	ToolTimeout time.Duration // Tool 执行超时（默认 30s）
	Seed        AISeed        // 种子配置：首次启动时自动创建 Provider
}

// AISeed AI 种子配置
// 通过环境变量注入，首次启动时自动创建 AI Provider 并激活
// 已有 Provider 数据时跳过（不覆盖 Web UI 配置）
type AISeed struct {
	Provider string // 提供商类型: gemini / openai / anthropic / ollama
	Name     string // 显示名称
	APIKey   string // API Key（Ollama 可为空）
	Model    string // 模型名称
	BaseURL  string // 自定义 API 地址（Ollama 使用）
}

// EventAlertConfig 事件告警配置
type EventAlertConfig struct {
	Enabled       bool          // 是否启用事件告警
	CheckInterval time.Duration // 检测间隔
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string // 日志级别: debug / info / warn / error (默认 info)
	Format string // 日志格式: text / json (默认 text)
}

// SLOConfig SLO 监控配置
// Master 端 SLO 功能始终启用，无需开关配置
// 如果 Agent 未上报数据，前端显示"暂无数据"即可
type SLOConfig struct {
	AggregateInterval time.Duration // 聚合间隔（默认 1h）
	CleanupInterval   time.Duration // 清理间隔（默认 1h）
	RawRetention      time.Duration // raw 数据保留时间（默认 48h）
	HourlyRetention   time.Duration // hourly 数据保留时间（默认 90d）
	StatusRetention   time.Duration // 状态历史保留时间（默认 180d）
}

// MetricsPersistConfig 节点指标持久化配置
type MetricsPersistConfig struct {
	SampleInterval  time.Duration // 历史数据采样间隔（默认 30s）
	RetentionDays   int           // 历史数据保留天数（默认 30 天）
	CleanupInterval time.Duration // 清理检查间隔（默认 1h）
}

// AIOpsConfig AIOps 引擎配置
type AIOpsConfig struct {
	Enable        bool          // 是否启用 AIOps 引擎（默认 true）
	FlushInterval time.Duration // 基线状态 flush 间隔（默认 5min）
}

// GitHubConfig GitHub App 配置
type GitHubConfig struct {
	AppID          int64  // GitHub App ID
	AppSlug        string // GitHub App URL slug (e.g. "atlhyper")
	PrivateKeyPath string // Private Key PEM 文件路径
	CallbackURL    string // App 安装回调 URL
}

// AppConfig Master 顶层配置结构体
type AppConfig struct {
	Log            LogConfig
	Server         ServerConfig
	DataHub        DataHubConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	Event          EventConfig
	EventAlert     EventAlertConfig
	Timeout        TimeoutConfig
	JWT            JWTConfig
	Admin          AdminConfig
	AI             AIConfig
	SLO            SLOConfig
	MetricsPersist MetricsPersistConfig
	AIOps          AIOpsConfig
	GitHub         GitHubConfig
}

// GlobalConfig 全局配置实例
var GlobalConfig AppConfig
