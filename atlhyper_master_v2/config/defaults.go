// atlhyper_master_v2/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- DataHub 配置 --------------------
	"MASTER_DATAHUB_EVENT_RETENTION":  "30m", // Event 保留时间
	"MASTER_DATAHUB_HEARTBEAT_EXPIRE": "45s", // 心跳过期时间

	// -------------------- 超时配置 --------------------
	"MASTER_TIMEOUT_COMMAND_POLL": "60s", // 长轮询超时
	"MASTER_TIMEOUT_HEARTBEAT":    "45s", // 心跳超时阈值

	// -------------------- Event 持久化 --------------------
	"MASTER_EVENT_CLEANUP_INTERVAL": "1h", // 清理检查间隔

	// -------------------- JWT 配置 --------------------
	"MASTER_JWT_TOKEN_EXPIRY": "24h", // Token 有效期

	// -------------------- AI 配置 --------------------
	"MASTER_AI_TOOL_TIMEOUT": "30s", // Tool 执行超时
}

// ============================================================
// 整数类型默认值
// ============================================================
var defaultInts = map[string]int{
	// -------------------- 服务器配置 --------------------
	"MASTER_GATEWAY_PORT":  8080, // Gateway 端口（Web/API）
	"MASTER_AGENTSDK_PORT": 8081, // AgentSDK 端口（Agent 数据上报）
	"MASTER_TESTER_PORT":   9080, // Tester 端口（测试服务）

	// -------------------- DataHub 配置 --------------------
	"MASTER_DATAHUB_SNAPSHOT_RETAIN": 10, // 快照保留数量

	// -------------------- Redis 配置 --------------------
	"MASTER_REDIS_DB": 0, // Redis 数据库编号

	// -------------------- 数据库配置 --------------------
	"MASTER_DB_MAX_CONNS": 10, // 最大连接数

	// -------------------- Event 持久化 --------------------
	"MASTER_EVENT_RETENTION_DAYS": 30,     // 保留天数
	"MASTER_EVENT_MAX_COUNT":      100000, // 单集群最大事件数

	// -------------------- 邮件配置 --------------------
	"MASTER_MAIL_SMTP_PORT": 587, // SMTP 端口（587 for TLS）
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- DataHub 配置 --------------------
	"MASTER_DATAHUB_TYPE": "memory", // DataHub 类型

	// -------------------- Redis 配置 --------------------
	"MASTER_REDIS_ADDR":     "localhost:6379", // Redis 地址
	"MASTER_REDIS_PASSWORD": "",               // Redis 密码

	// -------------------- 数据库配置 --------------------
	"MASTER_DB_TYPE": "sqlite",         // 数据库类型
	"MASTER_DB_PATH": "atlhyper_master_v2/database/sqlite/data/master.db", // SQLite 路径
	"MASTER_DB_DSN":  "",               // MySQL/PG 连接串

	// -------------------- JWT 配置 --------------------
	"MASTER_JWT_SECRET": "atlhyper-default-secret-change-in-production", // JWT 密钥

	// -------------------- 邮件配置 --------------------
	"MASTER_MAIL_SMTP_HOST": "",                    // SMTP 服务器地址
	"MASTER_MAIL_USERNAME":  "",                    // SMTP 用户名
	"MASTER_MAIL_PASSWORD":  "",                    // SMTP 密码（敏感信息）
	"MASTER_MAIL_FROM":      "", // 发件人地址
	"MASTER_MAIL_TO":        "",                    // 收件人列表，逗号分隔

	// -------------------- Webhook 配置 --------------------
	"MASTER_WEBHOOK_URL":    "", // Webhook URL
	"MASTER_WEBHOOK_SECRET": "", // Webhook 签名密钥

	// -------------------- 默认管理员配置 --------------------
	"MASTER_ADMIN_USERNAME":     "admin",         // 管理员用户名
	"MASTER_ADMIN_PASSWORD":     "admin123",      // 管理员密码（首次登录后请修改）
	"MASTER_ADMIN_DISPLAY_NAME": "Administrator", // 管理员显示名称

	// -------------------- AI 配置 --------------------
	"MASTER_AI_PROVIDER":       "gemini",          // LLM 提供商
	"MASTER_AI_GEMINI_API_KEY": "",                // Gemini API Key（必须设置才能启用 AI）
	"MASTER_AI_GEMINI_MODEL":   "gemini-2.0-flash", // Gemini 模型
}

// ============================================================
// 布尔类型默认值
// ============================================================
var defaultBools = map[string]bool{
	// -------------------- 邮件配置 --------------------
	"MASTER_MAIL_ENABLED": false, // 是否启用邮件通知

	// -------------------- Webhook 配置 --------------------
	"MASTER_WEBHOOK_ENABLED": false, // 是否启用 Webhook

	// -------------------- AI 配置 --------------------
	"MASTER_AI_ENABLED": true, // 是否启用 AI 功能
}
