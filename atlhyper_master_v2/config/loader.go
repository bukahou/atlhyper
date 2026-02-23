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
	GlobalConfig.Log = LogConfig{
		Level:  getString("MASTER_LOG_LEVEL"),
		Format: getString("MASTER_LOG_FORMAT"),
	}

	GlobalConfig.Server = ServerConfig{
		GatewayPort:  getInt("MASTER_GATEWAY_PORT"),
		AgentSDKPort: getInt("MASTER_AGENTSDK_PORT"),
		TesterPort:   getInt("MASTER_TESTER_PORT"),
	}

	GlobalConfig.DataHub = DataHubConfig{
		Type:              getString("MASTER_DATAHUB_TYPE"),
		EventRetention:    getDuration("MASTER_DATAHUB_EVENT_RETENTION"),
		SnapshotRetain:    getInt("MASTER_DATAHUB_SNAPSHOT_RETAIN"),
		HeartbeatExpire:   getDuration("MASTER_DATAHUB_HEARTBEAT_EXPIRE"),
		SnapshotRetention: getDuration("MASTER_DATAHUB_SNAPSHOT_RETENTION"),
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

	GlobalConfig.EventAlert = EventAlertConfig{
		Enabled:       getBool("MASTER_EVENT_ALERT_ENABLED"),
		CheckInterval: getDuration("MASTER_EVENT_ALERT_INTERVAL"),
	}

	GlobalConfig.Timeout = TimeoutConfig{
		CommandPoll: getDuration("MASTER_TIMEOUT_COMMAND_POLL"),
		Heartbeat:   getDuration("MASTER_TIMEOUT_HEARTBEAT"),
	}

	GlobalConfig.JWT = JWTConfig{
		SecretKey:   getString("MASTER_JWT_SECRET"),
		TokenExpiry: getDuration("MASTER_JWT_TOKEN_EXPIRY"),
	}

	GlobalConfig.Admin = AdminConfig{
		Username:    getString("MASTER_ADMIN_USERNAME"),
		Password:    getString("MASTER_ADMIN_PASSWORD"),
		DisplayName: getString("MASTER_ADMIN_DISPLAY_NAME"),
	}

	GlobalConfig.AI = AIConfig{
		Enabled:     getBool("MASTER_AI_ENABLED"),
		ToolTimeout: getDuration("MASTER_AI_TOOL_TIMEOUT"),
	}

	GlobalConfig.SLO = SLOConfig{
		AggregateInterval: getDuration("MASTER_SLO_AGGREGATE_INTERVAL"),
		CleanupInterval:   getDuration("MASTER_SLO_CLEANUP_INTERVAL"),
		RawRetention:      getDuration("MASTER_SLO_RAW_RETENTION"),
		HourlyRetention:   getDuration("MASTER_SLO_HOURLY_RETENTION"),
		StatusRetention:   getDuration("MASTER_SLO_STATUS_RETENTION"),
	}

	GlobalConfig.MetricsPersist = MetricsPersistConfig{
		SampleInterval:  getDuration("MASTER_METRICS_SAMPLE_INTERVAL"),
		RetentionDays:   getInt("MASTER_METRICS_RETENTION_DAYS"),
		CleanupInterval: getDuration("MASTER_METRICS_CLEANUP_INTERVAL"),
	}

	log.Printf("[config] Master 配置加载完成: GatewayPort=%d, AgentSDKPort=%d, TesterPort=%d, DBType=%s, Admin=%s",
		GlobalConfig.Server.GatewayPort, GlobalConfig.Server.AgentSDKPort, GlobalConfig.Server.TesterPort, GlobalConfig.Database.Type, GlobalConfig.Admin.Username)
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

// getStringSlice 获取字符串数组类型配置（逗号分隔）
func getStringSlice(envKey string) []string {
	val := os.Getenv(envKey)
	if val == "" {
		if def, ok := defaultStrings[envKey]; ok {
			val = def
		}
	}
	if val == "" {
		return []string{}
	}
	// 按逗号分隔并去除空格
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
