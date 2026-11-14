// Package command 定义 Master → Agent 指令模型
package command

import (
	"errors"
	"time"
)

// Command 操作指令（Master 下发给 Agent）
type Command struct {
	ID        string         `json:"id"`
	ClusterID string         `json:"clusterId"`
	Action    string         `json:"action"`
	Kind      string         `json:"kind,omitempty"`
	Namespace string         `json:"namespace,omitempty"`
	Name      string         `json:"name,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
	Source    string         `json:"source,omitempty"`
	CreatedBy string         `json:"createdBy,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// Result 指令执行结果（Agent 上报）
type Result struct {
	CommandID  string        `json:"commandId"`
	Success    bool          `json:"success"`
	Output     string        `json:"output,omitempty"`
	Error      string        `json:"error,omitempty"`
	ExecTime   time.Duration `json:"execTime,omitempty"`
	ExecutedAt time.Time     `json:"executedAt"`
}

// Status 指令状态查询
type Status struct {
	CommandID  string     `json:"commandId"`
	ClusterID  string     `json:"clusterId,omitempty"` // 指令所属的集群；回报结果时据此核对
	Status     string     `json:"status"`
	Result     *Result    `json:"result,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// ErrNotOwnedByCluster 回报结果的集群不是这条指令所属的集群
var ErrNotOwnedByCluster = errors.New("指令不属于回报结果的集群")

// 指令状态常量
const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusTimeout = "timeout"
)

// 指令动作常量
const (
	ActionScale          = "scale"
	ActionRestart        = "restart"
	ActionDelete         = "delete"
	ActionDeletePod      = "delete_pod"
	ActionExec           = "exec"
	ActionCordon         = "cordon"
	ActionUncordon       = "uncordon"
	ActionDrain          = "drain"
	ActionUpdateImage    = "update_image"
	ActionGetLogs        = "get_logs"
	ActionGetConfigMap   = "get_configmap"
	ActionGetSecret      = "get_secret"
	ActionDynamic        = "dynamic"
	ActionApplyManifests = "apply_manifests"

	// ClickHouse 查询动作
	ActionQueryTraces      = "query_traces"
	ActionQueryTraceDetail = "query_trace_detail"
	ActionQueryLogs        = "query_logs"
	ActionQueryMetrics     = "query_metrics"
	ActionQuerySLO         = "query_slo"
)

// ValidActions 有效的指令动作
var ValidActions = map[string]bool{
	ActionScale: true, ActionRestart: true, ActionDelete: true,
	ActionDeletePod: true, ActionExec: true, ActionCordon: true,
	ActionUncordon: true, ActionDrain: true, ActionUpdateImage: true,
	ActionGetLogs: true, ActionGetConfigMap: true, ActionGetSecret: true,
	ActionDynamic: true, ActionApplyManifests: true,
	ActionQueryTraces: true, ActionQueryTraceDetail: true,
	ActionQueryLogs: true, ActionQueryMetrics: true, ActionQuerySLO: true,
}
