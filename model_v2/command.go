// model_v2/command.go
// 统一指令模型定义
// Master/Agent/HTTP 传输层共用，避免字段丢失
package model_v2

import "time"

// ============================================================
// Command 指令模型（统一定义）
// ============================================================

// Command 操作指令
//
// Master 下发给 Agent 执行的操作指令。
// 此结构体被 Master、Agent、HTTP 传输层共用。
// 新增字段时只需修改此处，全链路自动生效。
type Command struct {
	// 标识
	ID        string `json:"id"`         // 指令唯一 ID（UUID）
	ClusterID string `json:"cluster_id"` // 目标集群 ID

	// 指令内容
	Action    string         `json:"action"`              // 操作类型 (scale, restart, dynamic 等)
	Kind      string         `json:"kind,omitempty"`      // 资源类型 (Pod, Deployment, Node 等)
	Namespace string         `json:"namespace,omitempty"` // 目标命名空间
	Name      string         `json:"name,omitempty"`      // 目标资源名称
	Params    map[string]any `json:"params,omitempty"`    // 额外参数（随 Action 不同而变化）

	// 来源
	Source    string `json:"source,omitempty"`     // 来源: "ai" / "web"
	CreatedBy string `json:"created_by,omitempty"` // 创建者用户名

	// 时间
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// ============================================================
// CommandResult 指令执行结果
// ============================================================

// CommandResult 指令执行结果
//
// Agent 执行指令后上报给 Master。
type CommandResult struct {
	CommandID  string        `json:"command_id"`          // 关联的指令 ID
	Success    bool          `json:"success"`             // 是否成功
	Output     string        `json:"output,omitempty"`    // 返回数据（如日志、查询结果）
	Error      string        `json:"error,omitempty"`     // 错误信息（失败时）
	ExecTime   time.Duration `json:"exec_time,omitempty"` // 执行耗时
	ExecutedAt time.Time     `json:"executed_at"`         // 执行时间
}

// ============================================================
// CommandStatus 指令状态
// ============================================================

// CommandStatus 指令状态
//
// 用于查询指令执行进度和结果。
type CommandStatus struct {
	CommandID  string         `json:"command_id"`
	Status     string         `json:"status"` // pending, running, success, failed, timeout
	Result     *CommandResult `json:"result,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
}

// ============================================================
// 常量定义
// ============================================================

// 指令状态
const (
	CommandStatusPending = "pending"
	CommandStatusRunning = "running"
	CommandStatusSuccess = "success"
	CommandStatusFailed  = "failed"
	CommandStatusTimeout = "timeout"
)

// 指令动作
const (
	ActionScale        = "scale"
	ActionRestart      = "restart"
	ActionDelete       = "delete"
	ActionDeletePod    = "delete_pod"
	ActionExec         = "exec"
	ActionCordon       = "cordon"
	ActionUncordon     = "uncordon"
	ActionDrain        = "drain"
	ActionUpdateImage  = "update_image"
	ActionGetLogs      = "get_logs"
	ActionGetConfigMap = "get_configmap"
	ActionGetSecret    = "get_secret"
	ActionDynamic      = "dynamic" // AI 只读查询
)

// ValidActions 有效的指令动作
var ValidActions = map[string]bool{
	ActionScale:        true,
	ActionRestart:      true,
	ActionDelete:       true,
	ActionDeletePod:    true,
	ActionExec:         true,
	ActionCordon:       true,
	ActionUncordon:     true,
	ActionDrain:        true,
	ActionUpdateImage:  true,
	ActionGetLogs:      true,
	ActionGetConfigMap: true,
	ActionGetSecret:    true,
	ActionDynamic:      true,
}
