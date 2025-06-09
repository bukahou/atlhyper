package rootcause

type RootCause struct {
	Code       string
	Source     string // Internal / External
	Scope      string // Pod / Node / Endpoint / Deployment / ...
	RootCause  string
	MatchRules []string
	Priority   int // 优先级，数值越高表示越可能是主因
}

// PriorityLevel 定义 RootCause 的优先级等级（数值越高，越可能为根本原因）
const (
	// 优先级 100 - 核心基础故障（系统级、影响全局）
	PriorityCritical = 100 // 例：NodeNotReady，系统不可用根因

	// 优先级 90 - 节点资源压力（系统级、但具备恢复能力）
	PriorityHigh = 90 // 例：NodePressure

	// 优先级 80 - 服务入口不可用（例如 Endpoint 无 Ready Pod）
	PriorityServiceUnavailable = 80 // 例：NoReadyEndpoint

	// 优先级 60 - 应用不可用（容器运行异常、探针失败等）
	PriorityAppFailure = 60 // 例：ProbeFailed / CrashLoopBackOff

	// 优先级 50 - 控制器层异常（Deployment 副本不匹配）
	PriorityControlMismatch = 50 // 例：DeploymentMismatch

	// 优先级 30 - 配置或交付问题（镜像拉取失败等）
	PriorityConfigIssue = 30 // 例：ImagePullError

	// 优先级 10 - 异常不明确或信息不足
	PriorityUnknown = 10
)
