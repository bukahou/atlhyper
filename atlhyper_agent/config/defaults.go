// atlhyper_agent/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- 诊断系统 --------------------
	"AGENT_DIAGNOSIS_CLEAN_INTERVAL":             "5s",  // 诊断数据清理器执行间隔
	"AGENT_DIAGNOSIS_RETENTION_RAW_DURATION":     "10m", // 原始诊断事件保留时长
	"AGENT_DIAGNOSIS_RETENTION_CLEANED_DURATION": "5m",  // 已清理事件保留时长

	// -------------------- Kubernetes --------------------
	"AGENT_KUBERNETES_API_HEALTH_CHECK_INTERVAL": "15s", // K8s API 健康检查间隔

	// -------------------- 数据推送 --------------------
	"AGENT_PUSH_INTERVAL": "10s", // 向 Master 推送数据的间隔
	"AGENT_PUSH_TIMEOUT":  "5s",  // 推送请求超时时间

	// -------------------- REST 客户端 --------------------
	"AGENT_REST_CLIENT_TIMEOUT": "8s", // REST 客户端单次请求超时

	// -------------------- 内存存储 --------------------
	"AGENT_STORE_TTL_MAX_AGE":        "10m", // 存储数据最大存活时间
	"AGENT_STORE_CLEANUP_INTERVAL":   "1m",  // 存储清理器执行间隔
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- Kubernetes --------------------
	"KUBECONFIG": "", // kubeconfig 文件路径（空则使用 InCluster 模式）

	// -------------------- 集群标识 --------------------
	"CLUSTER_ID": "", // 集群唯一标识（空则自动获取 kube-system UID）

	// -------------------- 数据推送 --------------------
	"AGENT_PUSH_MASTER_URL": "http://localhost:8080", // Master 服务地址

	// -------------------- REST 客户端 --------------------
	"AGENT_API_BASE_URL": "http://localhost:8080", // REST API 基础地址

	// -------------------- 服务器配置 --------------------
	"AGENT_SERVER_PORT": "8082", // Agent HTTP 服务端口
}

// ============================================================
// 整数类型默认值
// ============================================================
var defaultInts = map[string]int64{
	"AGENT_REST_CLIENT_MAX_RESP_BYTES": 1 << 20, // REST 响应体读取上限（1MB）
}

// ============================================================
// 布尔类型默认值
// ============================================================
var defaultBools = map[string]bool{
	"AGENT_REST_CLIENT_GZIP": true, // REST 客户端是否启用 gzip 压缩
}
