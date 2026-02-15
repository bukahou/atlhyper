// atlhyper_master_v2/aiops/correlator/query.go
// 依赖图管理 + BFS 遍历查询
package correlator

import (
	"sync"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Correlator 依赖图管理器
type Correlator struct {
	mu     sync.RWMutex
	graphs map[string]*aiops.DependencyGraph // clusterID -> graph
}

// NewCorrelator 创建依赖图管理器
func NewCorrelator() *Correlator {
	return &Correlator{
		graphs: make(map[string]*aiops.DependencyGraph),
	}
}

// Update 更新指定集群的依赖图（全量替换）
func (c *Correlator) Update(clusterID string, newGraph *aiops.DependencyGraph) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.graphs[clusterID] = newGraph
}

// GetGraph 返回指定集群的完整依赖图
func (c *Correlator) GetGraph(clusterID string) *aiops.DependencyGraph {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.graphs[clusterID]
}

// ListClusters 返回所有集群 ID
func (c *Correlator) ListClusters() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.graphs))
	for id := range c.graphs {
		ids = append(ids, id)
	}
	return ids
}

// Trace 从指定实体出发，BFS 遍历上游或下游链路
func (c *Correlator) Trace(clusterID, fromKey, direction string, maxDepth int) *aiops.TraceResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	graph := c.graphs[clusterID]
	if graph == nil {
		return &aiops.TraceResult{}
	}

	if maxDepth <= 0 {
		maxDepth = 10
	}

	visited := make(map[string]bool)
	result := &aiops.TraceResult{
		Nodes: make([]*aiops.GraphNode, 0),
		Edges: make([]*aiops.GraphEdge, 0),
	}

	type bfsItem struct {
		key   string
		depth int
	}
	queue := []bfsItem{{fromKey, 0}}

	// 预建边索引，避免 O(N) 遍历
	edgeIndex := make(map[string][]*aiops.GraphEdge)
	for _, edge := range graph.Edges {
		if direction == "downstream" {
			edgeIndex[edge.From] = append(edgeIndex[edge.From], edge)
		} else {
			edgeIndex[edge.To] = append(edgeIndex[edge.To], edge)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.key] || current.depth > maxDepth {
			continue
		}
		visited[current.key] = true

		if node, ok := graph.Nodes[current.key]; ok {
			result.Nodes = append(result.Nodes, node)
		}

		if current.depth > result.Depth {
			result.Depth = current.depth
		}

		// 根据方向选择邻接表
		var neighbors []string
		if direction == "downstream" {
			neighbors = graph.Adjacency()[current.key]
		} else {
			neighbors = graph.Reverse()[current.key]
		}

		for _, neighbor := range neighbors {
			if visited[neighbor] {
				continue
			}
			queue = append(queue, bfsItem{neighbor, current.depth + 1})

			// 收集边
			for _, edge := range edgeIndex[current.key] {
				if direction == "downstream" && edge.To == neighbor {
					result.Edges = append(result.Edges, edge)
				} else if direction != "downstream" && edge.From == neighbor {
					result.Edges = append(result.Edges, edge)
				}
			}
		}
	}

	return result
}
