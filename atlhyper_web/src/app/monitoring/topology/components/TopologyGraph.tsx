"use client";

import { useRef, useEffect, useCallback, useMemo } from "react";
import type { DependencyGraph, EntityRisk } from "@/api/aiops";

// 节点颜色按风险等级 (rFinal: 0-1)
function riskColor(rFinal: number): string {
  if (rFinal >= 0.8) return "#ef4444";
  if (rFinal >= 0.5) return "#f97316";
  if (rFinal >= 0.3) return "#eab308";
  if (rFinal >= 0.1) return "#3b82f6";
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function addState(g: any, id: string, state: string) {
  const cur: string[] = g.getElementState(id);
  if (!cur.includes(state)) {
    g.setElementState(id, [...cur, state]);
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function removeState(g: any, id: string, state: string) {
  const cur: string[] = g.getElementState(id);
  if (cur.includes(state)) {
    g.setElementState(id, cur.filter((s: string) => s !== state));
  }
}

/** 清除所有元素的 "selected" 状态，再为选中节点 + 关联边添加 "selected" */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function applySelection(g: any, nodeId: string | null) {
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

interface TopologyGraphProps {
  graph: DependencyGraph;
  entityRisks: Record<string, EntityRisk>;
  selectedNode: string | null;
  onNodeSelect: (key: string | null) => void;
}

export function TopologyGraph({ graph, entityRisks, selectedNode, onNodeSelect }: TopologyGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const graphRef = useRef<any>(null);
  const readyRef = useRef(false);
  const onNodeSelectRef = useRef(onNodeSelect);
  onNodeSelectRef.current = onNodeSelect;
  const selectedNodeRef = useRef(selectedNode);
  selectedNodeRef.current = selectedNode;

  // 拓扑结构指纹：仅节点集合 + 边集合变化时才重建图
  const topologyKey = useMemo(() => {
    const nk = Object.keys(graph.nodes).sort().join(",");
    const ek = graph.edges.map((e) => `${e.from}>${e.to}:${e.type}`).sort().join(",");
    return nk + "||" + ek;
  }, [graph]);

  // 构造 G6 数据（使用稳定的 edge ID）
  const buildData = useCallback(() => {
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
            ? [{ text: (rFinal * 100).toFixed(0), placement: "right-top" as const, backgroundFill: color, fill: "#fff", fontSize: 8 }]
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
        (entityRisks[e.from]?.rFinal ?? 0) > 0.5 ||
        (entityRisks[e.to]?.rFinal ?? 0) > 0.5;
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
  }, [graph, entityRisks]);

  // Effect 1: 拓扑结构变化时 → 销毁重建（含力导向布局）
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    readyRef.current = false;

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
          style: { type: "line" },
          state: {
            active: {
              stroke: "#3b82f6",
              lineWidth: 2,
            },
            selected: {
              stroke: "#3b82f6",
              lineWidth: 2.5,
              strokeOpacity: 1,
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
          // hover-activate 替换为手动实现，避免 pointerout 恢复旧状态时清除 "selected"
        ],
        plugins: [
          {
            type: "tooltip",
            getContent: (_event: unknown, items: Array<{ id: string; data?: { type?: string; namespace?: string; rFinal?: number } }>) => {
              if (!items?.length) return "";
              const item = items[0];
              const d = item.data;
              if (!d) return item.id;
              const riskLabel = d.rFinal ? ` | R: ${(d.rFinal * 100).toFixed(0)}` : "";
              return `<div style="padding:6px 10px;font-size:12px;border-radius:6px;background:rgba(0,0,0,0.8);color:#fff">
                <b>${item.id.split("/").pop()}</b><br/>
                <span style="opacity:0.7">${d.type ?? ""}${d.namespace ? ` · ${d.namespace}` : ""}${riskLabel}</span>
              </div>`;
            },
          },
          { type: "minimap", size: [120, 80] },
        ],
        animation: true,
      });

      // ── 手动 hover 高亮（degree-1 邻居） ──
      let hoveredIds: string[] = [];

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:pointerover", (evt: any) => {
        const nodeId = evt?.target?.id;
        if (!nodeId) return;
        try {
          hoveredIds = [];
          const activate = (id: string) => {
            addState(instance, id, "active");
            hoveredIds.push(id);
          };
          activate(nodeId);
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          for (const e of instance.getEdgeData()) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const ed = e as any;
            if (ed.source === nodeId || ed.target === nodeId) {
              activate(ed.id);
              activate(ed.source === nodeId ? ed.target : ed.source);
            }
          }
        } catch { /* ignore */ }
      });

      instance.on("node:pointerout", () => {
        try {
          for (const id of hoveredIds) removeState(instance, id, "active");
          hoveredIds = [];
        } catch { /* ignore */ }
      });

      // ── 点击选中（持久高亮关联边） ──
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:click", (evt: any) => {
        const nodeId = evt?.target?.id ?? evt?.targetId;
        if (!nodeId) return;
        try { applySelection(instance, nodeId); } catch { /* ignore */ }
        onNodeSelectRef.current(nodeId);
      });

      // 点击画布空白 → 取消选中
      instance.on("canvas:click", () => {
        try { applySelection(instance, null); } catch { /* ignore */ }
        onNodeSelectRef.current(null);
      });

      await instance.render();

      if (destroyed) {
        try { instance.destroy(); } catch { /* DOM already removed */ }
        return;
      }

      graphRef.current = instance;
      readyRef.current = true;

      // 如果创建时已有选中节点，立即应用选中状态
      if (selectedNodeRef.current) {
        try { applySelection(instance, selectedNodeRef.current); } catch { /* ignore */ }
      }
    }

    createGraph();

    return () => {
      destroyed = true;
      readyRef.current = false;
      if (graphRef.current) {
        try {
          graphRef.current.destroy();
        } catch { /* ignore */ }
        graphRef.current = null;
      }
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [topologyKey]);

  // Effect 2: 风险数据变化时 → 原地更新样式（不重建布局，节点位置不变）
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;

    try {
      const data = buildData();
      g.updateData(data);
      g.draw();
      // updateData 可能重置状态，重新应用选中高亮
      if (selectedNodeRef.current) {
        applySelection(g, selectedNodeRef.current);
      }
    } catch {
      // G6 API 不支持或图未就绪，等待下次结构变化时重建
    }
  }, [buildData]);

  // Effect 3: selectedNode 外部变化时同步（如切换视图导致清除选中）
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;
    try {
      applySelection(g, selectedNode);
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
