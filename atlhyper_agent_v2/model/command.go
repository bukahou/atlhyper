package model

import "time"

// =============================================================================
// 注意：Command 和 Action 常量已迁移到 model_v2.Command
// 本文件只保留 Agent 内部使用的类型
// =============================================================================

// Result 指令执行结果
//
// Agent 执行指令后上报给 Master
type Result struct {
	// 关联指令
	CommandID string `json:"command_id"` // 对应的指令 ID

	// 执行结果
	Success bool   `json:"success"`          // 是否成功
	Output  string `json:"output,omitempty"` // 返回数据 (如日志内容)
	Error   string `json:"error,omitempty"`  // 错误信息

	// 时间
	ExecutedAt time.Time `json:"executed_at"` // 执行时间
}

// =============================================================================
// 动态请求/响应 (AI 只读查询)
// =============================================================================

// DynamicRequest 动态 API 请求
//
// 用于 AI 发起 K8s API 只读查询 (仅 GET)
// 安全限制: 不支持写操作
type DynamicRequest struct {
	Path  string            `json:"path"`  // API 路径 (如 /api/v1/namespaces/default/pods)
	Query map[string]string `json:"query"` // 查询参数
}

// DynamicResponse 动态 API 响应
type DynamicResponse struct {
	StatusCode int    `json:"status_code"` // HTTP 状态码
	Body       []byte `json:"body"`        // 响应体
}
