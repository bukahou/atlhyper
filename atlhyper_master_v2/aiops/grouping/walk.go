// atlhyper_master_v2/aiops/grouping/walk.go
// 在依赖图上从一个事件的成员出发做广度优先搜索：找能走到的其他事件，以及到某个事件的最短路径
package grouping

import "AtlHyper/atlhyper_master_v2/aiops"

// walker 一次归组用到的索引：实体 → 所属事件、边 → 类型
type walker struct {
	adjacency  map[string][]string
	edgeType   map[[2]string]string
	owner      map[string]int // 实体键 → 候选事件下标
	candidates []Candidate
}

func newWalker(graph *aiops.DependencyGraph, candidates []Candidate) *walker {
	w := &walker{
		adjacency:  graph.Adjacency(),
		edgeType:   make(map[[2]string]string, len(graph.Edges)),
		owner:      map[string]int{},
		candidates: candidates,
	}
	for _, edge := range graph.Edges {
		w.edgeType[[2]string{edge.From, edge.To}] = edge.Type
	}
	for i, c := range candidates {
		for _, member := range c.Members {
			w.owner[member] = i
		}
	}
	return w
}

// bfs 从事件 from 的成员出发沿依赖方向走，最多 depth 步。
// 每到一个实体调用 visit(实体, 父实体映射)；visit 返回 true 时停止
func (w *walker) bfs(from, depth int, visit func(entity string, parent map[string]string) bool) {
	parent := map[string]string{}
	seen := map[string]bool{}
	frontier := make([]string, 0, len(w.candidates[from].Members))
	for _, member := range w.candidates[from].Members {
		if !seen[member] {
			seen[member] = true
			frontier = append(frontier, member)
		}
	}
	for step := 0; step < depth && len(frontier) > 0; step++ {
		var next []string
		for _, entity := range frontier {
			for _, dep := range w.adjacency[entity] {
				if seen[dep] {
					continue
				}
				seen[dep] = true
				parent[dep] = entity
				if visit(dep, parent) {
					return
				}
				next = append(next, dep)
			}
		}
		frontier = next
	}
}

// reachableCandidates 从事件 from 沿依赖方向 depth 步内能走到的其他事件（按首次走到的先后）
func (w *walker) reachableCandidates(from, depth int) []int {
	var found []int
	hit := map[int]bool{}
	w.bfs(from, depth, func(entity string, _ map[string]string) bool {
		if owner, ok := w.owner[entity]; ok && owner != from && !hit[owner] {
			hit[owner] = true
			found = append(found, owner)
		}
		return false
	})
	return found
}

// pathBetween 从事件 from 的成员走到事件 to 的成员的最短路径（起点的边为空串）；走不到时为空
func (w *walker) pathBetween(from, to int) []aiops.LinkPathStep {
	var path []aiops.LinkPathStep
	w.bfs(from, pathDepth, func(entity string, parent map[string]string) bool {
		if owner, ok := w.owner[entity]; !ok || owner != to {
			return false
		}
		// 从终点沿父指针倒推回起点
		var reversed []aiops.LinkPathStep
		for current := entity; ; {
			prev, ok := parent[current]
			if !ok {
				reversed = append(reversed, aiops.LinkPathStep{EntityKey: current})
				break
			}
			reversed = append(reversed, aiops.LinkPathStep{EntityKey: current, Edge: w.edgeType[[2]string{prev, current}]})
			current = prev
		}
		for i := len(reversed) - 1; i >= 0; i-- {
			path = append(path, reversed[i])
		}
		return true
	})
	return path
}
