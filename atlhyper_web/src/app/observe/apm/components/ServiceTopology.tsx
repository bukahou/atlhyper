"use client";

import { useRef, useEffect, useCallback, useMemo, useState } from "react";
import { Maximize2, Minimize2 } from "lucide-react";
import type { Topology, HealthStatus } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";

// Node color based on status + successRate
function nodeColor(status: HealthStatus, successRate: number): string {
  if (status === "critical" || successRate < 0.95) return "#ef4444";
  if (status === "warning" || successRate < 0.99) return "#fbbf24";
  return "#4ade80";
}

// Node shape icon by type
function nodeIcon(type: string): string {
  if (type === "database") return "D";
  if (type === "external") return "E";
  return "S";
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
    // RPS range for node sizing
    const rpsValues = topology.nodes.map((n) => n.rps);
    const minRps = Math.min(...rpsValues, 0);
    const maxRps = Math.max(...rpsValues, 0.001);
    const rpsRange = maxRps - minRps || 1;

    const nodes = topology.nodes.map((n) => {
      const color = nodeColor(n.status, n.successRate);
      const baseSize = 30 + ((n.rps - minRps) / rpsRange) * 16;

      const badges = n.successRate < 0.99
        ? [{
            text: `${((1 - n.successRate) * 100).toFixed(1)}%`,
            placement: "right-top" as const,
            backgroundFill: n.successRate < 0.95 ? "#ef4444" : "#fbbf24",
            fill: "#fff",
            fontSize: 8,
          }]
        : [];

      return {
        id: n.id,
        data: { rps: n.rps, successRate: n.successRate, p99Ms: n.p99Ms, namespace: n.namespace, type: n.type },
        style: {
          type: (n.type === "database" ? "diamond" : "circle") as "circle" | "diamond",
          size: baseSize,
          fill: color,
          fillOpacity: 0.2,
          stroke: color,
          lineWidth: 2,
          labelText: n.name.length > 22 ? n.name.slice(0, 20) + ".." : n.name,
          labelFontSize: 10,
          labelFill: color,
          labelPlacement: "bottom" as const,
          labelOffsetY: 4,
          labelBackground: true,
          labelBackgroundFill: "rgba(0,0,0,0.55)",
          labelBackgroundRadius: 3,
          labelBackgroundPadding: [1, 4, 1, 4],
          iconText: nodeIcon(n.type),
          iconFontSize: 13,
          iconFontWeight: 700,
          iconFill: color,
          cursor: "pointer" as const,
          badges,
        },
      };
    });

    // Edge width by callCount
    const callCounts = topology.edges.map((e) => e.callCount);
    const minC = Math.min(...callCounts, 1);
    const maxC = Math.max(...callCounts, 1);
    const cRange = maxC - minC || 1;

    const edges = topology.edges.map((e) => {
      const width = 1 + ((e.callCount - minC) / cRange) * 3;
      const hasError = e.errorRate > 0;
      return {
        id: `${e.source}>${e.target}`,
        source: e.source,
        target: e.target,
        data: { callCount: e.callCount, avgMs: e.avgMs, errorRate: e.errorRate },
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
          style: { cursor: "pointer" as const },
          state: {
            active: { lineWidth: 3 },
            selected: { lineWidth: 3, shadowBlur: 18, shadowColor: "#3b82f6" },
          },
        },
        edge: {
          style: { type: "line" },
          state: {
            active: { stroke: "#3b82f6", lineWidth: 2, strokeOpacity: 0.8 },
            selected: { stroke: "#3b82f6", lineWidth: 2.5, strokeOpacity: 1 },
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
                const color = nodeColor(
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
        // Only navigate if it's a service node (not database/external)
        const nodeData = topology.nodes.find((n) => n.id === nodeId);
        try { applySelection(instance, nodeId); } catch { /* ignore */ }
        setSelectedNode(nodeId);
        if (nodeData?.type === "service") {
          onSelectRef.current(nodeId);
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
          <span className="w-2.5 h-2.5 rounded-full bg-[#4ade80] inline-block" />
          {t.nodeTypeService}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rotate-45 bg-[#60a5fa] inline-block" style={{ borderRadius: 2 }} />
          {t.nodeTypeDatabase}
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-full bg-[#a78bfa] inline-block" />
          {t.nodeTypeExternal}
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
