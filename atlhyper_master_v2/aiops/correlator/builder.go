// atlhyper_master_v2/aiops/correlator/builder.go
// 从 ClusterSnapshot 构建依赖图 DAG
package correlator

import (
	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v2"
)

// BuildFromSnapshot 从快照构建完整依赖图
func BuildFromSnapshot(clusterID string, snap *model_v2.ClusterSnapshot) *aiops.DependencyGraph {
	g := aiops.NewDependencyGraph(clusterID)

	// 1. Pod → Node (runs_on)
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		podKey := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		g.AddNode(podKey, "pod", pod.Summary.Namespace, pod.Summary.Name, nil)

		if pod.Summary.NodeName != "" {
			nodeKey := aiops.EntityKey("_cluster", "node", pod.Summary.NodeName)
			g.AddNode(nodeKey, "node", "_cluster", pod.Summary.NodeName, nil)
			g.AddEdge(podKey, nodeKey, "runs_on", 1.0)
		}
	}

	// 2. Service → Pod (selects)
	for i := range snap.Services {
		svc := &snap.Services[i]
		svcKey := aiops.EntityKey(svc.Summary.Namespace, "service", svc.Summary.Name)
		g.AddNode(svcKey, "service", svc.Summary.Namespace, svc.Summary.Name, nil)

		if len(svc.Selector) == 0 {
			continue
		}
		for j := range snap.Pods {
			pod := &snap.Pods[j]
			if svc.Summary.Namespace != pod.Summary.Namespace {
				continue
			}
			if matchSelector(svc.Selector, pod.Labels) {
				podKey := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
				g.AddEdge(svcKey, podKey, "selects", 1.0)
			}
		}
	}

	// 3. Ingress → Service (routes_to)
	for i := range snap.Ingresses {
		ing := &snap.Ingresses[i]
		ingKey := aiops.EntityKey(ing.Summary.Namespace, "ingress", ing.Summary.Name)
		g.AddNode(ingKey, "ingress", ing.Summary.Namespace, ing.Summary.Name, nil)

		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.Paths {
				if path.Backend != nil && path.Backend.Service != nil {
					svcKey := aiops.EntityKey(ing.Summary.Namespace, "service", path.Backend.Service.Name)
					g.AddEdge(ingKey, svcKey, "routes_to", 1.0)
				}
			}
		}
	}

	// 4. Service → Service (calls, 从 SLO Edge 数据)
	if snap.SLOData != nil {
		for _, edge := range snap.SLOData.Edges {
			srcKey := aiops.EntityKey(edge.SrcNamespace, "service", edge.SrcName)
			dstKey := aiops.EntityKey(edge.DstNamespace, "service", edge.DstName)
			// 确保节点存在
			g.AddNode(srcKey, "service", edge.SrcNamespace, edge.SrcName, nil)
			g.AddNode(dstKey, "service", edge.DstNamespace, edge.DstName, nil)
			g.AddEdge(srcKey, dstKey, "calls", 1.0)
		}
	}

	// 确保 Node 节点存在（可能没有 Pod 调度到的独立节点）
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		nodeKey := aiops.EntityKey("_cluster", "node", node.GetName())
		g.AddNode(nodeKey, "node", "_cluster", node.GetName(), nil)
	}

	g.RebuildIndex()
	return g
}

// matchSelector 检查 Pod Labels 是否匹配 Service Selector
func matchSelector(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}
