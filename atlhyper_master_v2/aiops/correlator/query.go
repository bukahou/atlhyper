// atlhyper_master_v2/aiops/correlator/query.go
// 依赖图管理 + BFS 遍历查询 + calls 边缓存
package correlator

import (
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/common/logger"
)

var corrLog = logger.Module("Correlator")

// calls 边缓存 TTL：最后一次观察到流量后保持 1 小时
// 低流量服务间可能长时间无请求，10 分钟太短会导致拓扑图连接频繁断裂
const defaultEdgeTTL = 1 * time.Hour

// callsEdgeEntry 缓存的 calls 边（service → service 调用关系）
type callsEdgeEntry struct {
	srcKey       string
	dstKey       string
	srcNamespace string
	srcName      string
	dstNamespace string
	dstName      string
	lastSeen     time.Time
}

// Correlator 依赖图管理器
type Correlator struct {
	mu     sync.RWMutex
	graphs map[string]*aiops.DependencyGraph // clusterID -> graph

	// calls 边缓存: 低流量时 delta=0 导致 calls 边消失，缓存保持已知的调用关系
	edgeCache map[string]map[string]*callsEdgeEntry // clusterID -> (from|to -> entry)
	edgeTTL   time.Duration
}

// NewCorrelator 创建依赖图管理器
func NewCorrelator() *Correlator {
	return &Correlator{
		graphs:    make(map[string]*aiops.DependencyGraph),
		edgeCache: make(map[string]map[string]*callsEdgeEntry),
		edgeTTL:   defaultEdgeTTL,
	}
}

// Update 更新指定集群的依赖图
//
// 除了存储新图外，还维护 calls 边缓存：
//  1. 从新图中提取 calls 边更新缓存（记录 lastSeen）
//  2. 将缓存中未过期的 calls 边注入新图（补充低流量丢失的调用关系）
//  3. 清理过期缓存
func (c *Correlator) Update(clusterID string, newGraph *aiops.DependencyGraph) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// 初始化集群边缓存
	if c.edgeCache[clusterID] == nil {
		c.edgeCache[clusterID] = make(map[string]*callsEdgeEntry)
	}
	cache := c.edgeCache[clusterID]

	// 1. 从新图中提取 calls 边，更新缓存 lastSeen
	currentCalls := make(map[string]bool)
	for _, edge := range newGraph.Edges {
		if edge.Type != "calls" {
			continue
		}
		eKey := edge.From + "|" + edge.To
		currentCalls[eKey] = true

		entry := &callsEdgeEntry{
			srcKey:   edge.From,
			dstKey:   edge.To,
			lastSeen: now,
		}
		if srcNode := newGraph.Nodes[edge.From]; srcNode != nil {
			entry.srcNamespace = srcNode.Namespace
			entry.srcName = srcNode.Name
		}
		if dstNode := newGraph.Nodes[edge.To]; dstNode != nil {
			entry.dstNamespace = dstNode.Namespace
			entry.dstName = dstNode.Name
		}
		cache[eKey] = entry
	}

	// 2. 注入缓存中未过期且不在新图中的 calls 边
	injected := 0
	for eKey, entry := range cache {
		if currentCalls[eKey] {
			continue
		}

		expired := now.Sub(entry.lastSeen) > c.edgeTTL
		if expired {
			// 过期且两端服务已从集群中消失 → 清理
			// 两端仍存在说明服务还活着，只是暂时无流量，保留缓存
			srcExists := newGraph.Nodes[entry.srcKey] != nil
			dstExists := newGraph.Nodes[entry.dstKey] != nil
			if !srcExists || !dstExists {
				delete(cache, eKey)
				continue
			}
		}

		// 确保两端节点存在
		newGraph.AddNode(entry.srcKey, "service", entry.srcNamespace, entry.srcName, nil)
		newGraph.AddNode(entry.dstKey, "service", entry.dstNamespace, entry.dstName, nil)
		newGraph.AddEdge(entry.srcKey, entry.dstKey, "calls", 1.0)
		injected++
	}

	if injected > 0 {
		newGraph.RebuildIndex()
		corrLog.Debug("注入缓存 calls 边", "cluster", clusterID, "injected", injected, "cacheSize", len(cache))
	}

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
