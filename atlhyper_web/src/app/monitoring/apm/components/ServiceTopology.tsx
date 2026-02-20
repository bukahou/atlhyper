"use client";

import { useRef, useEffect, useCallback, useMemo } from "react";
import type { ServiceTopologyData } from "@/api/apm";
import type { ApmTranslations } from "@/types/i18n";

// Error rate -> node color
function errorColor(rate: number): string {
  if (rate >= 0.1) return "#ef4444";  // red-400
  if (rate >= 0.01) return "#fbbf24"; // amber-400
  return "#4ade80";                    // green-400
}

// Format duration for tooltip
function fmtDuration(us: number): string {
  if (us >= 1_000_000) return `${(us / 1_000_000).toFixed(2)}s`;
  if (us >= 1_000) return `${(us / 1_000).toFixed(1)}ms`;
  return `${Math.round(us)}μs`;
}

interface ServiceTopologyProps {
  t: ApmTranslations;
  topology: ServiceTopologyData;
  onSelectService: (name: string) => void;
}

export function ServiceTopology({ t, topology, onSelectService }: ServiceTopologyProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const graphRef = useRef<any>(null);
  const readyRef = useRef(false);
  const onSelectRef = useRef(onSelectService);
  onSelectRef.current = onSelectService;

  // Structural fingerprint: rebuild only when node/edge sets change
  const topoKey = useMemo(() => {
    const nk = topology.nodes.map((n) => n.id).sort().join(",");
    const ek = topology.edges.map((e) => e.id).sort().join(",");
    return nk + "||" + ek;
  }, [topology]);

  // Build G6 data
  const buildData = useCallback(() => {
    // Compute throughput range for node size scaling
    const throughputs = topology.nodes.map((n) => n.throughput);
    const minT = Math.min(...throughputs, 1);
    const maxT = Math.max(...throughputs, 1);
    const tRange = maxT - minT || 1;

    const nodes = topology.nodes.map((n) => {
      const color = errorColor(n.errorRate);
      const size = 28 + ((n.throughput - minT) / tRange) * 20; // 28~48px

      return {
        id: n.id,
        data: { latencyAvg: n.latencyAvg, throughput: n.throughput, errorRate: n.errorRate },
        style: {
          type: "circle" as const,
          size,
          fill: color,
          fillOpacity: 0.25,
          stroke: color,
          lineWidth: 2,
          labelText: n.label.length > 20 ? n.label.slice(0, 18) + ".." : n.label,
          labelFontSize: 10,
          labelFill: color,
          labelPlacement: "bottom" as const,
          labelOffsetY: 4,
          labelBackground: true,
          labelBackgroundFill: "rgba(0,0,0,0.5)",
          labelBackgroundRadius: 3,
          labelBackgroundPadding: [1, 4, 1, 4],
          cursor: "pointer" as const,
        },
      };
    });

    // Edge width scaling
    const callCounts = topology.edges.map((e) => e.callCount);
    const minC = Math.min(...callCounts, 1);
    const maxC = Math.max(...callCounts, 1);
    const cRange = maxC - minC || 1;

    const edges = topology.edges.map((e) => {
      const width = 1 + ((e.callCount - minC) / cRange) * 3; // 1~4px
      const hasError = e.errorCount > 0;
      return {
        id: e.id,
        source: e.source,
        target: e.target,
        data: { callCount: e.callCount, errorCount: e.errorCount, avgLatency: e.avgLatency },
        style: {
          stroke: hasError ? "#f87171" : "#888",
          lineWidth: width,
          strokeOpacity: hasError ? 0.7 : 0.4,
          endArrow: true,
          endArrowSize: 6,
          endArrowFill: hasError ? "#f87171" : "#888",
        },
      };
    });

    return { nodes, edges };
  }, [topology]);

  // Effect 1: topology structure change -> destroy & rebuild
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
          state: {
            active: { lineWidth: 3 },
          },
        },
        edge: {
          style: { type: "line" },
          state: {
            active: { stroke: "#3b82f6", lineWidth: 2, strokeOpacity: 0.8 },
          },
        },
        layout: {
          type: "d3-force",
          link: { distance: 160 },
          charge: { strength: -500 },
          collide: { radius: 50 },
        },
        behaviors: ["drag-canvas", "zoom-canvas", "drag-element"],
        plugins: [
          {
            type: "tooltip",
            getContent: (
              _event: unknown,
              items: Array<{ id: string; data?: { latencyAvg?: number; throughput?: number; errorRate?: number; callCount?: number; errorCount?: number; avgLatency?: number } }>
            ) => {
              if (!items?.length) return "";
              const item = items[0];
              const d = item.data;
              if (!d) return item.id;

              // Node tooltip
              if (d.throughput !== undefined) {
                return `<div style="padding:6px 10px;font-size:12px;border-radius:6px;background:rgba(0,0,0,0.85);color:#fff">
                  <b>${item.id}</b><br/>
                  <span style="opacity:0.7">${t.topoLatency}: ${fmtDuration(d.latencyAvg ?? 0)}</span><br/>
                  <span style="opacity:0.7">${t.topoThroughput}: ${d.throughput} traces</span><br/>
                  <span style="opacity:0.7">${t.topoErrorRate}: ${((d.errorRate ?? 0) * 100).toFixed(1)}%</span>
                </div>`;
              }

              // Edge tooltip
              if (d.callCount !== undefined) {
                return `<div style="padding:6px 10px;font-size:12px;border-radius:6px;background:rgba(0,0,0,0.85);color:#fff">
                  <span style="opacity:0.7">${t.topoCalls}: ${d.callCount}</span><br/>
                  <span style="opacity:0.7">${t.topoLatency}: ${fmtDuration(d.avgLatency ?? 0)}</span><br/>
                  <span style="opacity:0.7">${t.topoErrorRate}: ${d.errorCount}</span>
                </div>`;
              }

              return item.id;
            },
          },
        ],
        animation: true,
      });

      // Hover: highlight 1-hop neighbors
      let hoveredIds: string[] = [];

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const addState = (id: string, state: string) => {
        try {
          const cur: string[] = instance.getElementState(id);
          if (!cur.includes(state)) instance.setElementState(id, [...cur, state]);
          hoveredIds.push(id);
        } catch { /* ignore */ }
      };

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:pointerover", (evt: any) => {
        const nodeId = evt?.target?.id;
        if (!nodeId) return;
        hoveredIds = [];
        addState(nodeId, "active");
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        for (const e of instance.getEdgeData()) {
          const ed = e as { id: string; source: string; target: string };
          if (ed.source === nodeId || ed.target === nodeId) {
            addState(ed.id, "active");
            addState(ed.source === nodeId ? ed.target : ed.source, "active");
          }
        }
      });

      instance.on("node:pointerout", () => {
        try {
          for (const id of hoveredIds) {
            const cur: string[] = instance.getElementState(id);
            if (cur.includes("active")) {
              instance.setElementState(id, cur.filter((s: string) => s !== "active"));
            }
          }
          hoveredIds = [];
        } catch { /* ignore */ }
      });

      // Click: navigate to service detail
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:click", (evt: any) => {
        const nodeId = evt?.target?.id;
        if (nodeId) onSelectRef.current(nodeId);
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

  // Effect 2: data changes (metrics) -> update styles in place
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;

    try {
      const data = buildData();
      g.updateData(data);
      g.draw();
    } catch {
      // G6 not ready, next structural change will rebuild
    }
  }, [buildData]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full"
    />
  );
}
