"use client";

import { useRef, useEffect, useCallback, useMemo, useState } from "react";
import { Maximize2, Minimize2 } from "lucide-react";
import type { Topology, HealthStatus } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";

// Health-based ring color (Kibana-style softer palette)
function ringColor(status: HealthStatus, successRate: number): string {
  if (status === "critical" || successRate < 0.95) return "#ef4444";
  if (status === "warning" || successRate < 0.99) return "#f59e0b";
  return "#60a5fa"; // healthy = blue
}

// Node icon by type
function nodeIcon(type: string): string {
  if (type === "database") return "⛁";
  if (type === "external") return "⊕";
  return "⬡";
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

  const buildData = useCallback(() => {
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
          // Double-ring effect via halo
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
  }, [topology]);

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

  return (
    <div className="border border-[var(--border-color)] rounded-xl bg-card overflow-hidden">
      {/* Header */}
      <div className="px-4 py-2.5 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h3 className="text-sm font-medium text-default">{t.serviceTopology}</h3>
          <div className="flex items-center gap-2">
            <StatPill label={t.services} value={String(stats.nodeCount)} />
            <StatPill label={t.topoCalls} value={String(stats.totalCalls)} />
            <StatPill label="P99" value={formatDurationMs(stats.avgP99)} />
            {stats.errorNodes > 0 && (
              <StatPill label={t.topoErrorRate} value={String(stats.errorNodes)} variant="error" />
            )}
          </div>
        </div>
        <button
          onClick={() => setExpanded((v) => !v)}
          className="p-1.5 rounded-lg hover:bg-[var(--hover-bg)] transition-colors text-muted"
        >
          {expanded ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
        </button>
      </div>

      {/* Graph */}
      <div
        ref={containerRef}
        className="w-full transition-[height] duration-300 ease-in-out"
        style={{ height: expanded ? 480 : 320 }}
      />

      {/* Legend */}
      <div className="px-4 py-2 border-t border-[var(--border-color)] flex items-center gap-4 text-[10px] text-muted">
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-full border-2 border-[#60a5fa] bg-[#60a5fa]/10 inline-block" />
          {t.nodeTypeService}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rotate-45 border-2 border-[#60a5fa] bg-[#60a5fa]/10 inline-block" style={{ borderRadius: 2 }} />
          {t.nodeTypeDatabase}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-full border-2 border-[#f59e0b] bg-[#f59e0b]/10 inline-block" />
          {t.topoErrorRate} &gt;1%
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-full border-2 border-[#ef4444] bg-[#ef4444]/10 inline-block" />
          {t.topoErrorRate} &gt;5%
        </span>
      </div>
    </div>
  );
}

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
