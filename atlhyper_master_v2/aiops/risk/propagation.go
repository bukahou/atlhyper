// atlhyper_master_v2/aiops/risk/propagation.go
// Stage 3: 沿依赖图传播风险
// 按层级排序: Node(先算) → Pod → Service → Ingress(后算)
// R_final(v) = α × R_weighted(v) + (1-α) × avg(R_final of dependencies)
package risk

import (
	"sort"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Propagate 沿依赖图传播风险
func Propagate(
	graph *aiops.DependencyGraph,
	weightedRisks map[string]float64,
	selfWeight float64,
) (map[string]float64, []*aiops.PropagationPath) {
	finalRisks := make(map[string]float64, len(graph.Nodes))
	var paths []*aiops.PropagationPath

	// 拓扑排序（按层级: node=0, pod=1, service=2, ingress=3）
	sorted := topologicalSort(graph)

	// 预建边索引: from -> edges
	edgeByFrom := make(map[string][]*aiops.GraphEdge)
	for _, edge := range graph.Edges {
		edgeByFrom[edge.From] = append(edgeByFrom[edge.From], edge)
	}

	// 从底层（Node）到顶层（Ingress）依次计算
	for _, entityKey := range sorted {
		rWeighted := weightedRisks[entityKey]

		// 获取该实体的下游依赖（adjacency: from -> [to...]）
		deps := graph.Adjacency()[entityKey]
		var propagatedRisk float64
		if len(deps) > 0 {
			var totalWeight float64
			for _, depKey := range deps {
				// 获取边权重
				edgeWeight := 1.0 / float64(len(deps))
				for _, edge := range edgeByFrom[entityKey] {
					if edge.To == depKey && edge.Weight > 0 {
						edgeWeight = edge.Weight
						break
					}
				}
				propagatedRisk += edgeWeight * finalRisks[depKey]
				totalWeight += edgeWeight

				if finalRisks[depKey] > 0 {
					// 找边类型
					edgeType := ""
					for _, edge := range edgeByFrom[entityKey] {
						if edge.To == depKey {
							edgeType = edge.Type
							break
						}
					}
					paths = append(paths, &aiops.PropagationPath{
						From:         depKey,
						To:           entityKey,
						EdgeType:     edgeType,
						Contribution: edgeWeight * finalRisks[depKey],
					})
				}
			}
			if totalWeight > 0 {
				propagatedRisk /= totalWeight
			}
		}

		// R_final = max(R_weighted, α × R_weighted + (1-α) × propagated)
		// max 确保传播不会稀释实体自身风险（健康依赖不应压低本体异常）
		mixed := selfWeight*rWeighted + (1-selfWeight)*propagatedRisk
		if rWeighted > mixed {
			finalRisks[entityKey] = rWeighted
		} else {
			finalRisks[entityKey] = mixed
		}
		if finalRisks[entityKey] > 1.0 {
			finalRisks[entityKey] = 1.0
		}
	}

	return finalRisks, paths
}

// topologicalSort 按层级排序
func topologicalSort(graph *aiops.DependencyGraph) []string {
	layerOrder := map[string]int{
		"node":    0,
		"pod":     1,
		"service": 2,
		"ingress": 3,
	}

	type entry struct {
		key   string
		layer int
	}
	entries := make([]entry, 0, len(graph.Nodes))
	for key, node := range graph.Nodes {
		layer := layerOrder[node.Type]
		entries = append(entries, entry{key, layer})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].layer < entries[j].layer
	})

	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.key
	}
	return result
}
