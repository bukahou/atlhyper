package correlator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Fingerprint 依赖图的拓扑指纹：节点与边相同的两张图指纹相同。
//
// 不含构建时间，也不受节点 / 边的排列顺序影响 —— 依赖图每份快照都重建一次，
// 调用方用指纹判断拓扑是否真的变了。
func Fingerprint(graph *aiops.DependencyGraph) string {
	lines := make([]string, 0, len(graph.Nodes)+len(graph.Edges))
	for key, node := range graph.Nodes {
		lines = append(lines, fmt.Sprintf("n\x00%s\x00%s\x00%s\x00%s\x00%s",
			key, node.Type, node.Namespace, node.Name, sortedPairs(node.Metadata)))
	}
	for _, edge := range graph.Edges {
		lines = append(lines, fmt.Sprintf("e\x00%s\x00%s\x00%s\x00%g", edge.From, edge.To, edge.Type, edge.Weight))
	}
	sort.Strings(lines)

	hash := sha256.New()
	for _, line := range lines {
		hash.Write([]byte(line))
		hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func sortedPairs(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}
