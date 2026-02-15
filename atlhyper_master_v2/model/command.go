// atlhyper_master_v2/model/command.go
// 指令模型（camelCase JSON tag，用于 Web API 响应和内部 MQ 序列化）
package model

import "time"

// Command 指令
type Command struct {
	ID              string                 `json:"id"`
	ClusterID       string                 `json:"clusterId"`
	Action          string                 `json:"action"` // scale, restart, delete_pod, exec, cordon, uncordon, etc.
	TargetKind      string                 `json:"targetKind"`
	TargetNamespace string                 `json:"targetNamespace"`
	TargetName      string                 `json:"targetName"`
	Params          map[string]interface{} `json:"params,omitempty"`
	Source          string                 `json:"source,omitempty"` // 来源: "ai" / "web"
	CreatedAt       time.Time              `json:"createdAt"`
	CreatedBy       string                 `json:"createdBy,omitempty"` // 创建者用户名
}

// CommandResult 指令执行结果
type CommandResult struct {
	CommandID string        `json:"commandId"`
	Success   bool          `json:"success"`
	Output    string        `json:"output,omitempty"`
	Error     string        `json:"error,omitempty"`
	ExecTime  time.Duration `json:"execTime,omitempty"`
}

// CommandStatus 指令状态
type CommandStatus struct {
	CommandID  string         `json:"commandId"`
	Status     string         `json:"status"` // pending, running, success, failed, timeout
	Result     *CommandResult `json:"result,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
	StartedAt  *time.Time     `json:"startedAt,omitempty"`
	FinishedAt *time.Time     `json:"finishedAt,omitempty"`
}

// 指令状态常量
const (
	CommandStatusPending = "pending"
	CommandStatusRunning = "running"
	CommandStatusSuccess = "success"
	CommandStatusFailed  = "failed"
	CommandStatusTimeout = "timeout"
)

// 指令动作常量
const (
	ActionScale        = "scale"
	ActionRestart      = "restart"
	ActionDelete       = "delete"
	ActionDeletePod    = "delete_pod" // alias for delete on Pod
	ActionExec         = "exec"
	ActionCordon       = "cordon"
	ActionUncordon     = "uncordon"
	ActionDrain        = "drain"
	ActionUpdateImage  = "update_image"
	ActionGetLogs      = "get_logs"
	ActionGetConfigMap = "get_configmap" // 获取 ConfigMap 数据
	ActionGetSecret    = "get_secret"    // 获取 Secret 数据
	ActionDynamic      = "dynamic"       // AI 只读查询
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
