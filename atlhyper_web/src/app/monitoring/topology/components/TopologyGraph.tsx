"use client";

import { useRef, useEffect, useCallback } from "react";
import type { DependencyGraph, EntityRisk } from "@/api/aiops";

// 节点颜色按风险等级
function riskColor(rFinal: number): string {
  if (rFinal >= 80) return "#ef4444";
  if (rFinal >= 50) return "#f97316";
  if (rFinal >= 30) return "#eab308";
  if (rFinal >= 10) return "#3b82f6";
  return "#22c55e";
}

// 实体类型 → G6 节点形状
const TYPE_SHAPE: Record<string, string> = {
  service: "circle",
  pod: "rect",
  node: "hexagon",
  ingress: "diamond",
};

// 边样式按类型区分
function edgeStyle(type: string, isAnomaly: boolean) {
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

interface TopologyGraphProps {
  graph: DependencyGraph;
  entityRisks: Record<string, EntityRisk>;
  selectedNode: string | null;
  onNodeSelect: (key: string) => void;
}

export function TopologyGraph({ graph, entityRisks, selectedNode, onNodeSelect }: TopologyGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const graphRef = useRef<any>(null);
  const onNodeSelectRef = useRef(onNodeSelect);
  onNodeSelectRef.current = onNodeSelect;

  // 构造 G6 数据
  const buildData = useCallback(() => {
    // 预计算每个节点的度数
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

      // 大小：基准按类型 + 度数动态加成
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
            ? [{ text: rFinal.toFixed(0), placement: "right-top" as const, backgroundFill: color, fill: "#fff", fontSize: 8 }]
            : [],
        },
      };
    });

    const edges = graph.edges.map((e, i) => {
      // 异常判断：源或目标任一端风险 > 50
      const isAnomaly =
        (entityRisks[e.from]?.rFinal ?? 0) > 50 ||
        (entityRisks[e.to]?.rFinal ?? 0) > 50;
      const style = edgeStyle(e.type, isAnomaly);

      return {
        id: `edge-${i}`,
        source: e.from,
        target: e.to,
        style: {
          ...style,
          endArrow: true,
          endArrowSize: 6,
          endArrowFill: style.stroke,
        },
      };
    });

    return { nodes, edges };
  }, [graph, entityRisks]);

  // 初始化 G6
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // 销毁旧实例 + 清空容器（防止残留 canvas）
    if (graphRef.current) {
      try { graphRef.current.destroy(); } catch { /* ignore */ }
      graphRef.current = null;
    }
    container.innerHTML = "";

    let destroyed = false;

    async function createGraph() {
      const { Graph } = await import("@antv/g6");

      if (destroyed || !container) return;

      const data = buildData();

      const instance = new Graph({
        container,
        autoFit: "view",
        data,
        node: {
          style: {
            labelBackground: true,
            labelBackgroundFill: "rgba(0,0,0,0.5)",
            labelBackgroundRadius: 3,
            labelBackgroundPadding: [1, 4, 1, 4],
            cursor: "pointer",
          },
          state: {
            selected: {
              lineWidth: 3,
              shadowBlur: 16,
              shadowColor: "#3b82f6",
            },
            active: {
              lineWidth: 2,
            },
          },
        },
        edge: {
          style: {
            type: "line",
          },
          state: {
            active: {
              stroke: "#3b82f6",
              lineWidth: 2,
            },
          },
        },
        layout: {
          type: "d3-force",
          link: { distance: 140 },
          charge: { strength: -400 },
          collide: { radius: 40 },
        },
        behaviors: [
          "drag-canvas",
          "zoom-canvas",
          "drag-element",
          {
            type: "hover-activate",
            degree: 1,
          },
        ],
        plugins: [
          {
            type: "tooltip",
            getContent: (_event: unknown, items: Array<{ id: string; data?: { type?: string; namespace?: string; rFinal?: number } }>) => {
              if (!items?.length) return "";
              const item = items[0];
              const d = item.data;
              if (!d) return item.id;
              const riskLabel = d.rFinal !== undefined ? ` | R: ${d.rFinal.toFixed(1)}` : "";
              return `<div style="padding:6px 10px;font-size:12px;border-radius:6px;background:rgba(0,0,0,0.8);color:#fff">
                <b>${item.id.split("/").pop()}</b><br/>
                <span style="opacity:0.7">${d.type ?? ""}${d.namespace ? ` · ${d.namespace}` : ""}${riskLabel}</span>
              </div>`;
            },
          },
          {
            type: "minimap",
            size: [120, 80],
          },
        ],
        animation: true,
      });

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:click", (evt: any) => {
        const nodeId = evt?.target?.id ?? evt?.targetId;
        if (nodeId) onNodeSelectRef.current(nodeId);
      });

      await instance.render();

      if (destroyed) {
        instance.destroy();
        return;
      }

      graphRef.current = instance;
    }

    createGraph();

    return () => {
      destroyed = true;
      if (graphRef.current) {
        try { graphRef.current.destroy(); } catch { /* ignore */ }
        graphRef.current = null;
      }
    };
  }, [buildData]);

  // 选中状态同步
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !selectedNode) return;
    try {
      g.setElementState(selectedNode, "selected");
    } catch {
      // 节点可能尚未渲染
    }
  }, [selectedNode]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full rounded-xl border border-[var(--border-color)] overflow-hidden bg-[var(--background)]"
    />
  );
}
