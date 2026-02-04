// atlhyper_metrics_v2/config/loader.go
// 配置加载逻辑
package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// GlobalConfig 全局配置实例
var GlobalConfig Config

// Load 从环境变量加载配置
//
// 从环境变量加载配置，未设置则使用默认值。
// 加载完成后配置存储在 GlobalConfig 全局变量中。
func Load() {
	GlobalConfig.Log = LogConfig{
		Level:  getString("METRICS_LOG_LEVEL"),
		Format: getString("METRICS_LOG_FORMAT"),
	}

	GlobalConfig.Paths = PathsConfig{
		ProcRoot: getString("METRICS_PROC_ROOT"),
		SysRoot:  getString("METRICS_SYS_ROOT"),
		HostRoot: getString("METRICS_HOST_ROOT"),
	}

	GlobalConfig.Collect = CollectConfig{
		TopProcesses:    getInt("METRICS_TOP_PROCESSES"),
		CPUInterval:     getDuration("METRICS_CPU_INTERVAL"),
		ProcInterval:    getDuration("METRICS_PROC_INTERVAL"),
		CollectInterval: getDuration("METRICS_COLLECT_INTERVAL"),
	}

	GlobalConfig.Push = PushConfig{
		AgentAddr:  getString("METRICS_AGENT_ADDR"),
		Timeout:    getDuration("METRICS_PUSH_TIMEOUT"),
		RetryCount: getInt("METRICS_PUSH_RETRY_COUNT"),
		RetryDelay: getDuration("METRICS_PUSH_RETRY_DELAY"),
	}

	// 节点名称和主机名
	GlobalConfig.NodeName = os.Getenv("NODE_NAME")
	GlobalConfig.Hostname = getHostname()

	// 如果没有设置 NodeName，使用 Hostname
	if GlobalConfig.NodeName == "" {
		GlobalConfig.NodeName = GlobalConfig.Hostname
	}

	log.Printf("[config] Metrics 配置加载完成: NodeName=%s, AgentAddr=%s, Interval=%v",
		GlobalConfig.NodeName, GlobalConfig.Push.AgentAddr, GlobalConfig.Collect.CollectInterval)
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

// getHostname 获取主机名
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
