// atlhyper_master/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- 诊断系统 --------------------
	"MASTER_DIAGNOSIS_CLEAN_INTERVAL":             "5s",  // 诊断数据清理器执行间隔
	"MASTER_DIAGNOSIS_WRITE_INTERVAL":             "6s",  // 诊断日志写入间隔
	"MASTER_DIAGNOSIS_RETENTION_RAW_DURATION":     "10m", // 原始诊断事件保留时长
	"MASTER_DIAGNOSIS_RETENTION_CLEANED_DURATION": "5m",  // 已清理事件保留时长
	"MASTER_DIAGNOSIS_UNREADY_THRESHOLD_DURATION": "7s",  // Pod 不可用持续多久后触发告警
	"MASTER_DIAGNOSIS_ALERT_DISPATCH_INTERVAL":    "5s",  // 邮件告警轮询检测间隔

	// -------------------- Kubernetes --------------------
	"MASTER_KUBERNETES_API_HEALTH_CHECK_INTERVAL": "15s", // K8s API 健康检查间隔

	// -------------------- Slack --------------------
	"MASTER_SLACK_ALERT_DISPATCH_INTERVAL": "5s", // Slack 告警推送间隔

	// -------------------- 内存存储（master_store） --------------------
	"MASTER_STORE_TTL":              "24h", // 每条记录的默认生存时间
	"MASTER_STORE_CLEANUP_INTERVAL": "5m",  // 清理任务的执行间隔
	"MASTER_STORE_METRICS_TTL":      "15m", // 指标数据的 TTL（比事件更短，减少内存占用）

	// -------------------- JWT 认证 --------------------
	"MASTER_JWT_TOKEN_EXPIRY": "24h", // JWT Token 有效期
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- 邮件配置 --------------------
	"MASTER_MAIL_SMTP_HOST": "smtp.gmail.com",      // SMTP 服务器地址
	"MASTER_MAIL_SMTP_PORT": "587",                 // SMTP 端口
	"MASTER_MAIL_USERNAME":  "",                    // SMTP 用户名（敏感信息）
	"MASTER_MAIL_PASSWORD":  "",                    // SMTP 密码（敏感信息）
	"MASTER_MAIL_FROM":      "noreply@example.com", // 发件人地址
	"MASTER_MAIL_TO":        "",                    // 收件人列表，逗号分隔

	// -------------------- Slack 配置 --------------------
	"MASTER_SLACK_WEBHOOK_URL": "", // Slack Webhook URL（敏感信息）

	// -------------------- 管理员配置 --------------------
	"MASTER_ADMIN_USERNAME":     "admin",             // 初始管理员用户名
	"MASTER_ADMIN_PASSWORD":     "123456",            // 初始管理员密码（敏感信息，生产环境请修改）
	"MASTER_ADMIN_DISPLAY_NAME": "Atlhyper",          // 初始管理员显示名称
	"MASTER_ADMIN_EMAIL":        "admin@example.com", // 初始管理员邮箱
	"MASTER_ADMIN_ROLE":         "3",                 // 初始管理员角色（3=超级管理员）

	// -------------------- 服务器配置 --------------------
	"MASTER_SERVER_PORT": "8080", // HTTP 服务端口

	// -------------------- JWT 认证 --------------------
	"MASTER_JWT_SECRET_KEY": "atlhyper_jwt_secret_key_change_in_production", // JWT 签名密钥（生产环境请修改）
}

// ============================================================
// 布尔类型默认值
// ============================================================
var defaultBools = map[string]bool{
	"MASTER_ENABLE_EMAIL_ALERT":    false, // 是否启用邮件告警
	"MASTER_ENABLE_SLACK_ALERT":    false, // 是否启用 Slack 告警
	"MASTER_ENABLE_WEBHOOK_SERVER": false, // 是否启用 Webhook 服务
}

// ============================================================
// 整数类型默认值
// ============================================================
var defaultInts = map[string]int{
	// -------------------- 内存存储（master_store） --------------------
	"MASTER_STORE_MAX_ITEMS": 50000, // 全局池最多保留的记录数（超过则裁剪最旧的）

	// -------------------- JWT 认证 --------------------
	"MASTER_JWT_MIN_PASSWORD_LEN": 6, // 密码最小长度
}

// ============================================================
// 浮点类型默认值
// ============================================================
var defaultFloats = map[string]float64{
	"MASTER_DIAGNOSIS_UNREADY_REPLICA_PERCENT": 0.6, // Pod 不可用比例阈值（超过则告警）
}
