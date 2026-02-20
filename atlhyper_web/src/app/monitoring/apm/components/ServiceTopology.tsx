"use client";

import { useRef, useEffect, useCallback, useMemo, useState } from "react";
import { Maximize2, Minimize2 } from "lucide-react";
import type { ServiceTopologyData } from "@/api/apm";
import type { ApmTranslations } from "@/types/i18n";

// Error rate -> node color (soft palette)
function errorColor(rate: number): string {
  if (rate >= 0.1) return "#ef4444";   // red-400: >=10%
  if (rate >= 0.01) return "#fbbf24";  // amber-400: >=1%
  return "#4ade80";                     // green-400: <1%
}

// Format duration for display
function fmtDuration(us: number): string {
  if (us >= 1_000_000) return `${(us / 1_000_000).toFixed(2)}s`;
  if (us >= 1_000) return `${(us / 1_000).toFixed(1)}ms`;
  return `${Math.round(us)}μs`;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function addState(g: any, id: string, state: string) {
  try {
    const cur: string[] = g.getElementState(id);
    if (!cur.includes(state)) g.setElementState(id, [...cur, state]);
  } catch { /* ignore */ }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function removeState(g: any, id: string, state: string) {
  try {
    const cur: string[] = g.getElementState(id);
    if (cur.includes(state)) g.setElementState(id, cur.filter((s: string) => s !== state));
  } catch { /* ignore */ }
}

/** Clear all "selected" state, then select a node + its edges */
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
    const ed = e as { id: string; source: string; target: string };
    if (ed.source === nodeId || ed.target === nodeId) {
      addState(g, ed.id, "selected");
    }
  }
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

  const [expanded, setExpanded] = useState(false);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  // Computed stats for header pills
  const stats = useMemo(() => {
    const nodeCount = topology.nodes.length;
    const edgeCount = topology.edges.length;
    const errorNodes = topology.nodes.filter((n) => n.errorRate >= 0.01).length;
    const totalLatency = topology.nodes.reduce((s, n) => s + n.latencyAvg, 0);
    const avgLatency = nodeCount > 0 ? totalLatency / nodeCount : 0;
    const totalCalls = topology.edges.reduce((s, e) => s + e.callCount, 0);
    return { nodeCount, edgeCount, errorNodes, avgLatency, totalCalls };
  }, [topology]);

  // Structural fingerprint
  const topoKey = useMemo(() => {
    const nk = topology.nodes.map((n) => n.id).sort().join(",");
    const ek = topology.edges.map((e) => e.id).sort().join(",");
    return nk + "||" + ek;
  }, [topology]);

  // Build G6 data
  const buildData = useCallback(() => {
    // Degree map for node sizing
    const degreeMap: Record<string, number> = {};
    for (const e of topology.edges) {
      degreeMap[e.source] = (degreeMap[e.source] ?? 0) + 1;
      degreeMap[e.target] = (degreeMap[e.target] ?? 0) + 1;
    }

    // Throughput range for size scaling
    const throughputs = topology.nodes.map((n) => n.throughput);
    const minT = Math.min(...throughputs, 1);
    const maxT = Math.max(...throughputs, 1);
    const tRange = maxT - minT || 1;

    const nodes = topology.nodes.map((n) => {
      const color = errorColor(n.errorRate);
      const degree = degreeMap[n.id] ?? 0;
      const baseSize = 30 + ((n.throughput - minT) / tRange) * 16; // 30~46px base
      const size = baseSize + Math.min(degree * 2, 10);             // +degree bonus

      const badges = n.errorRate >= 0.01
        ? [{
            text: `${(n.errorRate * 100).toFixed(0)}%`,
            placement: "right-top" as const,
            backgroundFill: n.errorRate >= 0.1 ? "#ef4444" : "#fbbf24",
            fill: "#fff",
            fontSize: 8,
          }]
        : [];

      return {
        id: n.id,
        data: { latencyAvg: n.latencyAvg, throughput: n.throughput, errorRate: n.errorRate },
        style: {
          type: "circle" as const,
          size,
          fill: color,
          fillOpacity: 0.2,
          stroke: color,
          lineWidth: 2,
          labelText: n.label.length > 22 ? n.label.slice(0, 20) + ".." : n.label,
          labelFontSize: 10,
          labelFill: color,
          labelPlacement: "bottom" as const,
          labelOffsetY: 4,
          labelBackground: true,
          labelBackgroundFill: "rgba(0,0,0,0.55)",
          labelBackgroundRadius: 3,
          labelBackgroundPadding: [1, 4, 1, 4],
          iconText: n.label[0]?.toUpperCase() ?? "S",
          iconFontSize: 13,
          iconFontWeight: 700,
          iconFill: color,
          cursor: "pointer" as const,
          badges,
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
          stroke: hasError ? "#f87171" : "#666",
          lineWidth: width,
          strokeOpacity: hasError ? 0.7 : 0.35,
          lineDash: hasError ? [6, 3] : (undefined as number[] | undefined),
          endArrow: true,
          endArrowSize: 6,
          endArrowFill: hasError ? "#f87171" : "#666",
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
        padding: [20, 20, 20, 20],
        data,
        node: {
          style: {
            cursor: "pointer" as const,
          },
          state: {
            active: {
              lineWidth: 3,
            },
            selected: {
              lineWidth: 3,
              shadowBlur: 18,
              shadowColor: "#3b82f6",
            },
          },
        },
        edge: {
          style: { type: "line" },
          state: {
            active: {
              stroke: "#3b82f6",
              lineWidth: 2,
              strokeOpacity: 0.8,
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
              items: Array<{
                id: string;
                data?: {
                  latencyAvg?: number;
                  throughput?: number;
                  errorRate?: number;
                  callCount?: number;
                  errorCount?: number;
                  avgLatency?: number;
                };
              }>
            ) => {
              if (!items?.length) return "";
              const item = items[0];
              const d = item.data;
              if (!d) return item.id;

              // Node tooltip
              if (d.throughput !== undefined) {
                const color = errorColor(d.errorRate ?? 0);
                const errPct = ((d.errorRate ?? 0) * 100).toFixed(1);
                return `<div style="padding:8px 12px;font-size:12px;border-radius:8px;background:rgba(0,0,0,0.88);color:#fff;min-width:140px;border-left:3px solid ${color}">
                  <div style="font-weight:600;margin-bottom:4px">${item.id}</div>
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>${t.topoLatency}</span><span>${fmtDuration(d.latencyAvg ?? 0)}</span>
                  </div>
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>${t.topoThroughput}</span><span>${d.throughput} traces</span>
                  </div>
                  <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
                    <span>${t.topoErrorRate}</span><span style="color:${(d.errorRate ?? 0) >= 0.01 ? color : '#fff'}">${errPct}%</span>
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
                    <span>${t.topoLatency}</span><span>${fmtDuration(d.avgLatency ?? 0)}</span>
                  </div>
                  ${d.errorCount ? `<div style="display:flex;justify-content:space-between;gap:12px;color:#f87171">
                    <span>${t.topoErrorRate}</span><span>${d.errorCount}</span>
                  </div>` : ""}
                </div>`;
              }

              return item.id;
            },
          },
        ],
        animation: true,
      });

      // ── Hover highlight (1-hop neighbors) ──
      let hoveredIds: string[] = [];

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:pointerover", (evt: any) => {
        const nodeId = evt?.target?.id;
        if (!nodeId) return;
        hoveredIds = [];
        const activate = (id: string) => {
          addState(instance, id, "active");
          hoveredIds.push(id);
        };
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

      // ── Click: persistent selection + navigate ──
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      instance.on("node:click", (evt: any) => {
        const nodeId = evt?.target?.id;
        if (!nodeId) return;
        try { applySelection(instance, nodeId); } catch { /* ignore */ }
        setSelectedNode(nodeId);
        onSelectRef.current(nodeId);
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

  // Effect 2: data changes -> update styles in place
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;
    try {
      const data = buildData();
      g.updateData(data);
      g.draw();
    } catch { /* ignore */ }
  }, [buildData]);

  // Resize graph when expanded changes
  useEffect(() => {
    const g = graphRef.current;
    if (!g || !readyRef.current) return;
    // Small delay to let DOM update height
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
      {/* Header: title + stats pills + expand toggle */}
      <div className="px-4 py-2.5 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h3 className="text-sm font-medium text-default">{t.serviceTopology}</h3>
          <div className="flex items-center gap-2">
            <StatPill label={t.services} value={String(stats.nodeCount)} />
            <StatPill label={t.topoCalls} value={String(stats.totalCalls)} />
            <StatPill label={t.topoLatency} value={fmtDuration(stats.avgLatency)} />
            {stats.errorNodes > 0 && (
              <StatPill label={t.topoErrorRate} value={String(stats.errorNodes)} variant="error" />
            )}
          </div>
        </div>
        <button
          onClick={() => setExpanded((v) => !v)}
          className="p-1.5 rounded-lg hover:bg-[var(--hover-bg)] transition-colors text-muted"
          title={expanded ? "Collapse" : "Expand"}
        >
          {expanded ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
        </button>
      </div>

      {/* Graph container */}
      <div
        ref={containerRef}
        className="w-full transition-[height] duration-300 ease-in-out"
        style={{ height: expanded ? 480 : 320 }}
      />

      {/* Footer: color legend */}
      <div className="px-4 py-2 border-t border-[var(--border-color)] flex items-center gap-4 text-[10px] text-muted">
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-full bg-[#4ade80] inline-block" />
          {"< 1%"}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-full bg-[#fbbf24] inline-block" />
          {"1-10%"}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-full bg-[#ef4444] inline-block" />
          {"> 10%"}
        </span>
        <span className="text-[var(--border-color)]">|</span>
        <span className="flex items-center gap-1.5">
          <span className="w-4 border-t border-[#666] inline-block" />
          {t.topoCalls}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-4 border-t border-dashed border-[#f87171] inline-block" />
          {t.topoErrorRate}
        </span>
      </div>
    </div>
  );
}

/** Small stat pill for header */
function StatPill({ label, value, variant }: { label: string; value: string; variant?: "error" }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] ${
        variant === "error"
          ? "bg-red-500/10 text-red-400"
          : "bg-[var(--hover-bg)] text-muted"
      }`}
    >
      <span className="opacity-70">{label}</span>
      <span className="font-semibold text-default">{value}</span>
    </span>
  );
}
