// sdk/provider.go
// SDKProvider 聚合接口定义
package sdk

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SDKProvider 基础设施 SDK 提供者
// 聚合所有资源操作能力，是 SDK 的核心入口
type SDKProvider interface {
	// ==================== 资源读取 ====================
	ResourceLister
	ResourceGetter

	// ==================== 资源操作 ====================
	// Pods 返回 Pod 操作接口
	Pods() PodOperator

	// Nodes 返回 Node 操作接口
	Nodes() NodeOperator

	// Deployments 返回 Deployment 操作接口
	Deployments() DeploymentOperator

	// ==================== 指标 ====================
	// Metrics 返回指标提供者
	Metrics() MetricsProvider

	// ==================== 集群信息 ====================
	// Cluster 返回集群信息接口
	Cluster() ClusterInfo

	// ==================== 底层客户端访问 ====================
	// RuntimeClient 返回 controller-runtime client (用于 Watcher)
	RuntimeClient() client.Client

	// CoreClient 返回 client-go clientset (用于原生 API 操作)
	CoreClient() *kubernetes.Clientset

	// MetricsClient 返回 metrics clientset (用于 metrics-server API 操作)
	// 如果 metrics-server 不可用，返回 nil
	MetricsClient() *metricsclient.Clientset

	// HasMetricsServer 检查 metrics-server 是否可用
	HasMetricsServer() bool

	// RestConfig 返回 REST 配置
	RestConfig() *rest.Config

	// ==================== 生命周期 ====================
	// Close 关闭提供者，释放资源
	Close() error
}
