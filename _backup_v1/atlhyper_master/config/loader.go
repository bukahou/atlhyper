// atlhyper_master/config/loader.go
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
func LoadConfig() {
	GlobalConfig.Diagnosis = DiagnosisConfig{
		CleanInterval:            getDuration("MASTER_DIAGNOSIS_CLEAN_INTERVAL"),
		WriteInterval:            getDuration("MASTER_DIAGNOSIS_WRITE_INTERVAL"),
		RetentionRawDuration:     getDuration("MASTER_DIAGNOSIS_RETENTION_RAW_DURATION"),
		RetentionCleanedDuration: getDuration("MASTER_DIAGNOSIS_RETENTION_CLEANED_DURATION"),
		UnreadyThresholdDuration: getDuration("MASTER_DIAGNOSIS_UNREADY_THRESHOLD_DURATION"),
		AlertDispatchInterval:    getDuration("MASTER_DIAGNOSIS_ALERT_DISPATCH_INTERVAL"),
		UnreadyReplicaPercent:    getFloat("MASTER_DIAGNOSIS_UNREADY_REPLICA_PERCENT"),
	}

	GlobalConfig.Kubernetes = KubernetesConfig{
		APIHealthCheckInterval: getDuration("MASTER_KUBERNETES_API_HEALTH_CHECK_INTERVAL"),
	}

	GlobalConfig.Mailer = MailerConfig{
		SMTPHost:         getString("MASTER_MAIL_SMTP_HOST"),
		SMTPPort:         getString("MASTER_MAIL_SMTP_PORT"),
		Username:         getString("MASTER_MAIL_USERNAME"),
		Password:         getString("MASTER_MAIL_PASSWORD"),
		From:             getString("MASTER_MAIL_FROM"),
		To:               getStringList("MASTER_MAIL_TO"),
		EnableEmailAlert: getBool("MASTER_ENABLE_EMAIL_ALERT"),
	}

	GlobalConfig.Slack = SlackConfig{
		WebhookURL:       getString("MASTER_SLACK_WEBHOOK_URL"),
		DispatchInterval: getDuration("MASTER_SLACK_ALERT_DISPATCH_INTERVAL"),
		EnableSlackAlert: getBool("MASTER_ENABLE_SLACK_ALERT"),
	}

	GlobalConfig.Webhook = WebhookConfig{
		Enable: getBool("MASTER_ENABLE_WEBHOOK_SERVER"),
	}

	GlobalConfig.Server = ServerConfig{
		Port: getString("MASTER_SERVER_PORT"),
	}

	GlobalConfig.CORS = CORSConfig{
		AllowOrigins:     getString("MASTER_CORS_ALLOW_ORIGINS"),
		AllowMethods:     getString("MASTER_CORS_ALLOW_METHODS"),
		AllowHeaders:     getString("MASTER_CORS_ALLOW_HEADERS"),
		AllowCredentials: getBool("MASTER_CORS_ALLOW_CREDENTIALS"),
	}

	GlobalConfig.Admin = AdminConfig{
		Username:    getString("MASTER_ADMIN_USERNAME"),
		Password:    getString("MASTER_ADMIN_PASSWORD"),
		DisplayName: getString("MASTER_ADMIN_DISPLAY_NAME"),
		Email:       getString("MASTER_ADMIN_EMAIL"),
		Role:        getString("MASTER_ADMIN_ROLE"),
	}

	GlobalConfig.Store = StoreConfig{
		TTL:             getDuration("MASTER_STORE_TTL"),
		MaxItems:        getInt("MASTER_STORE_MAX_ITEMS"),
		CleanupInterval: getDuration("MASTER_STORE_CLEANUP_INTERVAL"),
		MetricsTTL:      getDuration("MASTER_STORE_METRICS_TTL"),
	}

	GlobalConfig.JWT = JWTConfig{
		SecretKey:      getString("MASTER_JWT_SECRET_KEY"),
		TokenExpiry:    getDuration("MASTER_JWT_TOKEN_EXPIRY"),
		MinPasswordLen: getInt("MASTER_JWT_MIN_PASSWORD_LEN"),
	}

	log.Printf("[config] Master 配置加载完成: %+v", GlobalConfig)
}

// ==================== 工具函数 ====================

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

func getStringList(envKey string) []string {
	if val := os.Getenv(envKey); val != "" {
		list := strings.Split(val, ",")
		for i := range list {
			list[i] = strings.TrimSpace(list[i])
		}
		return list
	}
	def := defaultStrings[envKey]
	if def == "" {
		return []string{}
	}
	return strings.Split(def, ",")
}

func getBool(envKey string) bool {
	val := os.Getenv(envKey)
	if val != "" {
		val = strings.ToLower(val)
		return val == "true" || val == "1" || val == "yes"
	}
	def, ok := defaultBools[envKey]
	if !ok {
		log.Fatalf("[config] 未定义默认布尔配置项: %s", envKey)
	}
	return def
}

func getInt(envKey string) int {
	if val := os.Getenv(envKey); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
		log.Printf("[config] 环境变量 %s 格式错误，使用默认值", envKey)
	}
	def, ok := defaultInts[envKey]
	if !ok {
		log.Fatalf("[config] 未定义默认整数配置项: %s", envKey)
	}
	return def
}

func getFloat(envKey string) float64 {
	if val := os.Getenv(envKey); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
		log.Printf("[config] 环境变量 %s 格式错误，使用默认值", envKey)
	}
	def, ok := defaultFloats[envKey]
	if !ok {
		log.Fatalf("[config] 未定义默认浮点数配置项: %s", envKey)
	}
	return def
}
