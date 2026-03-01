import { useMemo } from "react";
import type { DependencyGraph, EntityRisk, GraphNode } from "@/api/aiops";

type ViewMode = "service" | "anomaly" | "full";

/**
 * 按视图模式和 namespace 筛选图数据。
 * - service: 只保留 service/ingress 节点和 calls/routes_to 边
 * - anomaly: 只保留异常实体及其一跳邻居
 * - full: 可选 namespace 筛选（保留关联的 _cluster 节点）
 */
export function useFilteredGraph(
  graph: DependencyGraph | null,
  entityRisks: EntityRisk[],
  viewMode: ViewMode,
  nsFilter: Set<string>,
): DependencyGraph | null {
  return useMemo(() => {
    if (!graph) return null;

    if (viewMode === "service") {
      const keepTypes = new Set(["service", "ingress"]);
      const keepNodes: Record<string, GraphNode> = {};
      for (const [key, node] of Object.entries(graph.nodes)) {
        if (keepTypes.has(node.type)) keepNodes[key] = node;
      }
      const keepKeys = new Set(Object.keys(keepNodes));
      const keepEdgeTypes = new Set(["calls", "routes_to"]);
      const edges = graph.edges.filter(
        (e) => keepEdgeTypes.has(e.type) && keepKeys.has(e.from) && keepKeys.has(e.to),
      );
      return { ...graph, nodes: keepNodes, edges };
    }

    if (viewMode === "anomaly") {
      const anomalyKeys = new Set<string>();
      for (const r of entityRisks) {
        if (r.rFinal > 0) anomalyKeys.add(r.entityKey);
      }
      // 一跳邻居
      const neighborKeys = new Set(anomalyKeys);
      for (const e of graph.edges) {
        if (anomalyKeys.has(e.from)) neighborKeys.add(e.to);
        if (anomalyKeys.has(e.to)) neighborKeys.add(e.from);
      }
      const keepNodes: Record<string, GraphNode> = {};
      for (const [key, node] of Object.entries(graph.nodes)) {
        if (neighborKeys.has(key)) keepNodes[key] = node;
      }
      const keepKeys = new Set(Object.keys(keepNodes));
      const edges = graph.edges.filter(
        (e) => keepKeys.has(e.from) && keepKeys.has(e.to),
      );
      return { ...graph, nodes: keepNodes, edges };
    }

    // full -- 可选 namespace 筛选
    if (nsFilter.size > 0) {
      // 按选中 namespace 过滤节点
      const keepNodes: Record<string, GraphNode> = {};
      for (const [key, node] of Object.entries(graph.nodes)) {
        if (nsFilter.has(node.namespace)) keepNodes[key] = node;
      }
      // 保留与选中节点相连的 _cluster 节点（物理 Node）
      const keepKeys = new Set(Object.keys(keepNodes));
      for (const e of graph.edges) {
        if (keepKeys.has(e.from) && !keepKeys.has(e.to)) {
          const n = graph.nodes[e.to];
          if (n?.namespace === "_cluster") { keepNodes[e.to] = n; keepKeys.add(e.to); }
        }
        if (keepKeys.has(e.to) && !keepKeys.has(e.from)) {
          const n = graph.nodes[e.from];
          if (n?.namespace === "_cluster") { keepNodes[e.from] = n; keepKeys.add(e.from); }
        }
      }
      const edges = graph.edges.filter((e) => keepKeys.has(e.from) && keepKeys.has(e.to));
      return { ...graph, nodes: keepNodes, edges };
    }

    return graph;
  }, [graph, entityRisks, viewMode, nsFilter]);
}
