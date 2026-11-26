// atlhyper_master_v2/aiops/correlator/builder.go
// 从 ClusterSnapshot 构建依赖图 DAG
package correlator

import (
	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/slo"
)

// BuildFromSnapshot 从快照构建完整依赖图
// otel 参数独立传入（而非从 snap.OTel 读取），便于 AIOps 引擎控制数据源
func BuildFromSnapshot(clusterID string, snap *cluster.ClusterSnapshot, otel *cluster.OTelSnapshot) *aiops.DependencyGraph {
	g := aiops.NewDependencyGraph(clusterID)

	// 1. Pod → Node (runs_on)
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		podKey := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		g.AddNode(podKey, "pod", pod.Summary.Namespace, pod.Summary.Name, nil)

		if pod.Summary.NodeName != "" {
			nodeKey := aiops.EntityKey(aiops.ClusterScope, "node", pod.Summary.NodeName)
			g.AddNode(nodeKey, "node", aiops.ClusterScope, pod.Summary.NodeName, nil)
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

	// 4. 入口（SLO）→ Service (routes_to)：入口指标按服务标识上报，落到该服务的入口实体上，
	// 与服务连起来，入口的异常才能沿依赖图传到后端
	if otel != nil {
		for i := range otel.SLOIngress {
			serviceKey := otel.SLOIngress[i].ServiceKey
			if serviceKey == "" {
				continue
			}
			ns, svcName := slo.SplitServiceKey(serviceKey)
			ingKey := aiops.IngressEntityKey(serviceKey)
			svcKey := aiops.EntityKey(ns, "service", svcName)
			g.AddNode(ingKey, "ingress", ns, svcName, nil)
			g.AddNode(svcKey, "service", ns, svcName, nil)
			g.AddEdge(ingKey, svcKey, "routes_to", 1.0)
		}
	}

	// 5. Service → Service (calls，来自 APM 拓扑，去重)
	// 边的两端是拓扑节点 Id（Agent 写作「命名空间/服务名」），先经节点表还原命名空间与服务名再拼实体键，
	// 与 APM / 日志的服务指标同一拼法；非服务节点（如数据库）没有对应的实体类型，跳过
	if otel != nil && otel.APMTopology != nil {
		nodesByID := make(map[string]apm.TopologyNode, len(otel.APMTopology.Nodes))
		for _, n := range otel.APMTopology.Nodes {
			nodesByID[n.Id] = n
		}
		edgeSet := make(map[string]bool) // "srcKey->dstKey" 去重
		for _, edge := range otel.APMTopology.Edges {
			src, srcOK := apmServiceNode(nodesByID, edge.Source)
			dst, dstOK := apmServiceNode(nodesByID, edge.Target)
			if !srcOK || !dstOK {
				continue
			}
			srcKey := aiops.EntityKey(src.Namespace, "service", src.Name)
			dstKey := aiops.EntityKey(dst.Namespace, "service", dst.Name)
			dedupKey := srcKey + "->" + dstKey
			if edgeSet[dedupKey] {
				continue
			}
			edgeSet[dedupKey] = true
			g.AddNode(srcKey, "service", entityNamespace(src.Namespace), src.Name, nil)
			g.AddNode(dstKey, "service", entityNamespace(dst.Namespace), dst.Name, nil)
			g.AddEdge(srcKey, dstKey, "calls", 1.0)
		}
	}

	// 6. 虚拟实体：logs/global（OTel 日志聚合节点）
	if otel != nil && otel.LogsSummary != nil {
		logsKey := aiops.EntityKey(aiops.ClusterScope, "logs", "global")
		g.AddNode(logsKey, "logs", aiops.ClusterScope, "Log Summary", nil)
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

// apmServiceNode 按 Id 取 APM 拓扑里的服务节点。不认识的 Id、非服务节点（如数据库）返回 false；
// 旧版 Agent 不填 Type，按服务处理
func apmServiceNode(nodesByID map[string]apm.TopologyNode, id string) (apm.TopologyNode, bool) {
	node, ok := nodesByID[id]
	if !ok || (node.Type != "" && node.Type != "service") {
		return apm.TopologyNode{}, false
	}
	return node, true
}

// entityNamespace 实体所在的命名空间：空值按集群级（与 aiops.EntityKey 一致）
func entityNamespace(namespace string) string {
	if namespace == "" {
		return aiops.ClusterScope
	}
	return namespace
}
