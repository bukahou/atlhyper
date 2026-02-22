package model

// =============================================================================
// Agent 内部类型（不跨项目共享）
// =============================================================================

// DynamicRequest 动态 API 请求
type DynamicRequest struct {
	Path  string            `json:"path"`
	Query map[string]string `json:"query"`
}

// DynamicResponse 动态 API 响应
type DynamicResponse struct {
	StatusCode int    `json:"status_code"`
	Body       []byte `json:"body"`
}
