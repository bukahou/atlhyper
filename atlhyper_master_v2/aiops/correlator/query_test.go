// atlhyper_master_v2/aiops/correlator/query_test.go
package correlator

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

func makeTestGraph() *aiops.DependencyGraph {
	g := aiops.NewDependencyGraph("test")

	// ingress -> service -> pod -> node
	g.AddNode("_cluster/ingress/api.example.com", "ingress", "_cluster", "api.example.com", nil)
	g.AddNode("default/service/api-svc", "service", "default", "api-svc", nil)
	g.AddNode("default/pod/api-pod-1", "pod", "default", "api-pod-1", nil)
	g.AddNode("_cluster/node/worker-1", "node", "_cluster", "worker-1", nil)

	g.AddEdge("_cluster/ingress/api.example.com", "default/service/api-svc", "routes_to", 1.0)
	g.AddEdge("default/service/api-svc", "default/pod/api-pod-1", "selects", 1.0)
	g.AddEdge("default/pod/api-pod-1", "_cluster/node/worker-1", "runs_on", 1.0)

	g.RebuildIndex()
	return g
}

func TestTrace_Downstream(t *testing.T) {
	c := NewCorrelator()
	c.Update("test", makeTestGraph())

	result := c.Trace("test", "_cluster/ingress/api.example.com", "downstream", 10)
	if result == nil {
		t.Fatal("result should not be nil")
	}

	// 应该包含所有 4 个节点
	if len(result.Nodes) != 4 {
		t.Fatalf("downstream from ingress should reach 4 nodes, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 3 {
		t.Fatalf("downstream should have 3 edges, got %d", len(result.Edges))
	}
	if result.Depth != 3 {
		t.Fatalf("downstream depth should be 3, got %d", result.Depth)
	}
}

func TestTrace_Upstream(t *testing.T) {
	c := NewCorrelator()
	c.Update("test", makeTestGraph())

	result := c.Trace("test", "_cluster/node/worker-1", "upstream", 10)
	if result == nil {
		t.Fatal("result should not be nil")
	}

	// 从 node 往上追踪应该找到所有节点
	if len(result.Nodes) != 4 {
		t.Fatalf("upstream from node should reach 4 nodes, got %d", len(result.Nodes))
	}
}

func TestTrace_MaxDepth(t *testing.T) {
	c := NewCorrelator()
	c.Update("test", makeTestGraph())

	// 限制深度为 1，从 ingress 出发应该只到 service
	result := c.Trace("test", "_cluster/ingress/api.example.com", "downstream", 1)
	if len(result.Nodes) != 2 {
		t.Fatalf("depth=1 from ingress should reach 2 nodes (ingress+service), got %d", len(result.Nodes))
	}
}

func TestTrace_NonExistentCluster(t *testing.T) {
	c := NewCorrelator()
	result := c.Trace("nonexistent", "key", "downstream", 10)
	if result == nil {
		t.Fatal("should return empty TraceResult, not nil")
	}
	if len(result.Nodes) != 0 {
		t.Fatalf("should have 0 nodes, got %d", len(result.Nodes))
	}
}

func TestCorrelator_ListClusters(t *testing.T) {
	c := NewCorrelator()
	c.Update("cluster-a", makeTestGraph())
	c.Update("cluster-b", makeTestGraph())

	ids := c.ListClusters()
	if len(ids) != 2 {
		t.Fatalf("should list 2 clusters, got %d", len(ids))
	}
}
