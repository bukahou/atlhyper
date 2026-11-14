// atlhyper_master_v2/service/query/cluster_registry.go
// QueryClusterRegistry 实现 — 集群管理页的列表：注册表 + 在线状态 + 认证状态
package query

import (
	"context"
	"sort"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/model/convert"
	agentmodel "AtlHyper/model_v3/agent"
)

// ClusterRegistryReader 集群注册表的只读部分。
// 查询层只拿读方法：登记、签发令牌等写操作在 service/operations。
type ClusterRegistryReader interface {
	GetByID(ctx context.Context, clusterID string) (*database.Cluster, error)
	List(ctx context.Context) ([]*database.Cluster, error)
}

// ClusterAuthStatus 各集群最近一次上报是否带了有效令牌。
// 由 Agent 入口记录；这里只声明查询层用到的那一个方法，不依赖入口的具体类型。
type ClusterAuthStatus interface {
	// IsAuthenticated known 为假表示该集群还没上报过
	IsAuthenticated(clusterID string) (authenticated, known bool)
}

// ListClusterRegistry 列出集群管理页的全部行，按集群 ID 排序。
//
// 行的来源是两侧的并集：已登记的集群（无论是否上报过），以及正在上报但尚未登记的集群
// （后者 Registered 为假，管理员可以直接登记收编）。
func (q *QueryService) ListClusterRegistry(ctx context.Context) ([]model.ClusterRegistryItem, error) {
	registered, err := q.clusterRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	agents, err := q.store.ListAgents()
	if err != nil {
		return nil, err
	}

	presenceByCluster := make(map[string]*agentmodel.AgentInfo, len(agents))
	for i := range agents {
		presenceByCluster[agents[i].ClusterID] = &agents[i]
	}

	items := make([]model.ClusterRegistryItem, 0, len(registered)+len(agents))
	for _, cluster := range registered {
		items = append(items, q.clusterRegistryItem(cluster, presenceByCluster[cluster.ClusterID]))
		delete(presenceByCluster, cluster.ClusterID)
	}
	// 剩下的是只在上报、没有登记的集群
	for _, presence := range presenceByCluster {
		items = append(items, q.clusterRegistryItem(nil, presence))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ClusterID < items[j].ClusterID })
	return items, nil
}

// GetClusterRegistryItem 取单个集群的那一行；既没登记也没上报过时返回 (nil, nil)
func (q *QueryService) GetClusterRegistryItem(ctx context.Context, clusterID string) (*model.ClusterRegistryItem, error) {
	registered, err := q.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	agents, err := q.store.ListAgents()
	if err != nil {
		return nil, err
	}

	var presence *agentmodel.AgentInfo
	for i := range agents {
		if agents[i].ClusterID == clusterID {
			presence = &agents[i]
			break
		}
	}
	if registered == nil && presence == nil {
		return nil, nil
	}
	item := q.clusterRegistryItem(registered, presence)
	return &item, nil
}

// clusterRegistryItem 补上认证状态后交给转换层。registered 与 presence 至少有一个非 nil
func (q *QueryService) clusterRegistryItem(registered *database.Cluster, presence *agentmodel.AgentInfo) model.ClusterRegistryItem {
	clusterID := ""
	switch {
	case registered != nil:
		clusterID = registered.ClusterID
	case presence != nil:
		clusterID = presence.ClusterID
	}
	return convert.ClusterRegistryItem(registered, presence, q.isClusterAuthenticated(clusterID))
}

// isClusterAuthenticated 没上报过、或没有接入认证状态时一律为假
func (q *QueryService) isClusterAuthenticated(clusterID string) bool {
	if q.clusterAuthStatus == nil {
		return false
	}
	authenticated, known := q.clusterAuthStatus.IsAuthenticated(clusterID)
	return known && authenticated
}
