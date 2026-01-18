package model_v2

import "time"

// ============================================================
// Command 指令模型
// ============================================================

// Command 操作指令
//
// Master 下发给 Agent 执行的操作指令。
// 指令通过长轮询机制下发，Agent 执行后上报结果。
type Command struct {
	// 标识
	ID        string `json:"id"`         // 指令唯一 ID（UUID）
	ClusterID string `json:"cluster_id"` // 目标集群 ID

	// 指令类型
	Type string `json:"type"` // 指令类型（见 CommandType 常量）

	// 目标资源
	Target CommandTarget `json:"target"` // 操作目标

	// 参数（根据 Type 不同而不同）
	Params map[string]interface{} `json:"params,omitempty"`

	// 时间
	CreatedAt time.Time `json:"created_at"` // 创建时间
	ExpiresAt time.Time `json:"expires_at"` // 过期时间
}

// CommandTarget 指令目标
type CommandTarget struct {
	Kind      string `json:"kind"`                // 资源类型 (Pod, Deployment, Node 等)
	Namespace string `json:"namespace,omitempty"` // 命名空间
	Name      string `json:"name"`                // 资源名称
}

// 指令类型常量
const (
	// Pod 操作
	CommandTypePodLogs    = "pod_logs"    // 获取 Pod 日志
	CommandTypePodRestart = "pod_restart" // 重启 Pod
	CommandTypePodExec    = "pod_exec"    // 执行命令

	// Deployment 操作
	CommandTypeDeploymentScale   = "deployment_scale"   // 扩缩容
	CommandTypeDeploymentRestart = "deployment_restart" // 重启（滚动更新）
	CommandTypeDeploymentImage   = "deployment_image"   // 更新镜像

	// Node 操作
	CommandTypeNodeCordon   = "node_cordon"   // 禁止调度
	CommandTypeNodeUncordon = "node_uncordon" // 允许调度
	CommandTypeNodeDrain    = "node_drain"    // 驱逐 Pod
)

// ============================================================
// CommandResult 指令执行结果
// ============================================================

// CommandResult 指令执行结果
//
// Agent 执行指令后上报的结果。
type CommandResult struct {
	// 标识
	CommandID string `json:"command_id"` // 关联的指令 ID
	ClusterID string `json:"cluster_id"` // 集群 ID

	// 结果
	Success bool   `json:"success"`         // 是否成功
	Message string `json:"message"`         // 结果消息
	Error   string `json:"error,omitempty"` // 错误信息（失败时）

	// 数据（根据指令类型不同而不同）
	Data interface{} `json:"data,omitempty"` // 返回数据（如日志内容）

	// 时间
	StartedAt  time.Time `json:"started_at"`  // 开始执行时间
	FinishedAt time.Time `json:"finished_at"` // 完成时间
}

// ============================================================
// CommandStatus 指令状态
// ============================================================

// CommandStatus 指令状态
//
// 用于查询指令执行进度和结果。
type CommandStatus struct {
	// 标识
	CommandID string `json:"command_id"`
	ClusterID string `json:"cluster_id"`

	// 状态
	Status string `json:"status"` // pending, running, success, failed, expired

	// 指令信息
	Type   string        `json:"type"`
	Target CommandTarget `json:"target"`

	// 结果（执行完成后填充）
	Result *CommandResult `json:"result,omitempty"`

	// 时间
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// 指令状态常量
const (
	CommandStatusPending = "pending" // 等待执行
	CommandStatusRunning = "running" // 执行中
	CommandStatusSuccess = "success" // 执行成功
	CommandStatusFailed  = "failed"  // 执行失败
	CommandStatusExpired = "expired" // 已过期
)

// IsPending 判断是否等待执行
func (s *CommandStatus) IsPending() bool {
	return s.Status == CommandStatusPending
}

// IsRunning 判断是否执行中
func (s *CommandStatus) IsRunning() bool {
	return s.Status == CommandStatusRunning
}

// IsCompleted 判断是否已完成（成功或失败）
func (s *CommandStatus) IsCompleted() bool {
	return s.Status == CommandStatusSuccess || s.Status == CommandStatusFailed
}

// IsSuccess 判断是否执行成功
func (s *CommandStatus) IsSuccess() bool {
	return s.Status == CommandStatusSuccess
}

// IsFailed 判断是否执行失败
func (s *CommandStatus) IsFailed() bool {
	return s.Status == CommandStatusFailed
}
