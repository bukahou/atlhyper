// atlhyper_master_v2/service/interfaces.go
// Service 接口定义
// Query: 只读查询 (datahub.Store + mq.Producer 读取)
// Ops:   写入操作 (mq.Producer 写入)
package service

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	aiopsai "AtlHyper/atlhyper_master_v2/aiops/ai"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/model_v2"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// ================================================================
// 子接口（按功能域划分）
// ================================================================

// QueryK8s K8s 资源快照查询
type QueryK8s interface {
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
	GetJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.Job, error)
	GetCronJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.CronJob, error)
	GetPersistentVolumes(ctx context.Context, clusterID string) ([]model_v2.PersistentVolume, error)
	GetPersistentVolumeClaims(ctx context.Context, clusterID string, namespace string) ([]model_v2.PersistentVolumeClaim, error)
	GetNetworkPolicies(ctx context.Context, clusterID string, namespace string) ([]model_v2.NetworkPolicy, error)
	GetResourceQuotas(ctx context.Context, clusterID string, namespace string) ([]model_v2.ResourceQuota, error)
	GetLimitRanges(ctx context.Context, clusterID string, namespace string) ([]model_v2.LimitRange, error)
	GetServiceAccounts(ctx context.Context, clusterID string, namespace string) ([]model_v2.ServiceAccount, error)
}

// QueryOTel OTel 快照/时间线查询
type QueryOTel interface {
	GetOTelSnapshot(ctx context.Context, clusterID string) (*cluster.OTelSnapshot, error)
	GetOTelTimeline(ctx context.Context, clusterID string, since time.Time) ([]cluster.OTelEntry, error)
	// QueryLogsFromSnapshot 从快照查询日志（过滤+分页+facets）
	QueryLogsFromSnapshot(ctx context.Context, clusterID string, opts model.LogSnapshotQueryOpts) (*model.LogSnapshotResult, error)
}

// QuerySLO SLO 服务网格查询
type QuerySLO interface {
	GetMeshTopology(ctx context.Context, clusterID, timeRange string) (*model.ServiceMeshTopologyResponse, error)
	GetServiceDetail(ctx context.Context, clusterID, namespace, name, timeRange string) (*model.ServiceDetailResponse, error)
}

// QueryAIOps AIOps 查询与 AI 增强
type QueryAIOps interface {
	GetAIOpsGraph(ctx context.Context, clusterID string) (*aiops.DependencyGraph, error)
	GetAIOpsGraphTrace(ctx context.Context, clusterID, fromKey, direction string, maxDepth int) (*aiops.TraceResult, error)
	GetAIOpsBaseline(ctx context.Context, clusterID, entityKey string) (*aiops.EntityBaseline, error)
	GetAIOpsClusterRisk(ctx context.Context, clusterID string) (*aiops.ClusterRisk, error)
	GetAIOpsEntityRisks(ctx context.Context, clusterID, sortBy string, limit int) ([]*aiops.EntityRisk, error)
	GetAIOpsEntityRisk(ctx context.Context, clusterID, entityKey string) (*aiops.EntityRiskDetail, error)
	GetAIOpsIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error)
	GetAIOpsIncidentDetail(ctx context.Context, incidentID string) (*aiops.IncidentDetail, error)
	GetAIOpsIncidentStats(ctx context.Context, clusterID string, since time.Time) (*aiops.IncidentStats, error)
	GetAIOpsIncidentPatterns(ctx context.Context, entityKey string, since time.Time) ([]*aiops.IncidentPattern, error)
	SummarizeIncident(ctx context.Context, incidentID string) (*aiopsai.SummarizeResponse, error)
}

// QueryOverview 集群概览、Agent 状态、事件、单资源查询
type QueryOverview interface {
	ListClusters(ctx context.Context) ([]model_v2.ClusterInfo, error)
	GetCluster(ctx context.Context, clusterID string) (*model_v2.ClusterDetail, error)
	GetAgentStatus(ctx context.Context, clusterID string) (*model_v2.AgentStatus, error)
	GetCommandStatus(ctx context.Context, commandID string) (*command.Status, error)
	GetOverview(ctx context.Context, clusterID string) (*model_v2.ClusterOverview, error)
	GetEvents(ctx context.Context, clusterID string, opts model.EventQueryOpts) ([]model_v2.Event, error)
	GetEventsByResource(ctx context.Context, clusterID, kind, namespace, name string) ([]model_v2.Event, error)
	// 单资源查询 (Event Alert Enrichment)
	GetPod(ctx context.Context, clusterID, namespace, name string) (*model_v2.Pod, error)
	GetNode(ctx context.Context, clusterID, name string) (*model_v2.Node, error)
	GetDeployment(ctx context.Context, clusterID, namespace, name string) (*model_v2.Deployment, error)
	GetDeploymentByReplicaSet(ctx context.Context, clusterID, namespace, rsName string) (*model_v2.Deployment, error)
}

// ================================================================
// 组合接口（向后兼容，现有代码无需修改）
// ================================================================

// Query 只读查询接口
type Query interface {
	QueryK8s
	QueryOTel
	QuerySLO
	QueryAIOps
	QueryOverview
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
