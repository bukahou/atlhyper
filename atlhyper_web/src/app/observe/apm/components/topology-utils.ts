import type { HealthStatus, Topology } from "@/types/model/apm";

/** Health-based ring color (Kibana-style softer palette) */
export function ringColor(status: HealthStatus, successRate: number): string {
  if (status === "critical" || successRate < 0.95) return "#ef4444";
  if (status === "warning" || successRate < 0.99) return "#f59e0b";
  return "#60a5fa"; // healthy = blue
}

/** Node icon by type */
export function nodeIcon(type: string): string {
  if (type === "database") return "\u26C1";
  if (type === "external") return "\u2295";
  return "\u2B21";
}

/** Build G6 graph data from topology */
export function buildGraphData(topology: Topology) {
  const isDark = document.documentElement.classList.contains("dark");
  const labelColor = isDark ? "#d1d5db" : "#374151";
  const labelBg = isDark ? "rgba(0,0,0,0.6)" : "rgba(255,255,255,0.85)";

  const nodes = topology.nodes.map((n) => {
    const color = ringColor(n.status, n.successRate);
    const isDb = n.type === "database";
    const size = isDb ? 36 : 42;

    return {
      id: n.id,
      data: { rps: n.rps, successRate: n.successRate, p99Ms: n.p99Ms, namespace: n.namespace, type: n.type },
      style: {
        type: (isDb ? "diamond" : "circle") as "circle" | "diamond",
        size,
        fill: isDark ? `${color}18` : `${color}12`,
        stroke: color,
        lineWidth: 2.5,
        shadowBlur: 0,
        labelText: n.name.length > 22 ? n.name.slice(0, 20) + ".." : n.name,
        labelFontSize: 11,
        labelFontWeight: 500,
        labelFill: labelColor,
        labelPlacement: "bottom" as const,
        labelOffsetY: 6,
        labelBackground: true,
        labelBackgroundFill: labelBg,
        labelBackgroundRadius: 4,
        labelBackgroundPadding: [2, 6, 2, 6],
        iconText: nodeIcon(n.type),
        iconFontSize: isDb ? 16 : 15,
        iconFontWeight: 400,
        iconFill: color,
        cursor: "pointer" as const,
        badges: n.successRate < 0.99
          ? [{
              text: `${((1 - n.successRate) * 100).toFixed(1)}%`,
              placement: "right-top" as const,
              backgroundFill: n.successRate < 0.95 ? "#ef4444" : "#f59e0b",
              fill: "#fff",
              fontSize: 8,
            }]
          : [],
        halo: true,
        haloStroke: color,
        haloStrokeOpacity: isDark ? 0.15 : 0.1,
        haloLineWidth: 8,
      },
    };
  });

  // Filter edges: both source and target must exist
  const nodeIds = new Set(topology.nodes.map((n) => n.id));
  const validEdges = topology.edges.filter((e) => nodeIds.has(e.source) && nodeIds.has(e.target));

  const edgeColor = isDark ? "#4b5563" : "#9ca3af";
  const edgeErrorColor = "#f87171";

  const edges = validEdges.map((e) => {
    const hasError = e.errorRate > 0;
    return {
      id: `${e.source}>${e.target}`,
      source: e.source,
      target: e.target,
      data: { callCount: e.callCount, avgMs: e.avgMs, errorRate: e.errorRate },
      style: {
        type: "cubic-horizontal" as const,
        stroke: hasError ? edgeErrorColor : edgeColor,
        lineWidth: 1.5,
        strokeOpacity: hasError ? 0.6 : 0.4,
        lineDash: hasError ? [6, 3] : (undefined as number[] | undefined),
        endArrow: true,
        endArrowSize: 5,
        endArrowFill: hasError ? edgeErrorColor : edgeColor,
        endArrowFillOpacity: hasError ? 0.6 : 0.4,
      },
    };
  });

  return { nodes, edges };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function addState(g: any, id: string, state: string) {
  try {
    const cur: string[] = g.getElementState(id);
    if (!cur.includes(state)) g.setElementState(id, [...cur, state]);
  } catch { /* ignore */ }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function removeState(g: any, id: string, state: string) {
  try {
    const cur: string[] = g.getElementState(id);
    if (cur.includes(state)) g.setElementState(id, cur.filter((s: string) => s !== state));
  } catch { /* ignore */ }
}

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
    const ed = e as { id: string; source: string; target: string };
    if (ed.source === nodeId || ed.target === nodeId) {
      addState(g, ed.id, "selected");
    }
  }
}
