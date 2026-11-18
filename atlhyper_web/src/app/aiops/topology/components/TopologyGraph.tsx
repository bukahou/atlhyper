"use client";

import { useRef, useEffect, useEffectEvent, useMemo } from "react";
// 只导入类型：G6 本体在下面按需动态加载，不进首屏包
import type { Graph } from "@antv/g6";
import type { DependencyGraph, EntityRisk } from "@/api/aiops";
import { addState, removeState, applySelection, buildG6Data } from "./topology-graph-utils";
import { formatRiskScore } from "@/lib/risk";
import { safeHtml } from "@/lib/safe-html";

/** G6 指针事件里用到的部分 */
interface GraphPointerEvent {
  target?: { id?: string };
  targetId?: string;
}

interface TopologyGraphProps {
  graph: DependencyGraph;
  entityRisks: Record<string, EntityRisk>;
  selectedNode: string | null;
  onNodeSelect: (key: string | null) => void;
}

export function TopologyGraph({ graph, entityRisks, selectedNode, onNodeSelect }: TopologyGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<Graph | null>(null);
  const readyRef = useRef(false);

  // 拓扑结构指纹：仅节点集合 + 边集合变化时才重建图
  const topologyKey = useMemo(() => {
    const nk = Object.keys(graph.nodes).sort().join(",");
    const ek = graph.edges.map((e) => `${e.from}>${e.to}:${e.type}`).sort().join(",");
    return nk + "||" + ek;
  }, [graph]);

  // 构造 G6 数据（使用稳定的 edge ID）
  const graphData = useMemo(() => buildG6Data(graph, entityRisks), [graph, entityRisks]);

  // 下面三个函数在 effect 里用，但要读的是最新的图数据 / 选中节点 / 回调。
  // 它们不该成为「重建图」那个 effect 的依赖 —— 否则每一轮轮询风险分数一变，整张图就销毁重画、布局重排
  const readLatestGraphData = useEffectEvent(() => graphData);
  const readSelectedNode = useEffectEvent(() => selectedNode);
  const reportNodeSelected = useEffectEvent((nodeKey: string | null) => onNodeSelect(nodeKey));

  // Effect 1: 拓扑结构变化时 → 销毁重建（含力导向布局）
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    readyRef.current = false;

    if (graphRef.current) {
      try { graphRef.current.destroy(); } catch { /* ignore */ }
      graphRef.current = null;
    }
    // G6 图表库要求清空容器 DOM 后重新创建，此处数据来源为内部计算，无 XSS 风险
    container.innerHTML = "";

    let destroyed = false;

    async function createGraph() {
      const { Graph } = await import("@antv/g6");

      if (destroyed || !container) return;

      const instance = new Graph({
        container,
        autoFit: "view",
        data: readLatestGraphData(),
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
              if (!d) return safeHtml`${item.id}`.toString();
              const riskLabel = d.rFinal ? ` | R: ${formatRiskScore(d.rFinal)}` : "";
              return safeHtml`<div style="padding:6px 10px;font-size:12px;border-radius:6px;background:rgba(0,0,0,0.8);color:#fff">
                <b>${item.id.split("/").pop()}</b><br/>
                <span style="opacity:0.7">${d.type ?? ""}${d.namespace ? ` · ${d.namespace}` : ""}${riskLabel}</span>
              </div>`.toString();
            },
          },
          { type: "minimap", size: [120, 80] },
        ],
        animation: true,
      });

      // ── 手动 hover 高亮（degree-1 邻居） ──
      let hoveredIds: string[] = [];

      instance.on("node:pointerover", (evt) => {
        const nodeId = (evt as GraphPointerEvent).target?.id;
        if (!nodeId) return;
        try {
          hoveredIds = [];
          const activate = (id: string) => {
            addState(instance, id, "active");
            hoveredIds.push(id);
          };
          activate(nodeId);
          for (const edge of instance.getEdgeData()) {
            if (edge.id !== undefined && (edge.source === nodeId || edge.target === nodeId)) {
              activate(edge.id);
              activate(edge.source === nodeId ? edge.target : edge.source);
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
      instance.on("node:click", (evt) => {
        const pointerEvent = evt as GraphPointerEvent;
        const nodeId = pointerEvent.target?.id ?? pointerEvent.targetId;
        if (!nodeId) return;
        try { applySelection(instance, nodeId); } catch { /* ignore */ }
        reportNodeSelected(nodeId);
      });

      // 点击画布空白 → 取消选中
      instance.on("canvas:click", () => {
        try { applySelection(instance, null); } catch { /* ignore */ }
        reportNodeSelected(null);
      });

      await instance.render();

      if (destroyed) {
        try { instance.destroy(); } catch { /* DOM already removed */ }
        return;
      }

      graphRef.current = instance;
      readyRef.current = true;

      // 如果创建时已有选中节点，立即应用选中状态
      const selectedAtCreation = readSelectedNode();
      if (selectedAtCreation) {
        try { applySelection(instance, selectedAtCreation); } catch { /* ignore */ }
      }
    }

    void createGraph();

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
    // 只依赖拓扑结构的指纹：节点集合或边集合变了才重建；数据与回调经上面的 Effect Event 读最新值
  }, [topologyKey]);

  // Effect 2: 风险数据变化时 → 原地更新样式（不重建布局，节点位置不变）
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;

    try {
      g.updateData(graphData);
      void g.draw();
      // updateData 可能重置状态，重新应用选中高亮
      const selectedNow = readSelectedNode();
      if (selectedNow) {
        applySelection(g, selectedNow);
      }
    } catch {
      // G6 API 不支持或图未就绪，等待下次结构变化时重建
    }
  }, [graphData]);

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
