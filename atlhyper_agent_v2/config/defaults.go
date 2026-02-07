// atlhyper_agent_v2/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- 调度器配置 --------------------
	"AGENT_SNAPSHOT_INTERVAL":     "10s", // 快照采集间隔
	"AGENT_COMMAND_POLL_INTERVAL": "1s",  // 指令轮询间隔
	"AGENT_HEARTBEAT_INTERVAL":    "15s", // 心跳发送间隔

	// -------------------- 超时配置 --------------------
	"AGENT_TIMEOUT_HTTP_CLIENT":      "90s", // HTTP 客户端超时 (需 > Master 长轮询超时 60s + 网络开销)
	"AGENT_TIMEOUT_SNAPSHOT_COLLECT": "30s", // 快照采集操作超时
	"AGENT_TIMEOUT_COMMAND_POLL":     "60s", // 指令轮询操作超时 (长轮询)
	"AGENT_TIMEOUT_HEARTBEAT":        "10s", // 心跳操作超时

	// -------------------- SLO 配置 --------------------
	"AGENT_SLO_SCRAPE_INTERVAL": "10s", // SLO 指标采集间隔
	"AGENT_SLO_SCRAPE_TIMEOUT":  "5s",  // SLO 指标采集超时
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- 日志配置 --------------------
	"AGENT_LOG_LEVEL":  "info", // 日志级别: debug / info / warn / error
	"AGENT_LOG_FORMAT": "text", // 日志格式: text / json

	// -------------------- Agent 基础配置 --------------------
	"AGENT_CLUSTER_ID": "", // 集群唯一标识，空则自动获取集群 UID

	// -------------------- Master 通信 --------------------
	"AGENT_MASTER_URL": "http://localhost:8081", // Master AgentSDK 端口（非 Gateway 端口）

	// -------------------- Kubernetes 配置 --------------------
	"AGENT_KUBECONFIG": "", // kubeconfig 文件路径，空则使用 InCluster 模式

	// -------------------- SLO 配置 --------------------
	"AGENT_SLO_OTEL_METRICS_URL":    "http://otel-collector.otel.svc:8889/metrics", // OTel Collector 指标端点
	"AGENT_SLO_OTEL_HEALTH_URL":     "http://otel-collector.otel.svc:13133",        // OTel Collector 健康检查
	"AGENT_SLO_EXCLUDE_NAMESPACES":  "linkerd,linkerd-viz,kube-system,otel",        // SLO 排除的 namespace (逗号分隔)
}

// ============================================================
// 布尔类型默认值
// ============================================================
var defaultBools = map[string]bool{
	// -------------------- SLO 配置 --------------------
	"AGENT_SLO_ENABLED": true, // 是否启用 SLO 采集

	// -------------------- Metrics SDK 配置 --------------------
	"AGENT_METRICS_SDK_ENABLED": true, // 是否启用 Metrics SDK
}

// ============================================================
// 整数类型默认值
// ============================================================
var defaultInts = map[string]int{
	// -------------------- Metrics SDK 配置 --------------------
	"AGENT_METRICS_SDK_PORT": 8082, // Metrics SDK HTTP 端口
}
