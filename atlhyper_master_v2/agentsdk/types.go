// atlhyper_master_v2/agentsdk/types.go
// Agent 通信协议类型定义
// 使用 model_v3/command（camelCase JSON tag），与 Agent V2 保持一致
package agentsdk

import (
	"time"

	"AtlHyper/model_v3/command"
)

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	ClusterID string    `json:"clusterId"`
	Timestamp time.Time `json:"timestamp"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Status string `json:"status"`
}

// CommandResponse 指令响应
type CommandResponse struct {
	HasCommand bool             `json:"hasCommand"`
	Command    *command.Command `json:"command,omitempty"`
}

// ResultRequest 执行结果请求
type ResultRequest struct {
	ClusterID  string `json:"clusterId"`
	CommandID  string `json:"commandId"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	ExecTimeMs int64  `json:"execTime"`
}

// ResultResponse 执行结果响应
type ResultResponse struct {
	Status string `json:"status"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}
