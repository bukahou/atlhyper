// atlhyper_metrics_v2/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- 采集配置 --------------------
	"METRICS_CPU_INTERVAL":     "1s", // CPU 采样间隔
	"METRICS_PROC_INTERVAL":    "3s", // 进程采样间隔
	"METRICS_COLLECT_INTERVAL": "5s", // 总采集间隔

	// -------------------- 推送配置 --------------------
	"METRICS_PUSH_TIMEOUT":     "5s", // 推送超时
	"METRICS_PUSH_RETRY_DELAY": "1s", // 重试间隔
}

// ============================================================
// 整数类型默认值
// ============================================================
var defaultInts = map[string]int{
	// -------------------- 采集配置 --------------------
	"METRICS_TOP_PROCESSES": 10, // Top N 进程数量

	// -------------------- 推送配置 --------------------
	"METRICS_PUSH_RETRY_COUNT": 3, // 重试次数
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- 日志配置 --------------------
	"METRICS_LOG_LEVEL":  "info", // 日志级别: debug / info / warn / error
	"METRICS_LOG_FORMAT": "text", // 日志格式: text / json

	// -------------------- 路径配置 --------------------
	"METRICS_PROC_ROOT": "/proc", // /proc 路径，容器中使用 /host_proc
	"METRICS_SYS_ROOT":  "/sys",  // /sys 路径，容器中使用 /host_sys
	"METRICS_HOST_ROOT": "/",     // 宿主机根路径，容器中使用 /host_root

	// -------------------- 推送配置 --------------------
	"METRICS_AGENT_ADDR": "http://localhost:8082", // Agent 地址
}
