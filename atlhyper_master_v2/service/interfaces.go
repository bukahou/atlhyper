// atlhyper_master_v2/service/interfaces.go
// Service 接口定义
// Query: 只读查询 (datahub.Store + mq.Producer 读取)
// Ops:   写入操作 (mq.Producer 写入)
package service

import (
	"context"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/model_v2"
)

// Query 只读查询接口
type Query interface {
	// ==================== 集群查询 ====================

	ListClusters(ctx context.Context) ([]model_v2.ClusterInfo, error)
	GetCluster(ctx context.Context, clusterID string) (*model_v2.ClusterDetail, error)

	// ==================== 快照查询 ====================

	GetSnapshot(ctx context.Context, clusterID string) (*model_v2.ClusterSnapshot, error)
	GetPods(ctx context.Context, clusterID string, opts model.PodQueryOpts) ([]model_v2.Pod, error)
	GetNodes(ctx context.Context, clusterID string) ([]model_v2.Node, error)
	GetDeployments(ctx context.Context, clusterID string, namespace string) ([]model_v2.Deployment, error)
	GetServices(ctx context.Context, clusterID string, namespace string) ([]model_v2.Service, error)
	GetIngresses(ctx context.Context, clusterID string, namespace string) ([]model_v2.Ingress, error)
	GetConfigMaps(ctx context.Context, clusterID string, namespace string) ([]model_v2.ConfigMap, error)
	GetSecrets(ctx context.Context, clusterID string, namespace string) ([]model_v2.Secret, error)
	GetNamespaces(ctx context.Context, clusterID string) ([]model_v2.Namespace, error)
	GetDaemonSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.DaemonSet, error)
	GetStatefulSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.StatefulSet, error)

	// ==================== Event 查询 ====================

	GetEvents(ctx context.Context, clusterID string, opts model.EventQueryOpts) ([]model_v2.Event, error)
	GetEventsByResource(ctx context.Context, clusterID, kind, namespace, name string) ([]model_v2.Event, error)

	// ==================== Agent / 指令状态查询 ====================

	GetAgentStatus(ctx context.Context, clusterID string) (*model_v2.AgentStatus, error)
	GetCommandStatus(ctx context.Context, commandID string) (*model.CommandStatus, error)

	// ==================== 概览 ====================

	GetOverview(ctx context.Context, clusterID string) (*model_v2.ClusterOverview, error)
}

// Ops 写入操作接口
type Ops interface {
	CreateCommand(req *operations.CreateCommandRequest) (*operations.CreateCommandResponse, error)
}

// Service 组合接口 (master.go 持有)
type Service interface {
	Query
	Ops
}
