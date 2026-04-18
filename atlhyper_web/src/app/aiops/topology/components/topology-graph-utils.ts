import type { DependencyGraph, EntityRisk } from "@/api/aiops";
import { riskColor, formatRiskScore, RISK_THRESHOLDS } from "@/lib/risk";

// 重新导出给同目录其他组件复用（TopologyGraph.tsx 等）
export { riskColor };

// 实体类型 → G6 节点形状
export const TYPE_SHAPE: Record<string, string> = {
  service: "circle",
  pod: "rect",
  node: "hexagon",
  ingress: "diamond",
};

// 边样式按类型区分
export function edgeStyle(type: string, isAnomaly: boolean) {
  if (isAnomaly) {
    return { stroke: "#ef4444", lineWidth: 2, strokeOpacity: 0.8, lineDash: undefined as number[] | undefined };
  }
  switch (type) {
    case "calls":     return { stroke: "#666", lineWidth: 1,   strokeOpacity: 0.5, lineDash: undefined as number[] | undefined };
    case "routes_to": return { stroke: "#888", lineWidth: 0.8, strokeOpacity: 0.4, lineDash: [6, 3] };
    case "selects":   return { stroke: "#888", lineWidth: 0.8, strokeOpacity: 0.3, lineDash: [3, 3] };
    case "runs_on":   return { stroke: "#888", lineWidth: 0.6, strokeOpacity: 0.2, lineDash: [2, 4] };
    default:          return { stroke: "#666", lineWidth: 0.8, strokeOpacity: 0.3, lineDash: undefined as number[] | undefined };
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function addState(g: any, id: string, state: string) {
  const cur: string[] = g.getElementState(id);
  if (!cur.includes(state)) {
    g.setElementState(id, [...cur, state]);
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function removeState(g: any, id: string, state: string) {
  const cur: string[] = g.getElementState(id);
  if (cur.includes(state)) {
    g.setElementState(id, cur.filter((s: string) => s !== state));
  }
}

/** 清除所有元素的 "selected" 状态，再为选中节点 + 关联边添加 "selected" */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function applySelection(g: any, nodeId: string | null) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  for (const n of g.getNodeData()) removeState(g, (n as any).id, "selected");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  for (const e of g.getEdgeData()) removeState(g, (e as any).id, "selected");

  if (!nodeId) return;

  addState(g, nodeId, "selected");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  for (const e of g.getEdgeData()) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const ed = e as any;
    if (ed.source === nodeId || ed.target === nodeId) {
      addState(g, ed.id, "selected");
    }
  }
}

/** 构造 G6 数据（节点 + 边） */
export function buildG6Data(graph: DependencyGraph, entityRisks: Record<string, EntityRisk>) {
  const degreeMap: Record<string, number> = {};
  for (const e of graph.edges) {
    degreeMap[e.from] = (degreeMap[e.from] ?? 0) + 1;
    degreeMap[e.to] = (degreeMap[e.to] ?? 0) + 1;
  }

  const nodes = Object.values(graph.nodes).map((n) => {
    const risk = entityRisks[n.key];
    const rFinal = risk?.rFinal ?? 0;
    const color = riskColor(rFinal);
    const shape = TYPE_SHAPE[n.type] ?? "circle";

    const degree = degreeMap[n.key] ?? 0;
    const baseSize = n.type === "node" ? 40 : n.type === "service" ? 34 : 26;
    const size = baseSize + Math.min(degree * 2, 14);

    return {
      id: n.key,
      data: { type: n.type, namespace: n.namespace, rFinal },
      style: {
        type: shape,
        size,
        fill: color,
        fillOpacity: 0.3,
        stroke: color,
        lineWidth: 2,
        labelText: n.name.length > 18 ? n.name.slice(0, 16) + ".." : n.name,
        labelFontSize: 10,
        labelFill: color,
        labelPlacement: "bottom" as const,
        labelOffsetY: 4,
        iconText: n.type[0].toUpperCase(),
        iconFontSize: 12,
        iconFontWeight: 700,
        iconFill: color,
        badges: rFinal > 0
          ? [{ text: formatRiskScore(rFinal), placement: "right-top" as const, backgroundFill: color, fill: "#fff", fontSize: 8 }]
          : [],
      },
    };
  });

  const edgeSeen = new Set<string>();
  const edges: Array<{ id: string; source: string; target: string; style: Record<string, unknown> }> = [];
  for (const e of graph.edges) {
    const eid = `${e.from}>${e.to}:${e.type}`;
    if (edgeSeen.has(eid)) continue;
    edgeSeen.add(eid);

    const isAnomaly =
      (entityRisks[e.from]?.rFinal ?? 0) >= RISK_THRESHOLDS.warning ||
      (entityRisks[e.to]?.rFinal ?? 0) >= RISK_THRESHOLDS.warning;
    const style = edgeStyle(e.type, isAnomaly);

    edges.push({
      id: eid,
      source: e.from,
      target: e.to,
      style: {
        ...style,
        endArrow: true,
        endArrowSize: 6,
        endArrowFill: style.stroke,
      },
    });
  }

  return { nodes, edges };
}
