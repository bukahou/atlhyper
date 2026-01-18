// atlhyper_agent_v2/config/loader.go
// 配置加载逻辑
package config

import (
	"log"
	"os"
	"time"
)

// LoadConfig 加载所有配置项
//
// 从环境变量加载配置，未设置则使用默认值。
// 加载完成后配置存储在 GlobalConfig 全局变量中。
func LoadConfig() {
	GlobalConfig.Agent = AgentConfig{
		ClusterID: getString("AGENT_CLUSTER_ID"),
	}

	GlobalConfig.Master = MasterConfig{
		URL: getString("AGENT_MASTER_URL"),
	}

	GlobalConfig.Kubernetes = KubernetesConfig{
		KubeConfig: getKubeConfig(),
	}

	GlobalConfig.Scheduler = SchedulerConfig{
		SnapshotInterval:    getDuration("AGENT_SNAPSHOT_INTERVAL"),
		CommandPollInterval: getDuration("AGENT_COMMAND_POLL_INTERVAL"),
		HeartbeatInterval:   getDuration("AGENT_HEARTBEAT_INTERVAL"),
	}

	GlobalConfig.Timeout = TimeoutConfig{
		HTTPClient:      getDuration("AGENT_TIMEOUT_HTTP_CLIENT"),
		SnapshotCollect: getDuration("AGENT_TIMEOUT_SNAPSHOT_COLLECT"),
		CommandPoll:     getDuration("AGENT_TIMEOUT_COMMAND_POLL"),
		Heartbeat:       getDuration("AGENT_TIMEOUT_HEARTBEAT"),
	}

	log.Printf("[config] Agent 配置加载完成: ClusterID=%s, MasterURL=%s",
		GlobalConfig.Agent.ClusterID, GlobalConfig.Master.URL)
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

// getKubeConfig 获取 kubeconfig 路径
// 优先级: AGENT_KUBECONFIG > KUBECONFIG > 空（使用 in-cluster）
func getKubeConfig() string {
	// 1. 优先使用 Agent 专用配置
	if val := os.Getenv("AGENT_KUBECONFIG"); val != "" {
		log.Printf("[config] 使用 AGENT_KUBECONFIG: %s", val)
		return val
	}

	// 2. 回退到标准 KUBECONFIG 环境变量
	if val := os.Getenv("KUBECONFIG"); val != "" {
		log.Printf("[config] 使用 KUBECONFIG: %s", val)
		return val
	}

	// 3. 返回空，使用 in-cluster 模式
	log.Printf("[config] 未设置 kubeconfig，将使用 in-cluster 模式")
	return ""
}
