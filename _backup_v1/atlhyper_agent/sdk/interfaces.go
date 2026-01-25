// sdk/interfaces.go
// SDK 核心接口定义 - 平台无关的抽象层
package sdk

import "context"

// ==================== 资源标识 ====================

// ObjectKey 通用资源标识
type ObjectKey struct {
	Namespace string
	Name      string
}

// ==================== 资源读取接口 ====================

// ResourceLister 资源列表查询接口
type ResourceLister interface {
	// Pod
	ListPods(ctx context.Context, namespace string) ([]PodInfo, error)

	// Node
	ListNodes(ctx context.Context) ([]NodeInfo, error)

	// Deployment
	ListDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error)

	// Service
	ListServices(ctx context.Context, namespace string) ([]ServiceInfo, error)

	// Namespace
	ListNamespaces(ctx context.Context) ([]NamespaceInfo, error)

	// ConfigMap
	ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMapInfo, error)

	// Ingress
	ListIngresses(ctx context.Context, namespace string) ([]IngressInfo, error)
}

// ResourceGetter 单个资源获取接口
type ResourceGetter interface {
	GetPod(ctx context.Context, key ObjectKey) (*PodInfo, error)
	GetNode(ctx context.Context, name string) (*NodeInfo, error)
	GetDeployment(ctx context.Context, key ObjectKey) (*DeploymentInfo, error)
	GetNamespace(ctx context.Context, name string) (*NamespaceInfo, error)
}

// ==================== 资源操作接口 ====================

// PodOperator Pod 操作接口
type PodOperator interface {
	// RestartPod 重启 Pod（通过删除实现，由控制器自动重建）
	RestartPod(ctx context.Context, key ObjectKey) error

	// GetPodLogs 获取 Pod 日志
	GetPodLogs(ctx context.Context, key ObjectKey, opts LogOptions) (string, error)
}

// NodeOperator Node 操作接口
type NodeOperator interface {
	// CordonNode 禁止调度节点
	CordonNode(ctx context.Context, name string) error

	// UncordonNode 恢复调度节点
	UncordonNode(ctx context.Context, name string) error
}

// DeploymentOperator Deployment 操作接口
type DeploymentOperator interface {
	// ScaleDeployment 扩缩容
	ScaleDeployment(ctx context.Context, key ObjectKey, replicas int32) error

	// UpdateDeploymentImage 更新镜像
	UpdateDeploymentImage(ctx context.Context, key ObjectKey, newImage string) error
}

// ==================== 指标接口 ====================

// MetricsProvider 资源指标提供者
type MetricsProvider interface {
	// IsAvailable 检查指标服务是否可用
	IsAvailable() bool

	// GetPodMetrics 获取 Pod 指标
	GetPodMetrics(ctx context.Context, namespace string) (map[string]PodMetrics, error)

	// GetNodeMetrics 获取 Node 指标
	GetNodeMetrics(ctx context.Context) (map[string]NodeMetrics, error)
}

// ==================== 集群信息接口 ====================

// ClusterInfo 集群信息接口
type ClusterInfo interface {
	// GetClusterID 获取集群唯一标识
	GetClusterID(ctx context.Context) (string, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// ==================== 配置选项 ====================

// LogOptions 日志获取选项
type LogOptions struct {
	Container  string // 容器名（多容器时必填）
	TailLines  int64  // 返回最后 N 行
	Timestamps bool   // 是否包含时间戳
}

// ProviderConfig 提供者配置
type ProviderConfig struct {
	Kubeconfig string // kubeconfig 文件路径，空则使用 InCluster 模式
}
