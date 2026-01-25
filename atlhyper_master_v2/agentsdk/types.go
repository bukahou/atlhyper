// atlhyper_master_v2/agentsdk/types.go
// Agent 通信协议类型定义
// 使用 model_v2 统一定义，避免字段丢失
package agentsdk

import (
	"time"

	"AtlHyper/model_v2"
)

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
	HasCommand bool             `json:"has_command"`
	Command    *model_v2.Command `json:"command,omitempty"` // 直接使用 model_v2.Command
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
