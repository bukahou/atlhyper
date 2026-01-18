// atlhyper_agent_v2/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- 调度器配置 --------------------
	"AGENT_SNAPSHOT_INTERVAL":     "20s", // 快照采集间隔
	"AGENT_COMMAND_POLL_INTERVAL": "1s",  // 指令轮询间隔
	"AGENT_HEARTBEAT_INTERVAL":    "15s", // 心跳发送间隔

	// -------------------- 超时配置 --------------------
	"AGENT_TIMEOUT_HTTP_CLIENT":       "90s", // HTTP 客户端超时 (需 > Master 长轮询超时 60s + 网络开销)
	"AGENT_TIMEOUT_SNAPSHOT_COLLECT":  "30s", // 快照采集操作超时
	"AGENT_TIMEOUT_COMMAND_POLL":      "60s", // 指令轮询操作超时 (长轮询)
	"AGENT_TIMEOUT_HEARTBEAT":         "10s", // 心跳操作超时
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- Agent 基础配置 --------------------
	"AGENT_CLUSTER_ID": "", // 集群唯一标识，空则自动获取集群 UID

	// -------------------- Master 通信 --------------------
	"AGENT_MASTER_URL": "http://localhost:8081", // Master AgentSDK 端口（非 Gateway 端口）

	// -------------------- Kubernetes 配置 --------------------
	"AGENT_KUBECONFIG": "", // kubeconfig 文件路径，空则使用 InCluster 模式
}
