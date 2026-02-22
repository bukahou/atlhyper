// Package sdk 封装外部客户端
//
// types.go - SDK 层公共类型定义
//
// 本文件定义 SDK 层使用的各种选项和数据类型。
// 这些类型被接口方法使用，也被实现层使用。
package sdk

// =============================================================================
// 查询选项
// =============================================================================

// ListOptions 列表查询选项
type ListOptions struct {
	LabelSelector string // 标签选择器，如 "app=nginx"
	FieldSelector string // 字段选择器，如 "status.phase=Running"
	Limit         int64  // 限制返回数量
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	GracePeriodSeconds *int64 // 优雅终止时间 (秒)
	Force              bool   // 是否强制删除
}

// LogOptions 日志选项
type LogOptions struct {
	Container    string // 容器名称 (多容器 Pod 需指定)
	TailLines    int64  // 返回最后 N 行
	SinceSeconds int64  // 返回最近 N 秒的日志
	Timestamps   bool   // 是否包含时间戳
	Previous     bool   // 是否获取之前容器的日志
}

// =============================================================================
// 资源标识
// =============================================================================

// GroupVersionKind 资源类型标识
type GroupVersionKind struct {
	Group   string // API 组，如 "apps"
	Version string // API 版本，如 "v1"
	Kind    string // 资源类型，如 "Deployment"
}

// =============================================================================
// 动态请求/响应
// =============================================================================

// DynamicRequest 动态请求 (仅 GET)
type DynamicRequest struct {
	Path  string            // API 路径
	Query map[string]string // 查询参数
}

// DynamicResponse 动态响应
type DynamicResponse struct {
	StatusCode int    // HTTP 状态码
	Body       []byte // 响应体
}

// =============================================================================
// Metrics 数据
// =============================================================================

// NodeMetrics Node 资源使用量
//
// 来自 metrics-server，包含 CPU 和内存的实时使用量
type NodeMetrics struct {
	CPU    string // CPU 使用量，如 "2300m" (2.3核)
	Memory string // 内存使用量，如 "18534Mi"
}

// PodMetrics Pod 资源使用量
//
// 来自 metrics-server，包含各容器的 CPU 和内存使用量
type PodMetrics struct {
	Namespace  string             // Pod 命名空间
	Name       string             // Pod 名称
	Containers []ContainerMetrics // 各容器的资源使用量
}

// ContainerMetrics 容器资源使用量
type ContainerMetrics struct {
	Name   string // 容器名称
	CPU    string // CPU 使用量，如 "100m"
	Memory string // 内存使用量，如 "128Mi"
}
