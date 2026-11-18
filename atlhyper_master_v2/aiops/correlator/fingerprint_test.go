package correlator

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

func graphOf(updatedAt time.Time, nodes []string, edges [][3]string) *aiops.DependencyGraph {
	g := aiops.NewDependencyGraph("c1")
	g.UpdatedAt = updatedAt
	for _, key := range nodes {
		g.Nodes[key] = &aiops.GraphNode{Key: key, Type: "pod", Namespace: "default", Name: key}
	}
	for _, e := range edges {
		g.Edges = append(g.Edges, &aiops.GraphEdge{From: e[0], To: e[1], Type: e[2], Weight: 1})
	}
	return g
}

// 指纹只反映拓扑：构建时间不同、边的排列顺序不同，都算同一张图。
func TestFingerprint_IgnoresTimestampAndEdgeOrder(t *testing.T) {
	a := graphOf(time.Unix(100, 0), []string{"a", "b", "c"}, [][3]string{{"a", "b", "calls"}, {"b", "c", "runs_on"}})
	b := graphOf(time.Unix(999, 0), []string{"c", "b", "a"}, [][3]string{{"b", "c", "runs_on"}, {"a", "b", "calls"}})
	if Fingerprint(a) != Fingerprint(b) {
		t.Error("同一拓扑的指纹应相同")
	}
}

func TestFingerprint_ChangesWithTopology(t *testing.T) {
	base := graphOf(time.Unix(100, 0), []string{"a", "b"}, [][3]string{{"a", "b", "calls"}})
	tests := []struct {
		name  string
		other *aiops.DependencyGraph
	}{
		{"多一个节点", graphOf(time.Unix(100, 0), []string{"a", "b", "c"}, [][3]string{{"a", "b", "calls"}})},
		{"少一条边", graphOf(time.Unix(100, 0), []string{"a", "b"}, nil)},
		{"边的方向反了", graphOf(time.Unix(100, 0), []string{"a", "b"}, [][3]string{{"b", "a", "calls"}})},
		{"边的类型变了", graphOf(time.Unix(100, 0), []string{"a", "b"}, [][3]string{{"a", "b", "selects"}})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Fingerprint(base) == Fingerprint(tt.other) {
				t.Error("拓扑变了，指纹应不同")
			}
		})
	}
}

// 节点的类型或元数据变化也要重新保存
func TestFingerprint_ChangesWithNodeFields(t *testing.T) {
	a := graphOf(time.Unix(100, 0), []string{"a"}, nil)
	b := graphOf(time.Unix(100, 0), []string{"a"}, nil)
	b.Nodes["a"].Metadata = map[string]string{"owner": "x"}
	if Fingerprint(a) == Fingerprint(b) {
		t.Error("节点元数据变了，指纹应不同")
	}
}
