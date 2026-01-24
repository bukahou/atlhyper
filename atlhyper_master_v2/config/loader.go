// atlhyper_master_v2/config/loader.go
// 配置加载逻辑
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadConfig 加载所有配置项
//
// 从环境变量加载配置，未设置则使用默认值。
// 加载完成后配置存储在 GlobalConfig 全局变量中。
func LoadConfig() {
	GlobalConfig.Server = ServerConfig{
		GatewayPort:  getInt("MASTER_GATEWAY_PORT"),
		AgentSDKPort: getInt("MASTER_AGENTSDK_PORT"),
	}

	GlobalConfig.DataHub = DataHubConfig{
		Type:            getString("MASTER_DATAHUB_TYPE"),
		EventRetention:  getDuration("MASTER_DATAHUB_EVENT_RETENTION"),
		SnapshotRetain:  getInt("MASTER_DATAHUB_SNAPSHOT_RETAIN"),
		HeartbeatExpire: getDuration("MASTER_DATAHUB_HEARTBEAT_EXPIRE"),
	}

	GlobalConfig.Redis = RedisConfig{
		Addr:     getString("MASTER_REDIS_ADDR"),
		Password: getString("MASTER_REDIS_PASSWORD"),
		DB:       getInt("MASTER_REDIS_DB"),
	}

	GlobalConfig.Database = DatabaseConfig{
		Type:     getString("MASTER_DB_TYPE"),
		Path:     getString("MASTER_DB_PATH"),
		DSN:      getString("MASTER_DB_DSN"),
		MaxConns: getInt("MASTER_DB_MAX_CONNS"),
	}

	GlobalConfig.Event = EventConfig{
		RetentionDays:   getInt("MASTER_EVENT_RETENTION_DAYS"),
		MaxCount:        getInt("MASTER_EVENT_MAX_COUNT"),
		CleanupInterval: getDuration("MASTER_EVENT_CLEANUP_INTERVAL"),
	}

	GlobalConfig.Timeout = TimeoutConfig{
		CommandPoll: getDuration("MASTER_TIMEOUT_COMMAND_POLL"),
		Heartbeat:   getDuration("MASTER_TIMEOUT_HEARTBEAT"),
	}

	GlobalConfig.JWT = JWTConfig{
		SecretKey:   getString("MASTER_JWT_SECRET"),
		TokenExpiry: getDuration("MASTER_JWT_TOKEN_EXPIRY"),
	}

	GlobalConfig.Notifier = NotifierConfig{
		Mail: MailConfig{
			Enabled:  getBool("MASTER_MAIL_ENABLED"),
			SMTPHost: getString("MASTER_MAIL_SMTP_HOST"),
			SMTPPort: getInt("MASTER_MAIL_SMTP_PORT"),
			Username: getString("MASTER_MAIL_USERNAME"),
			Password: getString("MASTER_MAIL_PASSWORD"),
			From:     getString("MASTER_MAIL_FROM"),
			To:       getString("MASTER_MAIL_TO"),
		},
		Webhook: WebhookConfig{
			Enabled: getBool("MASTER_WEBHOOK_ENABLED"),
			URL:     getString("MASTER_WEBHOOK_URL"),
			Secret:  getString("MASTER_WEBHOOK_SECRET"),
		},
	}

	GlobalConfig.Admin = AdminConfig{
		Username:    getString("MASTER_ADMIN_USERNAME"),
		Password:    getString("MASTER_ADMIN_PASSWORD"),
		DisplayName: getString("MASTER_ADMIN_DISPLAY_NAME"),
	}

	GlobalConfig.AI = AIConfig{
		Enabled:     getBool("MASTER_AI_ENABLED"),
		Provider:    getString("MASTER_AI_PROVIDER"),
		APIKey:      getString("MASTER_AI_GEMINI_API_KEY"),
		Model:       getString("MASTER_AI_GEMINI_MODEL"),
		ToolTimeout: getDuration("MASTER_AI_TOOL_TIMEOUT"),
	}

	log.Printf("[config] Master 配置加载完成: GatewayPort=%d, AgentSDKPort=%d, DBType=%s, Admin=%s",
		GlobalConfig.Server.GatewayPort, GlobalConfig.Server.AgentSDKPort, GlobalConfig.Database.Type, GlobalConfig.Admin.Username)

	// 打印通知配置状态
	if GlobalConfig.Notifier.Mail.Enabled {
		log.Printf("[config] 邮件通知已启用: %s -> %s", GlobalConfig.Notifier.Mail.From, GlobalConfig.Notifier.Mail.To)
	}
	if GlobalConfig.Notifier.Webhook.Enabled {
		log.Printf("[config] Webhook 通知已启用: %s", GlobalConfig.Notifier.Webhook.URL)
	}
	if GlobalConfig.AI.Enabled {
		log.Printf("[config] AI 功能已启用: provider=%s, model=%s", GlobalConfig.AI.Provider, GlobalConfig.AI.Model)
	}
}

// ==================== 工具函数 ====================

// getDuration 获取时间类型配置
func getDuration(envKey string) time.Duration {
	if val := os.Getenv(envKey); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		log.Printf("[config] 环境变量 %s 格式错误，使用默认值", envKey)
	}
	def, ok := defaultDurations[envKey]
	if !ok {
		log.Fatalf("[config] 未定义默认时间配置项: %s", envKey)
	}
	d, _ := time.ParseDuration(def)
	return d
}

// getInt 获取整数类型配置
func getInt(envKey string) int {
	if val := os.Getenv(envKey); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
		log.Printf("[config] 环境变量 %s 格式错误，使用默认值", envKey)
	}
	def, ok := defaultInts[envKey]
	if !ok {
		log.Fatalf("[config] 未定义默认整数配置项: %s", envKey)
	}
	return def
}

// getString 获取字符串类型配置
func getString(envKey string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	def, ok := defaultStrings[envKey]
	if !ok {
		log.Fatalf("[config] 未定义默认字符串配置项: %s", envKey)
	}
	return def
}

// getBool 获取布尔类型配置
func getBool(envKey string) bool {
	if val := os.Getenv(envKey); val != "" {
		// 支持多种布尔表示
		lower := strings.ToLower(val)
		return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
	}
	def, ok := defaultBools[envKey]
	if !ok {
		log.Fatalf("[config] 未定义默认布尔配置项: %s", envKey)
	}
	return def
}
