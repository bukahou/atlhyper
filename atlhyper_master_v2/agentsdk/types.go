// atlhyper_master_v2/agentsdk/types.go
// Agent 通信协议类型定义
// 注意：快照数据使用 model_v2.ClusterSnapshot，不再在此定义
package agentsdk

import "time"

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	ClusterID string    `json:"cluster_id"`
	Timestamp time.Time `json:"timestamp"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Status string `json:"status"`
}

// CommandResponse 指令响应
type CommandResponse struct {
	HasCommand bool         `json:"has_command"`
	Command    *CommandInfo `json:"command,omitempty"`
}

// CommandInfo 指令信息
type CommandInfo struct {
	ID              string                 `json:"id"`
	Action          string                 `json:"action"`
	TargetKind      string                 `json:"target_kind"`
	TargetNamespace string                 `json:"target_namespace"`
	TargetName      string                 `json:"target_name"`
	Params          map[string]interface{} `json:"params,omitempty"`
}

// ResultRequest 执行结果请求
type ResultRequest struct {
	ClusterID  string `json:"cluster_id"`
	CommandID  string `json:"command_id"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	ExecTimeMs int64  `json:"exec_time_ms"`
}

// ResultResponse 执行结果响应
type ResultResponse struct {
	Status string `json:"status"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}
