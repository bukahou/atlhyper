// atlhyper_master_v2/aiops/grouping/grouping.go
// 拓扑归并：依赖链上同一次故障的活跃事件归为一组，受影响事件挂到源头事件下（见 aiops-topology-grouping-design.md）。
//
// 规则（用户 2026-11-15 裁决 D2 / D3）：
//   - 下游 Y 的成员沿依赖方向（依赖图正向边，from 依赖 to）走不超过 maxDepth 步能到达上游 X 的成员
//   - X 的开始时刻不晚于 Y 的开始时刻 + startSkew（容忍采集与判定的先后误差）
//   - 源头 = 沿上游关系走到的最上游（没有活跃上游）；有多个时取开始最早的，再取风险高的；链式全部挂到最上游，不嵌套
//
// 纯函数：输入只有活跃事件与依赖图，调用方负责只传警告 / 事件中的事件
package grouping

import (
	"sort"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

const (
	maxDepth  = 4               // 下游到直接上游最多走几步
	pathDepth = 4 * maxDepth    // 下游到最上游（可能隔几层）的路径最多走几步
	startSkew = 2 * time.Minute // 上游最多可以比下游晚多久开始
)

// Candidate 一个活跃事件
type Candidate struct {
	IncidentID string
	SubjectKey string
	StartedAt  time.Time // 这一轮异常的起点
	Risk       float64
	Members    []string // 主体的成员实体键（工作负载 = 它的 Pod；Service / Node 等 = 自身）
}

// Link 受影响事件归到源头的结果
type Link struct {
	RootIncidentID string
	Path           []aiops.LinkPathStep
}

// Group 对一个集群的活跃事件归组。返回 受影响事件号 → 归到的源头与路径；独立事件与源头不在结果里
func Group(candidates []Candidate, graph *aiops.DependencyGraph) map[string]Link {
	links := map[string]Link{}
	if graph == nil || len(candidates) < 2 {
		return links
	}
	g := newWalker(graph, candidates)

	// 1. 每个事件的直接上游（依赖可达 + 时间先后）
	upstream := make([][]int, len(candidates))
	for y := range candidates {
		for _, x := range g.reachableCandidates(y, maxDepth) {
			if !candidates[x].StartedAt.After(candidates[y].StartedAt.Add(startSkew)) {
				upstream[y] = append(upstream[y], x)
			}
		}
	}

	// 2. 每个有上游的事件找最上游；源头是它自己则是独立事件
	for y := range candidates {
		if len(upstream[y]) == 0 {
			continue
		}
		root := pickRoot(candidates, topmost(y, upstream))
		if root == y {
			continue
		}
		links[candidates[y].IncidentID] = Link{
			RootIncidentID: candidates[root].IncidentID,
			Path:           g.pathBetween(y, root),
		}
	}
	return links
}

// topmost 沿上游关系走到的最上游事件（没有上游的）；链上有环、走不到这样的事件时，返回走到的全部（含自己）
func topmost(start int, upstream [][]int) []int {
	visited := map[int]bool{start: true}
	stack := []int{start}
	var tops, reached []int
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		reached = append(reached, current)
		if len(upstream[current]) == 0 {
			tops = append(tops, current)
			continue
		}
		for _, next := range upstream[current] {
			if !visited[next] {
				visited[next] = true
				stack = append(stack, next)
			}
		}
	}
	if len(tops) > 0 {
		return tops
	}
	return reached
}

// pickRoot 从若干候选里定源头：开始最早，再风险高，再事件号小（结果稳定）
func pickRoot(candidates []Candidate, options []int) int {
	sort.Slice(options, func(i, j int) bool {
		a, b := candidates[options[i]], candidates[options[j]]
		if !a.StartedAt.Equal(b.StartedAt) {
			return a.StartedAt.Before(b.StartedAt)
		}
		if a.Risk != b.Risk {
			return a.Risk > b.Risk
		}
		return a.IncidentID < b.IncidentID
	})
	return options[0]
}
