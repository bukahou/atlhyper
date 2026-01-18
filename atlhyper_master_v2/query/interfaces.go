// atlhyper_master_v2/query/interfaces.go
// Query 层接口定义
// Query 层是查询抽象层，屏蔽 DataHub 底层实现
// 所有外部访问（Gateway）禁止直接访问 DataHub，必须通过 Query 层
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// Query 查询接口
// 提供统一的查询 API，屏蔽 DataHub 具体实现
type Query interface {
	// ==================== 集群查询 ====================

	// ListClusters 列出所有集群
	ListClusters(ctx context.Context) ([]model_v2.ClusterInfo, error)

	// GetCluster 获取集群详情
	GetCluster(ctx context.Context, clusterID string) (*model_v2.ClusterDetail, error)

	// ==================== 快照查询 ====================

	// GetSnapshot 获取集群快照
	GetSnapshot(ctx context.Context, clusterID string) (*model_v2.ClusterSnapshot, error)

	// GetPods 获取 Pod 列表
	GetPods(ctx context.Context, clusterID string, opts model.PodQueryOpts) ([]model_v2.Pod, error)

	// GetNodes 获取 Node 列表
	GetNodes(ctx context.Context, clusterID string) ([]model_v2.Node, error)

	// GetDeployments 获取 Deployment 列表
	GetDeployments(ctx context.Context, clusterID string, namespace string) ([]model_v2.Deployment, error)

	// GetServices 获取 Service 列表
	GetServices(ctx context.Context, clusterID string, namespace string) ([]model_v2.Service, error)

	// GetIngresses 获取 Ingress 列表
	GetIngresses(ctx context.Context, clusterID string, namespace string) ([]model_v2.Ingress, error)

	// GetConfigMaps 获取 ConfigMap 列表
	GetConfigMaps(ctx context.Context, clusterID string, namespace string) ([]model_v2.ConfigMap, error)

	// GetSecrets 获取 Secret 列表
	GetSecrets(ctx context.Context, clusterID string, namespace string) ([]model_v2.Secret, error)

	// GetNamespaces 获取 Namespace 列表
	GetNamespaces(ctx context.Context, clusterID string) ([]model_v2.Namespace, error)

	// GetDaemonSets 获取 DaemonSet 列表
	GetDaemonSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.DaemonSet, error)

	// GetStatefulSets 获取 StatefulSet 列表
	GetStatefulSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.StatefulSet, error)

	// ==================== Event 查询 ====================

	// GetEvents 获取实时 Events（从 DataHub）
	GetEvents(ctx context.Context, clusterID string, opts model.EventQueryOpts) ([]model_v2.Event, error)

	// GetEventsByResource 按资源查询 Events
	GetEventsByResource(ctx context.Context, clusterID, kind, namespace, name string) ([]model_v2.Event, error)

	// ==================== Agent 状态查询 ====================

	// GetAgentStatus 获取 Agent 状态
	GetAgentStatus(ctx context.Context, clusterID string) (*model_v2.AgentStatus, error)

	// ==================== 指令状态查询 ====================

	// GetCommandStatus 获取指令状态
	GetCommandStatus(ctx context.Context, commandID string) (*model.CommandStatus, error)

	// ==================== 概览查询 ====================

	// GetOverview 获取集群概览
	GetOverview(ctx context.Context, clusterID string) (*model_v2.ClusterOverview, error)
}
