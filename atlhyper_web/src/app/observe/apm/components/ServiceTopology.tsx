"use client";

import { useRef, useEffect, useEffectEvent, useMemo, useState } from "react";
import type { Topology } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { buildGraphData, addState, removeState, applySelection } from "./topology-utils";
import { buildTopologyTooltip, type TopologyTooltipItem } from "./topology-tooltip";
import { TopologyHeader, TopologyLegend } from "./TopologyLegend";

interface ServiceTopologyProps {
  t: ApmTranslations;
  topology: Topology;
  onSelectService: (name: string) => void;
}

/** G6 图实例里用到的方法（G6 是按需动态加载的，这里只声明用到的部分） */
interface TopologyGraphInstance {
  destroy: () => void;
  updateData: (data: ReturnType<typeof buildGraphData>) => void;
  draw: () => void;
  resize: (width: number, height: number) => void;
  fitView: () => void;
}

/** G6 指针事件里用到的部分 */
interface GraphPointerEvent {
  target?: { id?: string };
}

export function ServiceTopology({ t, topology, onSelectService }: ServiceTopologyProps) {
  // 没有节点时不挂图（组件不隐藏，显示空状态）。图拆成子组件，hooks 才不会落在提前返回之后
  if (!topology.nodes || topology.nodes.length === 0) {
    return (
      <div className="border border-[var(--border-color)] rounded-xl bg-card overflow-hidden">
        <div className="px-4 py-2.5 border-b border-[var(--border-color)]">
          <h3 className="text-sm font-medium text-default">{t.serviceTopology}</h3>
        </div>
        <div className="py-12 text-center text-sm text-muted">{t.noData}</div>
      </div>
    );
  }
  return <ServiceTopologyGraph t={t} topology={topology} onSelectService={onSelectService} />;
}

function ServiceTopologyGraph({ t, topology, onSelectService }: ServiceTopologyProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<TopologyGraphInstance | null>(null);
  const readyRef = useRef(false);

  const [expanded, setExpanded] = useState(false);

  const stats = useMemo(() => {
    const nodeCount = topology.nodes.length;
    const errorNodes = topology.nodes.filter((n) => n.successRate < 0.99).length;
    const totalCalls = topology.edges.reduce((s, e) => s + e.callCount, 0);
    const avgP99 = nodeCount > 0
      ? topology.nodes.reduce((s, n) => s + n.p99Ms, 0) / nodeCount
      : 0;
    return { nodeCount, errorNodes, totalCalls, avgP99 };
  }, [topology]);

  // 图的「形状」：节点与边的集合。只有形状变了才重建图；数值变了只更新数据（见下面第二个 effect）
  const topoKey = useMemo(() => {
    const nk = topology.nodes.map((n) => n.id).sort().join(",");
    const ek = topology.edges.map((e) => `${e.source}>${e.target}`).sort().join(",");
    return nk + "||" + ek;
  }, [topology]);

  // 下面三个函数在「重建图」的 effect 里用，但要读的是最新的 topology / t / onSelectService。
  // 它们不该成为 effect 的依赖 —— 否则每一轮轮询数值一变，整张图就销毁重画、布局重排
  const buildLatestGraphData = useEffectEvent(() => buildGraphData(topology));
  const renderTooltip = useEffectEvent((items: TopologyTooltipItem[] | undefined) => buildTopologyTooltip(items, t));
  const selectServiceNode = useEffectEvent((nodeId: string) => {
    const node = topology.nodes.find((n) => n.id === nodeId);
    if (node?.type === "service") onSelectService(node.name);
  });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    readyRef.current = false;
    container.innerHTML = "";

    let destroyed = false;

    async function createGraph(graphContainer: HTMLDivElement) {
      const { Graph } = await import("@antv/g6");
      if (destroyed) return;

      const instance = new Graph({
        container: graphContainer,
        autoFit: "view",
        padding: [24, 40, 24, 40],
        data: buildLatestGraphData(),
        node: {
          style: { cursor: "pointer" as const },
          state: {
            active: { lineWidth: 3, shadowBlur: 12, shadowColor: "rgba(96,165,250,0.4)" },
            selected: { lineWidth: 3.5, shadowBlur: 20, shadowColor: "#3b82f6" },
          },
        },
        edge: {
          style: { type: "cubic-horizontal" },
          state: {
            active: { stroke: "#60a5fa", lineWidth: 2, strokeOpacity: 0.8 },
            selected: { stroke: "#3b82f6", lineWidth: 2.5, strokeOpacity: 1 },
          },
        },
        layout: {
          type: "dagre",
          rankdir: "LR",
          nodesep: 50,
          ranksep: 120,
        },
        behaviors: ["drag-canvas", "zoom-canvas", "drag-element"],
        plugins: [
          {
            type: "tooltip",
            getContent: (_event: unknown, items: TopologyTooltipItem[]) => renderTooltip(items),
          },
        ],
        animation: true,
      });

      let hoveredIds: string[] = [];

      instance.on("node:pointerover", (evt) => {
        const nodeId = (evt as GraphPointerEvent).target?.id;
        if (!nodeId) return;
        hoveredIds = [];
        const activate = (id: string) => { addState(instance, id, "active"); hoveredIds.push(id); };
        activate(nodeId);
        for (const e of instance.getEdgeData()) {
          const ed = e as { id: string; source: string; target: string };
          if (ed.source === nodeId || ed.target === nodeId) {
            activate(ed.id);
            activate(ed.source === nodeId ? ed.target : ed.source);
          }
        }
      });

      instance.on("node:pointerout", () => {
        for (const id of hoveredIds) removeState(instance, id, "active");
        hoveredIds = [];
      });

      instance.on("node:click", (evt) => {
        const nodeId = (evt as GraphPointerEvent).target?.id;
        if (!nodeId) return;
        try { applySelection(instance, nodeId); } catch { /* ignore */ }
        selectServiceNode(nodeId);
      });

      instance.on("canvas:click", () => {
        try { applySelection(instance, null); } catch { /* ignore */ }
      });

      await instance.render();

      if (destroyed) {
        try { instance.destroy(); } catch { /* ignore */ }
        return;
      }

      graphRef.current = instance;
      readyRef.current = true;
    }

    void createGraph(container);

    return () => {
      destroyed = true;
      readyRef.current = false;
      if (graphRef.current) {
        try { graphRef.current.destroy(); } catch { /* ignore */ }
        graphRef.current = null;
      }
    };
  }, [topoKey]);

  // 形状没变、只是数值变了（轮询刷新）：原地更新数据，不重建图
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;
    try {
      g.updateData(buildGraphData(topology));
      g.draw();
    } catch { /* ignore */ }
  }, [topology]);

  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;
    const timer = setTimeout(() => {
      try {
        const container = containerRef.current;
        if (container) {
          g.resize(container.offsetWidth, container.offsetHeight);
          g.fitView();
        }
      } catch { /* ignore */ }
    }, 50);
    return () => clearTimeout(timer);
  }, [expanded]);

  return (
    <div className="border border-[var(--border-color)] rounded-xl bg-card overflow-hidden">
      <TopologyHeader
        t={t}
        stats={stats}
        expanded={expanded}
        onToggleExpand={() => setExpanded((v) => !v)}
        formatDurationMs={formatDurationMs}
      />

      {/* Graph */}
      <div
        ref={containerRef}
        className="w-full transition-[height] duration-300 ease-in-out"
        style={{ height: expanded ? 480 : 320 }}
      />

      <TopologyLegend t={t} />
    </div>
  );
}
