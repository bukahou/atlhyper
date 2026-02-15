// atlhyper_master_v2/aiops/correlator/builder_test.go
package correlator

import (
	"testing"

	"AtlHyper/model_v2"
)

func TestBuildFromSnapshot_Empty(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{}
	graph := BuildFromSnapshot("test-cluster", snap)

	if graph == nil {
		t.Fatal("graph should not be nil")
	}
	if graph.ClusterID != "test-cluster" {
		t.Fatalf("want clusterID=test-cluster, got %s", graph.ClusterID)
	}
	if len(graph.Nodes) != 0 {
		t.Fatalf("empty snapshot should produce 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Fatalf("empty snapshot should produce 0 edges, got %d", len(graph.Edges))
	}
}

func TestBuildFromSnapshot_PodToNode(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Pods: []model_v2.Pod{
			{
				Summary: model_v2.PodSummary{
					Name:      "api-server-abc",
					Namespace: "default",
					NodeName:  "worker-1",
				},
				Labels: map[string]string{"app": "api"},
			},
		},
		Nodes: []model_v2.Node{
			{
				Summary: model_v2.NodeSummary{Name: "worker-1"},
			},
		},
	}

	graph := BuildFromSnapshot("test", snap)

	// 应有 pod + node 节点
	if len(graph.Nodes) < 2 {
		t.Fatalf("should have at least 2 nodes (pod+node), got %d", len(graph.Nodes))
	}

	// 检查 runs_on 边
	found := false
	for _, edge := range graph.Edges {
		if edge.Type == "runs_on" && edge.To == "_cluster/node/worker-1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("should have runs_on edge from pod to node")
	}
}

func TestBuildFromSnapshot_ServiceSelectsPod(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Services: []model_v2.Service{
			{
				Summary:  model_v2.ServiceSummary{Name: "api-svc", Namespace: "default"},
				Selector: map[string]string{"app": "api"},
			},
		},
		Pods: []model_v2.Pod{
			{
				Summary: model_v2.PodSummary{Name: "api-pod-1", Namespace: "default"},
				Labels:  map[string]string{"app": "api", "version": "v1"},
			},
			{
				Summary: model_v2.PodSummary{Name: "other-pod", Namespace: "default"},
				Labels:  map[string]string{"app": "other"},
			},
		},
	}

	graph := BuildFromSnapshot("test", snap)

	// 应有 selects 边从 service 到匹配的 pod
	selectsEdges := 0
	for _, edge := range graph.Edges {
		if edge.Type == "selects" {
			selectsEdges++
			if edge.From != "default/service/api-svc" {
				t.Fatalf("selects edge from should be service, got %s", edge.From)
			}
			if edge.To != "default/pod/api-pod-1" {
				t.Fatalf("selects edge to should be matching pod, got %s", edge.To)
			}
		}
	}
	if selectsEdges != 1 {
		t.Fatalf("should have exactly 1 selects edge, got %d", selectsEdges)
	}
}

func TestBuildFromSnapshot_IngressRoutesToService(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		Ingresses: []model_v2.Ingress{
			{
				Summary: model_v2.IngressSummary{Name: "my-ingress", Namespace: "default"},
				Spec: model_v2.IngressSpec{
					Rules: []model_v2.IngressRule{
						{
							Host: "api.example.com",
							Paths: []model_v2.IngressPath{
								{
									Path: "/",
									Backend: &model_v2.IngressBackend{
										Service: &model_v2.IngressServiceBackend{
											Name:       "api-svc",
											PortNumber: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Services: []model_v2.Service{
			{
				Summary: model_v2.ServiceSummary{Name: "api-svc", Namespace: "default"},
			},
		},
	}

	graph := BuildFromSnapshot("test", snap)

	// 检查 routes_to 边: ingress 节点 key 使用 ingress 资源名而非 host
	found := false
	for _, edge := range graph.Edges {
		if edge.Type == "routes_to" {
			found = true
			if edge.From != "default/ingress/my-ingress" {
				t.Fatalf("routes_to edge from should be ingress resource key, got %s", edge.From)
			}
			if edge.To != "default/service/api-svc" {
				t.Fatalf("routes_to edge to should be service, got %s", edge.To)
			}
		}
	}
	if !found {
		t.Fatal("should have routes_to edge from ingress to service")
	}
}

func TestBuildFromSnapshot_SLOCalls(t *testing.T) {
	snap := &model_v2.ClusterSnapshot{
		SLOData: &model_v2.SLOSnapshot{
			Edges: []model_v2.ServiceEdge{
				{
					SrcNamespace: "default",
					SrcName:      "frontend",
					DstNamespace: "default",
					DstName:      "backend",
				},
			},
		},
	}

	graph := BuildFromSnapshot("test", snap)

	// 检查 calls 边
	found := false
	for _, edge := range graph.Edges {
		if edge.Type == "calls" {
			found = true
			if edge.From != "default/service/frontend" {
				t.Fatalf("calls edge from should be src service, got %s", edge.From)
			}
			if edge.To != "default/service/backend" {
				t.Fatalf("calls edge to should be dst service, got %s", edge.To)
			}
		}
	}
	if !found {
		t.Fatal("should have calls edge from SLO data")
	}
}
