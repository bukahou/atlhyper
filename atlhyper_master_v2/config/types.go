// atlhyper_master_v2/config/types.go
// 配置结构体定义：环境变量名、默认值、必填都写在字段标签上（caarlos0/env），这里是唯一出处。
package config

import "time"

// ServerConfig 服务器配置
type ServerConfig struct {
	GatewayPort  int `env:"MASTER_GATEWAY_PORT" envDefault:"8080"`  // Gateway 端口（Web/API）
	AgentSDKPort int `env:"MASTER_AGENTSDK_PORT" envDefault:"8081"` // AgentSDK 端口（Agent 数据上报）
	// PublicURL 用户从浏览器访问本系统的地址（如 https://atlhyper.example.com），用来在通知里生成页面链接。
	// 可空：不配时通知里不带链接
	PublicURL string `env:"MASTER_PUBLIC_URL"`
	// Timezone 通知消息里时间的显示时区（IANA 名，如 Asia/Tokyo）。只影响显示，存储与接口一律 UTC
	Timezone string `env:"MASTER_TIMEZONE" envDefault:"UTC"`
	// ClientIPHeader 最外层入口（如 CDN）覆盖写入真实客户端地址的头名（如 CF-Connecting-IP）。
	// 空 = 不信任任何头，取 TCP 对端地址；⛔ 不要配成 X-Forwarded-For（客户端可以伪造）
	ClientIPHeader string `env:"MASTER_GATEWAY_CLIENT_IP_HEADER"`
	// MaxBodyKB Web API 单个请求体的上限（KB）
	MaxBodyKB int `env:"MASTER_GATEWAY_MAX_BODY_KB" envDefault:"1024"`
	// RateLimitPerMinute 每个客户端地址每分钟的 Web API 请求上限（兜底）
	RateLimitPerMinute int `env:"MASTER_GATEWAY_RATE_LIMIT_PER_MINUTE" envDefault:"3000"`
	// LoginRateLimitPerMinute 每个客户端地址每分钟的登录尝试上限（密码登录与 OIDC）
	LoginRateLimitPerMinute int `env:"MASTER_GATEWAY_LOGIN_RATE_LIMIT_PER_MINUTE" envDefault:"20"`
}

// MaxRequestBodyBytes Web API 单个请求体的上限（字节）
func (s ServerConfig) MaxRequestBodyBytes() int64 { return int64(s.MaxBodyKB) << 10 }

// AgentSDKConfig Agent 入口（AgentSDK）的认证与限额配置
type AgentSDKConfig struct {
	AuthMode      string `env:"MASTER_AGENT_AUTH_MODE" envDefault:"warn"`         // Agent 令牌认证模式: off / warn / enforce（启动时校验，未知值启动失败）
	MaxSnapshotMB int    `env:"MASTER_AGENTSDK_MAX_SNAPSHOT_MB" envDefault:"256"` // 单份快照的大小上限（MB，压缩前后都适用）
}

// DataHubConfig DataHub 配置
type DataHubConfig struct {
	EventRetention    time.Duration `env:"MASTER_DATAHUB_EVENT_RETENTION" envDefault:"30m"`    // Event 保留时间
	HeartbeatExpire   time.Duration `env:"MASTER_DATAHUB_HEARTBEAT_EXPIRE" envDefault:"45s"`   // 心跳过期时间
	SnapshotRetention time.Duration `env:"MASTER_DATAHUB_SNAPSHOT_RETENTION" envDefault:"15m"` // OTel 快照时间线保留时间
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type string `env:"MASTER_DB_TYPE" envDefault:"sqlite"`                                            // 类型：目前只实现了 sqlite
	Path string `env:"MASTER_DB_PATH" envDefault:"atlhyper_master_v2/database/sqlite/data/master.db"` // SQLite 路径

	CommandHistoryRetention time.Duration `env:"MASTER_COMMAND_HISTORY_RETENTION" envDefault:"2160h"` // 90 天
	AuditRetention          time.Duration `env:"MASTER_AUDIT_RETENTION" envDefault:"4320h"`           // 180 天
	AIOpsIncidentRetention  time.Duration `env:"MASTER_AIOPS_INCIDENT_RETENTION" envDefault:"4320h"`  // 180 天；只删已恢复的事件，子表随外键级联
	AIReportRetention       time.Duration `env:"MASTER_AI_REPORT_RETENTION" envDefault:"4320h"`       // 180 天
}

// EventConfig Event 持久化配置
type EventConfig struct {
	RetentionDays   int           `env:"MASTER_EVENT_RETENTION_DAYS" envDefault:"30"`   // 保留天数
	MaxCount        int           `env:"MASTER_EVENT_MAX_COUNT" envDefault:"100000"`    // 单集群最大事件数
	CleanupInterval time.Duration `env:"MASTER_EVENT_CLEANUP_INTERVAL" envDefault:"1h"` // 清理检查间隔
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	CommandPoll time.Duration `env:"MASTER_TIMEOUT_COMMAND_POLL" envDefault:"60s"` // 长轮询超时
}

// JWTConfig JWT 配置
type JWTConfig struct {
	SecretKey   string        `env:"MASTER_JWT_SECRET,required,notEmpty"`      // JWT 密钥（凭证：无默认值）
	TokenExpiry time.Duration `env:"MASTER_JWT_TOKEN_EXPIRY" envDefault:"24h"` // Token 有效期
}

