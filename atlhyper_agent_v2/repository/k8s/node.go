package k8s

import (
	"context"
	"math"
	"strconv"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
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
func (r *nodeRepository) List(ctx context.Context, opts model.ListOptions) ([]cluster.Node, error) {
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
	nodes := make([]cluster.Node, 0, len(k8sNodes))
	for i := range k8sNodes {
		node := ConvertNode(&k8sNodes[i])

		// 合并 metrics 数据 + Pressure
		pressure := extractPressure(node.Conditions)
		if metrics, ok := metricsMap[node.GetName()]; ok {
			node.Metrics = &cluster.NodeResourceUsage{
				CPU: cluster.NodeResourceMetric{
					Usage:       metrics.CPU,
					Allocatable: node.Allocatable.CPU,
					Capacity:    node.Capacity.CPU,
					UtilPct:     calcUtilPct(model_v3.ParseCPU(metrics.CPU), model_v3.ParseCPU(node.Allocatable.CPU)),
				},
				Memory: cluster.NodeResourceMetric{
					Usage:       metrics.Memory,
					Allocatable: node.Allocatable.Memory,
					Capacity:    node.Capacity.Memory,
					UtilPct:     calcUtilPct(model_v3.ParseMemory(metrics.Memory), model_v3.ParseMemory(node.Allocatable.Memory)),
				},
				Pressure: pressure,
			}
		} else {
			// 即使无 Metrics Server，也填充 Pressure（来自 Node Conditions）
			node.Metrics = &cluster.NodeResourceUsage{
				Pressure: pressure,
			}
		}

		nodes = append(nodes, node)
	}
	return nodes, nil
}

// Get 获取单个 Node
func (r *nodeRepository) Get(ctx context.Context, name string) (*cluster.Node, error) {
	k8sNode, err := r.client.GetNode(ctx, name)
	if err != nil {
		return nil, err
	}
	node := ConvertNode(k8sNode)
	return &node, nil
}

// calcUtilPct 计算使用率百分比（保留 2 位小数）
func calcUtilPct(usage, total int64) float64 {
	if total <= 0 {
		return 0
	}
	pct := float64(usage) / float64(total) * 100
	return math.Round(pct*100) / 100
}

// extractPressure 从 Node Conditions 提取压力标志
func extractPressure(conditions []cluster.NodeCondition) cluster.PressureFlags {
	var flags cluster.PressureFlags
	for _, c := range conditions {
		if c.Status != "True" {
			continue
		}
		switch c.Type {
		case "MemoryPressure":
			flags.MemoryPressure = true
		case "DiskPressure":
			flags.DiskPressure = true
		case "PIDPressure":
			flags.PIDPressure = true
		case "NetworkUnavailable":
			flags.NetworkUnavailable = true
		}
	}
	return flags
}

// parseInt 简单解析整数字符串
func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
