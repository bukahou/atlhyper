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
	Type            string        // 类型: memory / redis
	EventRetention  time.Duration // Event 保留时间（30 分钟）
	SnapshotRetain  int           // 快照保留数量
	HeartbeatExpire time.Duration // 心跳过期时间
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

// NotifierConfig 通知配置
type NotifierConfig struct {
	// 邮件配置
	Mail MailConfig

	// Webhook 配置（可选）
	Webhook WebhookConfig
}

// MailConfig 邮件配置
type MailConfig struct {
	Enabled  bool   // 是否启用邮件通知
	SMTPHost string // SMTP 服务器地址
	SMTPPort int    // SMTP 端口（587 for TLS, 465 for SSL）
	Username string // SMTP 用户名
	Password string // SMTP 密码
	From     string // 发件人地址
	To       string // 收件人列表，逗号分隔
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	Enabled bool   // 是否启用 Webhook
	URL     string // Webhook URL
	Secret  string // 签名密钥（可选）
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
type AIConfig struct {
	Enabled     bool          // 是否启用 AI 功能
	Provider    string        // LLM 提供商: gemini
	APIKey      string        // API Key
	Model       string        // 模型名称
	ToolTimeout time.Duration // Tool 执行超时
}

// AppConfig Master 顶层配置结构体
type AppConfig struct {
	Server   ServerConfig
	DataHub  DataHubConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Event    EventConfig
	Timeout  TimeoutConfig
	JWT      JWTConfig
	Notifier NotifierConfig
	Admin    AdminConfig
	AI       AIConfig
}

// GlobalConfig 全局配置实例
var GlobalConfig AppConfig