// AdminConfig 默认管理员配置
type AdminConfig struct {
	Username    string `env:"MASTER_ADMIN_USERNAME,required,notEmpty"` // 管理员用户名
	Password    string `env:"MASTER_ADMIN_PASSWORD,required,notEmpty"` // 管理员密码（明文，启动时会 bcrypt 加密）
	DisplayName string `env:"MASTER_ADMIN_DISPLAY_NAME"`               // 显示名称
}

// AIConfig AI 功能配置
// ToolTimeout 用于首次启动时初始化 ai_settings 表
// 角色分配即生效：Provider 有角色就使用，无角色就不使用
type AIConfig struct {
	ToolTimeout time.Duration `env:"MASTER_AI_TOOL_TIMEOUT" envDefault:"30s"` // Tool 执行超时
	Seed        AISeed        // 种子配置：首次启动时自动创建 Provider

	BedrockRegion string `env:"MASTER_AI_BEDROCK_REGION" envDefault:"ap-northeast-1"` // AWS Bedrock 区域（所有 bedrock provider 共用）

	// GitHubPrivateRepos AI 的 GitHub 工具可读的私有仓库（owner/repo，逗号分隔）；空 = 只读公开仓库。
	// ⛔ 不要放存有凭据的仓库（如部署配置仓库）
	GitHubPrivateRepos CSVList `env:"MASTER_AI_GITHUB_PRIVATE_REPOS"`
}

// AISeed AI 种子配置
// 通过环境变量注入，首次启动时自动创建 AI Provider 并激活
// 已有 Provider 数据时跳过（不覆盖 Web UI 配置）
type AISeed struct {
	Provider string `env:"MASTER_AI_SEED_PROVIDER"` // 提供商类型: gemini / openai / anthropic / ollama；为空则不初始化
	Name     string `env:"MASTER_AI_SEED_NAME"`     // 显示名称
	APIKey   string `env:"MASTER_AI_SEED_API_KEY"`  // API Key（Ollama 可为空）
	Model    string `env:"MASTER_AI_SEED_MODEL"`    // 模型名称
	BaseURL  string `env:"MASTER_AI_SEED_BASE_URL"` // 自定义 API 地址（Ollama 使用）
}

// EventAlertConfig 旧的逐条 K8s 事件告警的配置。
// 是否开启不在这里：由通知策略里的 k8sEventAlertsEnabled 控制（设置页可改，默认关）
type EventAlertConfig struct {
	CheckInterval time.Duration `env:"MASTER_EVENT_ALERT_INTERVAL" envDefault:"30s"` // 轮询间隔
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `env:"MASTER_LOG_LEVEL" envDefault:"info"`  // 日志级别: debug / info / warn / error
	Format string `env:"MASTER_LOG_FORMAT" envDefault:"text"` // 日志格式: text / json
}

// GitHubConfig GitHub App 配置
type GitHubConfig struct {
	AppID          int64  `env:"GITHUB_APP_ID" envDefault:"0"`                                                // GitHub App ID
	AppSlug        string `env:"GITHUB_APP_SLUG"`                                                             // GitHub App URL slug (e.g. "atlhyper")
	PrivateKeyPath string `env:"GITHUB_PRIVATE_KEY_PATH"`                                                     // Private Key PEM 文件路径
	CallbackURL    string `env:"GITHUB_CALLBACK_URL" envDefault:"http://localhost:3000/auth/github/callback"` // App 安装回调 URL
}

// OIDCConfig OIDC 登录配置（issuer 为空 = 关闭）
type OIDCConfig struct {
	Issuer       string `env:"MASTER_OIDC_ISSUER"`                          // issuer URL
	ClientID     string `env:"MASTER_OIDC_CLIENT_ID"`                       // 在 issuer 处登记的 client_id（上线后不可改：pairwise sub 按它派生）
	ClientSecret string `env:"MASTER_OIDC_CLIENT_SECRET"`                   // 凭证：无默认值
	RedirectURL  string `env:"MASTER_OIDC_REDIRECT_URL"`                    // https://<web-host>/api/v2/auth/oidc/callback，须与登记逐字一致
	CookieSecret string `env:"MASTER_OIDC_COOKIE_SECRET"`                   // 凭证：状态 cookie 签名（≥ 32 字节）
	CookieSecure bool   `env:"MASTER_OIDC_COOKIE_SECURE" envDefault:"true"` // 仅本机 http 调试时为 false
	DisplayName  string `env:"MASTER_OIDC_DISPLAY_NAME" envDefault:"SSO"`   // 登录按钮文字
}

// Enabled 是否启用
func (c OIDCConfig) Enabled() bool { return c.Issuer != "" }

// AppConfig Master 顶层配置结构体
type AppConfig struct {
	Log        LogConfig
	Server     ServerConfig
	AgentSDK   AgentSDKConfig
	DataHub    DataHubConfig
	Database   DatabaseConfig
	Event      EventConfig
	EventAlert EventAlertConfig
	Timeout    TimeoutConfig
	JWT        JWTConfig
	Admin      AdminConfig
	AI         AIConfig
	GitHub     GitHubConfig
	OIDC       OIDCConfig
}

// GlobalConfig 全局配置实例
var GlobalConfig AppConfig
