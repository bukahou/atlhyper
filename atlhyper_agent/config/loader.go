// atlhyper_agent/config/loader.go
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
	// -------------------- 诊断系统 --------------------
	GlobalConfig.Diagnosis = DiagnosisConfig{
		CleanInterval:            getDuration("AGENT_DIAGNOSIS_CLEAN_INTERVAL"),
		RetentionRawDuration:     getDuration("AGENT_DIAGNOSIS_RETENTION_RAW_DURATION"),
		RetentionCleanedDuration: getDuration("AGENT_DIAGNOSIS_RETENTION_CLEANED_DURATION"),
	}

	// -------------------- Kubernetes --------------------
	GlobalConfig.Kubernetes = KubernetesConfig{
		Kubeconfig:             getString("KUBECONFIG"),
		APIHealthCheckInterval: getDuration("AGENT_KUBERNETES_API_HEALTH_CHECK_INTERVAL"),
	}

	// -------------------- 集群标识 --------------------
	GlobalConfig.Cluster = ClusterConfig{
		ClusterID: getString("CLUSTER_ID"),
	}

	// -------------------- 数据推送 --------------------
	GlobalConfig.Push = PushConfig{
		MasterURL:    getString("AGENT_PUSH_MASTER_URL"),
		PushInterval: getDuration("AGENT_PUSH_INTERVAL"),
		Timeout:      getDuration("AGENT_PUSH_TIMEOUT"),
	}

	// -------------------- REST 客户端 --------------------
	GlobalConfig.RestClient = RestClientConfig{
		BaseURL:      getString("AGENT_API_BASE_URL"),
		Timeout:      getDuration("AGENT_REST_CLIENT_TIMEOUT"),
		MaxRespBytes: getInt64("AGENT_REST_CLIENT_MAX_RESP_BYTES"),
		Gzip:         getBool("AGENT_REST_CLIENT_GZIP"),
	}

	// -------------------- 内存存储 --------------------
	GlobalConfig.Store = StoreConfig{
		TTLMaxAge:       getDuration("AGENT_STORE_TTL_MAX_AGE"),
		CleanupInterval: getDuration("AGENT_STORE_CLEANUP_INTERVAL"),
	}

	// -------------------- 服务器配置 --------------------
	GlobalConfig.Server = ServerConfig{
		Port: getString("AGENT_SERVER_PORT"),
	}

	log.Printf("[config] Agent 配置加载完成: %+v", GlobalConfig)
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
	return defaultStrings[envKey]
}

func getInt64(envKey string) int64 {
	if val := os.Getenv(envKey); val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
		log.Printf("[config] 环境变量 %s 格式错误，使用默认值", envKey)
	}
	return defaultInts[envKey]
}

func getBool(envKey string) bool {
	if val := os.Getenv(envKey); val != "" {
		val = strings.ToLower(val)
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultBools[envKey]
}
