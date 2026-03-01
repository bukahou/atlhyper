"use client";

import { useRef, useEffect, useCallback, useMemo, useState } from "react";
import type { Topology } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { ringColor, buildGraphData, addState, removeState, applySelection } from "./topology-utils";
import { TopologyHeader, TopologyLegend } from "./TopologyLegend";

interface ServiceTopologyProps {
  t: ApmTranslations;
  topology: Topology;
  onSelectService: (name: string) => void;
}

export function ServiceTopology({ t, topology, onSelectService }: ServiceTopologyProps) {
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

  const containerRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const graphRef = useRef<any>(null);
  const readyRef = useRef(false);
  const onSelectRef = useRef(onSelectService);
  onSelectRef.current = onSelectService;

  const [expanded, setExpanded] = useState(false);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  const stats = useMemo(() => {
    const nodeCount = topology.nodes.length;
    const errorNodes = topology.nodes.filter((n) => n.successRate < 0.99).length;
    const totalCalls = topology.edges.reduce((s, e) => s + e.callCount, 0);
    const avgP99 = nodeCount > 0
      ? topology.nodes.reduce((s, n) => s + n.p99Ms, 0) / nodeCount
      : 0;
    return { nodeCount, errorNodes, totalCalls, avgP99 };
  }, [topology]);

  const topoKey = useMemo(() => {
    const nk = topology.nodes.map((n) => n.id).sort().join(",");
    const ek = topology.edges.map((e) => `${e.source}>${e.target}`).sort().join(",");
    return nk + "||" + ek;
  }, [topology]);

  const buildData = useCallback(() => buildGraphData(topology), [topology]);

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
        padding: [24, 40, 24, 40],
        data,
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
            getContent: (
              _event: unknown,
              items: Array<{
                id: string;
                data?: {
                  rps?: number;
                  successRate?: number;
                  p99Ms?: number;
                  namespace?: string;
                  type?: string;
                  callCount?: number;
                  avgMs?: number;
                  errorRate?: number;
                };
              }>
            ) => {
              if (!items?.length) return "";
              const item = items[0];
              const d = item.data;
              if (!d) return item.id;

              // Node tooltip
              if (d.rps !== undefined) {
                const color = ringColor(
                  d.successRate !== undefined && d.successRate < 0.95 ? "critical" : "healthy",
                  d.successRate ?? 1
                );
                return `<div style="padding:8px 12px;font-size:12px;border-radius:8px;background:rgba(0,0,0,0.88);color:#fff;min-width:140px;border-left:3px solid ${color}">
                  <div style="font-weight:600;margin-bottom:4px">${item.id}</div>
                  ${d.namespace ? `<div style="opacity:0.6;margin-bottom:2px">${d.namespace}</div>` : ""}
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>RPS</span><span>${d.rps?.toFixed(3)}</span>
                  </div>
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>${t.successRate}</span><span>${((d.successRate ?? 1) * 100).toFixed(1)}%</span>
                  </div>
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>P99</span><span>${formatDurationMs(d.p99Ms ?? 0)}</span>
                  </div>
                </div>`;
              }

              // Edge tooltip
              if (d.callCount !== undefined) {
                return `<div style="padding:8px 12px;font-size:12px;border-radius:8px;background:rgba(0,0,0,0.88);color:#fff;min-width:120px">
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>${t.topoCalls}</span><span>${d.callCount}</span>
                  </div>
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>${t.topoLatency}</span><span>${formatDurationMs(d.avgMs ?? 0)}</span>
                  </div>
                  ${(d.errorRate ?? 0) > 0 ? `<div style="display:flex;justify-content:space-between;gap:12px;color:#f87171">
                    <span>${t.topoErrorRate}</span><span>${((d.errorRate ?? 0) * 100).toFixed(1)}%</span>
                  </div>` : ""}
                </div>`;
              }

              return item.id;
            },
          },
        ],
        animation: true,
      });

      let hoveredIds: string[] = [];

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:pointerover", (evt: any) => {
        const nodeId = evt?.target?.id;
        if (!nodeId) return;
        hoveredIds = [];
        const activate = (id: string) => { addState(instance, id, "active"); hoveredIds.push(id); };
        activate(nodeId);
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
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

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:click", (evt: any) => {
        const nodeId = evt?.target?.id;
        if (!nodeId) return;
        const nodeData = topology.nodes.find((n) => n.id === nodeId);
        try { applySelection(instance, nodeId); } catch { /* ignore */ }
        setSelectedNode(nodeId);
        if (nodeData?.type === "service") {
          onSelectRef.current(nodeData.name);
        }
      });

      instance.on("canvas:click", () => {
        try { applySelection(instance, null); } catch { /* ignore */ }
        setSelectedNode(null);
      });

      await instance.render();

      if (destroyed) {
        try { instance.destroy(); } catch { /* ignore */ }
        return;
      }

      graphRef.current = instance;
      readyRef.current = true;
    }

    createGraph();

    return () => {
      destroyed = true;
      readyRef.current = false;
      if (graphRef.current) {
        try { graphRef.current.destroy(); } catch { /* ignore */ }
        graphRef.current = null;
      }
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [topoKey]);

  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;
    try {
      const data = buildData();
      g.updateData(data);
      g.draw();
    } catch { /* ignore */ }
  }, [buildData]);

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

  // Suppress unused-var warning — selectedNode is set for future detail panel usage
  void selectedNode;

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
