package model

import "time"

// Command 指令定义
//
// Master 下发给 Agent 执行的操作指令。
// Agent 轮询获取指令，执行后上报结果。
//
// 指令类型 (Action):
//   - scale: 扩缩容 Deployment
//   - restart: 重启 Deployment
//   - delete: 删除资源
//   - get_logs: 获取 Pod 日志
//   - dynamic: 动态 API 调用 (AI 用)
type Command struct {
	// 标识
	ID        string `json:"id"`         // 指令唯一 ID
	ClusterID string `json:"cluster_id"` // 目标集群

	// 指令内容
	Action    string         `json:"action"`              // 操作类型 (见常量定义)
	Kind      string         `json:"kind"`                // 资源类型 (Pod, Deployment 等)
	Namespace string         `json:"namespace"`           // 目标命名空间
	Name      string         `json:"name"`                // 目标资源名称
	Params    map[string]any `json:"params"`              // 额外参数 (随 Action 不同而变化)
	Source    string         `json:"source,omitempty"`    // 来源: "ai" / "web"

	// 时间
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// Result 指令执行结果
//
// Agent 执行指令后上报给 Master
type Result struct {
	// 关联指令
	CommandID string `json:"command_id"` // 对应的指令 ID

	// 执行结果
	Success bool   `json:"success"`           // 是否成功
	Output  string `json:"output,omitempty"`  // 返回数据 (如日志内容)
	Error   string `json:"error,omitempty"`   // 错误信息

	// 时间
	ExecutedAt time.Time `json:"executed_at"` // 执行时间
}

// =============================================================================
// Action 常量
// =============================================================================

const (
	ActionScale       = "scale"        // 扩缩容 (Params: replicas)
	ActionRestart     = "restart"      // 重启
	ActionDelete      = "delete"       // 删除
	ActionGetLogs     = "get_logs"     // 获取日志 (Params: container, tailLines)
	ActionCreate      = "create"       // 创建
	ActionUpdate      = "update"       // 更新
	ActionUpdateImage = "update_image" // 更新镜像 (Params: container, image)
	ActionDynamic     = "dynamic"      // 动态 API (Params: method, path, body)
	ActionExec        = "exec"         // 执行命令 (Params: command)
	ActionDescribe    = "describe"     // 描述资源
	ActionApply       = "apply"        // 应用配置
	ActionRollback    = "rollback"     // 回滚
	ActionCordon      = "cordon"       // 标记 Node 不可调度
	ActionUncordon    = "uncordon"     // 取消 Node 不可调度
	ActionGetConfigMap = "get_configmap" // 获取 ConfigMap 数据
	ActionGetSecret    = "get_secret"    // 获取 Secret 数据
)

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
