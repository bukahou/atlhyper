package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/model_v2"
)

// nodeRepository Node 仓库实现
type nodeRepository struct {
	client sdk.K8sClient
}

// NewNodeRepository 创建 Node 仓库
func NewNodeRepository(client sdk.K8sClient) repository.NodeRepository {
	return &nodeRepository{client: client}
}

// List 列出 Node
//
// 同时获取 metrics 数据，合并到 node.Usage 字段。
// 如果 metrics-server 不可用，Usage 为空。
func (r *nodeRepository) List(ctx context.Context, opts model.ListOptions) ([]model_v2.Node, error) {
	// 1. 获取 Node 列表
	k8sNodes, err := r.client.ListNodes(ctx, sdk.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	// 2. 获取 Node Metrics (可能为空)
	metricsMap, _ := r.client.ListNodeMetrics(ctx)

	// 3. 转换并合并数据
	nodes := make([]model_v2.Node, 0, len(k8sNodes))
	for i := range k8sNodes {
		node := ConvertNode(&k8sNodes[i])

		// 合并 metrics 数据
		if metrics, ok := metricsMap[node.GetName()]; ok {
			node.Metrics = &model_v2.NodeMetrics{
				CPU: model_v2.NodeResourceMetric{
					Usage:       metrics.CPU,
					Allocatable: node.Allocatable.CPU,
					Capacity:    node.Capacity.CPU,
				},
				Memory: model_v2.NodeResourceMetric{
					Usage:       metrics.Memory,
					Allocatable: node.Allocatable.Memory,
					Capacity:    node.Capacity.Memory,
				},
			}
		}

		nodes = append(nodes, node)
	}
	return nodes, nil
}

// Get 获取单个 Node
func (r *nodeRepository) Get(ctx context.Context, name string) (*model_v2.Node, error) {
	k8sNode, err := r.client.GetNode(ctx, name)
	if err != nil {
		return nil, err
	}
	node := ConvertNode(k8sNode)
	return &node, nil
}
