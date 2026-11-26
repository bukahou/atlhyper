package tools

import (
	"context"
	"errors"
	"fmt"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/model_v3/agent"
)

// ErrNoClusterData 这个集群还没有上报过数据（或已离线、快照过期）
var ErrNoClusterData = errors.New("该集群还没有数据")

type listClustersParams struct{}

type clusterOverviewParams struct{}

// overviewTools 集群列表与概览：不深入任何资源，只回答「有哪些集群、整体怎么样」
func overviewTools(d Deps) []*toolkit.Tool {
	return []*toolkit.Tool{
		toolkit.MustDefine(toolkit.Spec[listClustersParams]{
			Name:        "list_clusters",
			Description: "列出正在上报的集群：集群 ID、在线状态、最近上报时间、节点与 Pod 数量。不知道有哪些集群、或要确认某个集群是否在线时先用它。",
			Permission:  toolkit.PermClusterRead,
			NoCluster:   true,
			Handler: func(ctx context.Context, _ string, _ listClustersParams) (string, error) {
				clusters, err := d.Clusters.ListClusters(ctx)
				if err != nil {
					return "", fmt.Errorf("列出集群失败: %w", err)
				}
				if clusters == nil {
					clusters = []agent.ClusterInfo{}
				}
				return marshal(map[string]any{"clusters": clusters, "total": len(clusters)})
			},
		}),
		toolkit.MustDefine(toolkit.Spec[clusterOverviewParams]{
			Name: "get_cluster_overview",
			Description: "集群概览：健康状态与原因、节点与 Pod 的就绪情况、各类工作负载数量、最近的告警、各节点的资源用量。" +
				"想知道「集群现在整体怎么样」时先用它；要找出最可疑的实体用 get_cluster_risk。",
			Permission: toolkit.PermClusterRead,
			Handler: func(ctx context.Context, clusterID string, _ clusterOverviewParams) (string, error) {
				overview, err := d.Overview.GetOverview(ctx, clusterID)
				if err != nil {
					return "", fmt.Errorf("读取集群概览失败: %w", err)
				}
				if overview == nil {
					return "", fmt.Errorf("%w：%s", ErrNoClusterData, clusterID)
				}
				return marshal(overview)
			},
		}),
	}
}
