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
    const nodes = Object.values(graph.nodes).map((n) => {
      const risk = entityRisks[n.key];
      const rFinal = risk?.rFinal ?? 0;
      const color = riskColor(rFinal);
      const size = n.type === "node" ? 44 : n.type === "service" ? 36 : 28;
      const shape = TYPE_SHAPE[n.type] ?? "circle";

      return {
        id: n.key,
        data: { type: n.type, namespace: n.namespace, rFinal },
        style: {
          type: shape,
          size,
          fill: color,
          fillOpacity: 0.15,
          stroke: color,
          lineWidth: 1,
          labelText: n.name.length > 18 ? n.name.slice(0, 16) + ".." : n.name,
          labelFontSize: 10,
          labelFill: "#999",
          labelPlacement: "bottom" as const,
          labelOffsetY: 4,
          iconText: n.type[0].toUpperCase(),
          iconFontSize: 12,
          iconFontWeight: 700,
          iconFill: color,
          badges: rFinal > 50
            ? [{ text: rFinal.toFixed(0), placement: "right-top" as const, backgroundFill: color, fill: "#fff", fontSize: 8 }]
            : [],
        },
      };
    });

    const edges = graph.edges.map((e, i) => {
      const isAnomaly = e.type === "calls" && (entityRisks[e.from]?.rFinal ?? 0) > 50;
      return {
        id: `edge-${i}`,
        source: e.from,
        target: e.to,
        style: {
          stroke: isAnomaly ? "#ef4444" : "#666",
          lineWidth: isAnomaly ? 2 : 0.8,
          strokeOpacity: isAnomaly ? 0.8 : 0.3,
          endArrow: true,
          endArrowSize: 6,
          endArrowFill: isAnomaly ? "#ef4444" : "#666",
        },
      };
    });

    return { nodes, edges };
  }, [graph, entityRisks]);

  // 初始化 G6
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let g6Graph: any = null;
    let destroyed = false;

    async function createGraph() {
      const { Graph } = await import("@antv/g6");

      if (destroyed || !container) return null;

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
      return instance;
    }

    createGraph().then((instance) => {
      if (instance && !destroyed) {
        g6Graph = instance;
      }
    });

    return () => {
      destroyed = true;
      if (g6Graph) {
        g6Graph.destroy();
        g6Graph = null;
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
