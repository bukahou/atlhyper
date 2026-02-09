"use client";

import { useState, useMemo, useRef, useEffect, useCallback } from "react";
import { Network, Server, ArrowRight, Layers, Shield, ZoomIn, ZoomOut, Shrink, BarChart3, Loader2 } from "lucide-react";
import { getNamespaceColor } from "./common";
import { getMeshServiceDetail } from "@/api/mesh";
import type { MeshServiceNode, MeshServiceEdge, MeshTopologyResponse, MeshServiceDetailResponse } from "@/types/mesh";

interface MeshTabTranslations {
  serviceTopology: string;
  meshOverview: string;
  service: string;
  rps: string;
  p95Latency: string;
  errorRate: string;
  mtls: string;
  status: string;
  healthy: string;
  warning: string;
  critical: string;
  inbound: string;
  outbound: string;
  noCallData: string;
  callRelation: string;
  p50Latency: string;
  p99Latency: string;
  totalRequests: string;
  avgLatency: string;
  statusCodeBreakdown: string;
  latencyDistribution: string;
  requests: string;
  loading: string;
}

// Service Topology View (SVG) with zoom & pan
function ServiceTopologyView({ topology, onSelectNode, timeRange, t }: {
  topology: MeshTopologyResponse;
  onSelectNode?: (node: MeshServiceNode) => void;
  timeRange: string;
  t: MeshTabTranslations;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [hoveredEdge, setHoveredEdge] = useState<number | null>(null);
  const [draggingNode, setDraggingNode] = useState<string | null>(null);
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
  const [containerWidth, setContainerWidth] = useState(1000);
  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({});
  const [initialized, setInitialized] = useState(false);

  // Zoom & pan
  const [zoom, setZoom] = useState(1);
  const [viewOrigin, setViewOrigin] = useState({ x: 0, y: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const panStartRef = useRef({ x: 0, y: 0 });
  const panOriginRef = useRef({ x: 0, y: 0 });

  const nodeRadius = 32;
  const svgHeight = 500;
  const MIN_ZOOM = 0.2;
  const MAX_ZOOM = 4;

  // Group nodes by namespace for swim-lane layout
  const nsGroups = useMemo(() => {
    const groups: Record<string, string[]> = {};
    topology.nodes.forEach(n => {
      if (!groups[n.namespace]) groups[n.namespace] = [];
      groups[n.namespace].push(n.id);
    });
    return groups;
  }, [topology]);
  const sortedNamespaces = useMemo(() => Object.keys(nsGroups).sort(), [nsGroups]);
  const [nsLaneBounds, setNsLaneBounds] = useState<Record<string, { minX: number; minY: number; maxX: number; maxY: number }>>({});

  useEffect(() => {
    const updateWidth = () => { if (containerRef.current) setContainerWidth(containerRef.current.offsetWidth); };
    updateWidth();
    window.addEventListener("resize", updateWidth);
    return () => window.removeEventListener("resize", updateWidth);
  }, []);

  // Topological sort for layered layout
  const nodeLayers = useMemo(() => {
    const nodeIds = topology.nodes.map(n => n.id);
    const inDegree: Record<string, number> = {};
    const outEdges: Record<string, string[]> = {};
    nodeIds.forEach(id => { inDegree[id] = 0; outEdges[id] = []; });
    topology.edges.forEach(edge => {
      if (nodeIds.includes(edge.source) && nodeIds.includes(edge.target)) {
        inDegree[edge.target]++;
        outEdges[edge.source].push(edge.target);
      }
    });
    const level: Record<string, number> = {};
    const queue: string[] = [];
    nodeIds.forEach(id => { if (inDegree[id] === 0) { level[id] = 0; queue.push(id); } });
    while (queue.length > 0) {
      const current = queue.shift()!;
      outEdges[current].forEach(next => {
        const newLevel = level[current] + 1;
        if (level[next] === undefined || level[next] < newLevel) level[next] = newLevel;
        inDegree[next]--;
        if (inDegree[next] === 0) queue.push(next);
      });
    }
    nodeIds.forEach(id => { if (level[id] === undefined) level[id] = 0; });
    const maxLevel = Math.max(...Object.values(level), 0);
    const layers: string[][] = Array.from({ length: maxLevel + 1 }, () => []);
    nodeIds.forEach(id => layers[level[id]].push(id));
    return layers;
  }, [topology]);

  // Namespace swim-lane layout: each namespace gets a horizontal band,
  // nodes within each band are placed by topological layer (left→right)
  useEffect(() => {
    if (initialized || containerWidth < 100) return;
    const paddingX = 80, paddingY = 50, lanePadding = 35, laneGap = 20, nodeGapY = 85;
    const usableWidth = containerWidth - paddingX * 2;
    const layerCount = nodeLayers.length;
    const layerGapX = layerCount > 1 ? usableWidth / (layerCount - 1) : 0;

    const pos: Record<string, { x: number; y: number }> = {};
    const bounds: Record<string, { minX: number; minY: number; maxX: number; maxY: number }> = {};
    let currentY = paddingY;

    sortedNamespaces.forEach(ns => {
      const nsNodeIds = nsGroups[ns];
      // Map each node to its topological layer
      const nodesPerLayer: Record<number, string[]> = {};
      nsNodeIds.forEach(nodeId => {
        const layerIdx = nodeLayers.findIndex(layer => layer.includes(nodeId));
        if (layerIdx >= 0) {
          if (!nodesPerLayer[layerIdx]) nodesPerLayer[layerIdx] = [];
          nodesPerLayer[layerIdx].push(nodeId);
        }
      });
      const maxNodesInAnyLayer = Math.max(...Object.values(nodesPerLayer).map(a => a.length), 1);
      const laneHeight = (maxNodesInAnyLayer - 1) * nodeGapY + lanePadding * 2;
      const laneCenterY = currentY + laneHeight / 2;

      Object.entries(nodesPerLayer).forEach(([li, nodeIds]) => {
        const layerIdx = parseInt(li);
        const x = layerCount > 1 ? paddingX + layerIdx * layerGapX : containerWidth / 2;
        const startY = laneCenterY - ((nodeIds.length - 1) * nodeGapY) / 2;
        nodeIds.forEach((nodeId, idx) => { pos[nodeId] = { x, y: startY + idx * nodeGapY }; });
      });

      bounds[ns] = { minX: paddingX - lanePadding, minY: currentY, maxX: containerWidth - paddingX + lanePadding, maxY: currentY + laneHeight };
      currentY += laneHeight + laneGap;
    });

    setPositions(pos);
    setNsLaneBounds(bounds);

    // Auto-fit zoom to show all content
    const posValues = Object.values(pos);
    if (posValues.length > 0) {
      const pad = nodeRadius + 40;
      const minX = Math.min(...posValues.map(p => p.x)) - pad;
      const maxX = Math.max(...posValues.map(p => p.x)) + pad;
      const minY = Math.min(...posValues.map(p => p.y)) - pad;
      const maxY = Math.max(...posValues.map(p => p.y)) + pad;
      const contentW = maxX - minX;
      const contentH = maxY - minY;
      const fitZoom = Math.min(containerWidth / contentW, svgHeight / contentH, MAX_ZOOM);
      setZoom(fitZoom);
      setViewOrigin({
        x: minX - (containerWidth / fitZoom - contentW) / 2,
        y: minY - (svgHeight / fitZoom - contentH) / 2,
      });
    }
    setInitialized(true);
  }, [containerWidth, nodeLayers, nsGroups, sortedNamespaces, initialized]);

  // viewBox computed from zoom and pan origin
  const viewBox = `${viewOrigin.x} ${viewOrigin.y} ${containerWidth / zoom} ${svgHeight / zoom}`;

  const getEdgePath = (sourceId: string, targetId: string) => {
    const source = positions[sourceId], target = positions[targetId];
    if (!source || !target) return "";
    const startX = source.x + nodeRadius, startY = source.y, endX = target.x - nodeRadius, endY = target.y;
    const midX = (startX + endX) / 2;
    return `M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`;
  };

  const getEdgeLabelPos = (sourceId: string, targetId: string) => {
    const source = positions[sourceId], target = positions[targetId];
    if (!source || !target) return { x: 0, y: 0 };
    return { x: (source.x + target.x) / 2, y: (source.y + target.y) / 2 };
  };

  // Wheel zoom disabled — only use button controls to avoid scroll hijacking

  // Node drag start
  const handleNodeMouseDown = (e: React.MouseEvent, nodeId: string) => {
    if (e.button !== 0) return;
    e.stopPropagation();
    const svg = svgRef.current;
    if (!svg) return;
    const pt = svg.createSVGPoint();
    pt.x = e.clientX; pt.y = e.clientY;
    const svgP = pt.matrixTransform(svg.getScreenCTM()?.inverse());
    const pos = positions[nodeId];
    setDragOffset({ x: svgP.x - pos.x, y: svgP.y - pos.y });
    setDraggingNode(nodeId);
  };

  // Pan start (background click)
  const handleSvgMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return;
    setIsPanning(true);
    panStartRef.current = { x: e.clientX, y: e.clientY };
    panOriginRef.current = { ...viewOrigin };
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isPanning) {
      const dx = e.clientX - panStartRef.current.x;
      const dy = e.clientY - panStartRef.current.y;
      setViewOrigin({
        x: panOriginRef.current.x - dx / zoom,
        y: panOriginRef.current.y - dy / zoom,
      });
      return;
    }
    if (!draggingNode) return;
    const svg = svgRef.current;
    if (!svg) return;
    const pt = svg.createSVGPoint();
    pt.x = e.clientX; pt.y = e.clientY;
    const svgP = pt.matrixTransform(svg.getScreenCTM()?.inverse());
    setPositions(prev => ({ ...prev, [draggingNode]: {
      x: svgP.x - dragOffset.x,
      y: svgP.y - dragOffset.y,
    }}));
  };

  const handleMouseUp = () => {
    setDraggingNode(null);
    setIsPanning(false);
  };

  // Zoom controls
  const zoomBy = (factor: number) => {
    setZoom(prev => {
      const newZoom = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, prev * factor));
      const cx = containerWidth / 2;
      const cy = svgHeight / 2;
      setViewOrigin(vo => ({
        x: vo.x + cx * (1 / prev - 1 / newZoom),
        y: vo.y + cy * (1 / prev - 1 / newZoom),
      }));
      return newZoom;
    });
  };

  const fitToContent = () => {
    const posValues = Object.values(positions);
    if (posValues.length === 0) { setZoom(1); setViewOrigin({ x: 0, y: 0 }); return; }
    const pad = nodeRadius + 40;
    const minX = Math.min(...posValues.map(p => p.x)) - pad;
    const maxX = Math.max(...posValues.map(p => p.x)) + pad;
    const minY = Math.min(...posValues.map(p => p.y)) - pad;
    const maxY = Math.max(...posValues.map(p => p.y)) + pad;
    const contentW = maxX - minX;
    const contentH = maxY - minY;
    const fitZoom = Math.min(containerWidth / contentW, svgHeight / contentH, MAX_ZOOM);
    setZoom(fitZoom);
    setViewOrigin({
      x: minX - (containerWidth / fitZoom - contentW) / 2,
      y: minY - (svgHeight / fitZoom - contentH) / 2,
    });
  };

  const namespacesInTopology = useMemo(() => Array.from(new Set(topology.nodes.map(n => n.namespace))), [topology]);

  return (
    <div className="p-4 bg-[var(--hover-bg)] rounded-lg">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Network className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.serviceTopology}</span>
          <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400">Linkerd</span>
          <span className="text-[10px] text-muted">{topology.nodes.length} services · {timeRangeLabel(timeRange)}</span>
        </div>
        <div className="flex items-center gap-4 text-[11px]">
          {namespacesInTopology.map(ns => {
            const colors = getNamespaceColor(ns);
            return (
              <div key={ns} className="flex items-center gap-1.5">
                <span className="w-3 h-3 rounded-full border-2" style={{ backgroundColor: colors.fill, borderColor: colors.fill }} />
                <span className="text-muted">{ns}</span>
              </div>
            );
          })}
        </div>
      </div>
      <div ref={containerRef} className="relative rounded-xl bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800 border border-slate-200 dark:border-slate-700 overflow-hidden">
        {/* Zoom controls */}
        <div className="absolute top-3 right-3 z-10 flex items-center gap-1 bg-white/90 dark:bg-slate-800/90 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 px-1 py-0.5">
          <button onClick={() => zoomBy(1.3)} className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors" title="Zoom in">
            <ZoomIn className="w-3.5 h-3.5 text-slate-600 dark:text-slate-300" />
          </button>
          <span className="text-[10px] text-muted font-medium w-10 text-center select-none">{Math.round(zoom * 100)}%</span>
          <button onClick={() => zoomBy(0.77)} className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors" title="Zoom out">
            <ZoomOut className="w-3.5 h-3.5 text-slate-600 dark:text-slate-300" />
          </button>
          <div className="w-px h-4 bg-slate-200 dark:bg-slate-600 mx-0.5" />
          <button onClick={fitToContent} className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors" title="Fit to content">
            <Shrink className="w-3.5 h-3.5 text-slate-600 dark:text-slate-300" />
          </button>
        </div>

        <svg ref={svgRef} width={containerWidth} height={svgHeight} viewBox={viewBox}
          className={`select-none ${isPanning ? "cursor-grabbing" : "cursor-grab"}`}
          onMouseDown={handleSvgMouseDown} onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp} onMouseLeave={handleMouseUp}>
          <defs>
            <marker id="arrow-gray" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#94a3b8" /></marker>
            <marker id="arrow-blue" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#0891b2" /></marker>
            <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%"><feDropShadow dx="0" dy="2" stdDeviation="3" floodOpacity="0.2" /></filter>
          </defs>
          <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#e2e8f0" strokeWidth="0.5" className="dark:stroke-slate-700" /></pattern>
          <rect x={viewOrigin.x} y={viewOrigin.y} width={containerWidth / zoom} height={svgHeight / zoom} fill="url(#grid)" opacity="0.5" />

          {/* Namespace labels (layout preserved, no bounding box) */}
          {sortedNamespaces.map(ns => {
            const b = nsLaneBounds[ns];
            if (!b) return null;
            const colors = getNamespaceColor(ns);
            return (
              <text key={`ns-${ns}`} x={b.minX + 14} y={b.minY + 16}
                fontSize="11" fontWeight="600" fill={colors.stroke} fillOpacity={0.5}>
                {ns}
              </text>
            );
          })}

          {/* Edges */}
          <g>{topology.edges.map((edge, idx) => {
            const isHighlighted = hoveredNode === edge.source || hoveredNode === edge.target || selectedNode === edge.source || selectedNode === edge.target || hoveredEdge === idx;
            let strokeColor = "#cbd5e1";
            if (edge.error_rate > 1) strokeColor = isHighlighted ? "#ef4444" : "#fca5a5";
            else if (edge.error_rate > 0.1) strokeColor = isHighlighted ? "#f59e0b" : "#fcd34d";
            else if (isHighlighted) strokeColor = "#0ea5e9";
            const labelPos = getEdgeLabelPos(edge.source, edge.target);
            return (
              <g key={idx} onMouseEnter={() => setHoveredEdge(idx)} onMouseLeave={() => setHoveredEdge(null)}>
                <path d={getEdgePath(edge.source, edge.target)} fill="none" stroke="transparent" strokeWidth={12} style={{ cursor: "pointer" }} />
                <path d={getEdgePath(edge.source, edge.target)} fill="none" stroke={strokeColor} strokeWidth={isHighlighted ? 3 : 2} markerEnd={`url(#${isHighlighted ? "arrow-blue" : "arrow-gray"})`} className="transition-colors duration-200" />
                {isHighlighted && (
                  <g transform={`translate(${labelPos.x}, ${labelPos.y})`}>
                    <rect x="-42" y="-12" width="84" height="24" rx="4" fill="white" className="dark:fill-slate-800" stroke="#e2e8f0" strokeWidth="1" />
                    <text textAnchor="middle" y="4" className="text-[10px] font-medium fill-slate-600 dark:fill-slate-300">{edge.rps.toFixed(0)}/s · {edge.avg_latency}ms</text>
                  </g>
                )}
              </g>
            );
          })}</g>

          {/* Nodes */}
          <g>{topology.nodes.map((node) => {
            const pos = positions[node.id];
            if (!pos) return null;
            const colors = getNamespaceColor(node.namespace);
            const isSelected = selectedNode === node.id;
            const isHovered = hoveredNode === node.id;
            const isDragging = draggingNode === node.id;
            return (
              <g key={node.id} transform={`translate(${pos.x}, ${pos.y})`}
                onMouseDown={(e) => handleNodeMouseDown(e, node.id)}
                onMouseEnter={() => !draggingNode && setHoveredNode(node.id)}
                onMouseLeave={() => !draggingNode && setHoveredNode(null)}
                onClick={() => { if (!isDragging) { setSelectedNode(isSelected ? null : node.id); onSelectNode?.(node); } }}
                style={{ cursor: isDragging ? "grabbing" : "grab" }}>
                {(isSelected || isHovered) && <circle r={nodeRadius + 5} fill="none" stroke={colors.light} strokeWidth={2} strokeOpacity={isSelected ? 0.8 : 0.5} />}
                <circle r={nodeRadius} fill={colors.fill} stroke={colors.stroke} strokeWidth={2} filter="url(#shadow)" />
                <g style={{ transform: "translate(-10px, -10px)" }}>
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <rect width="20" height="8" x="2" y="2" rx="2" ry="2"/><rect width="20" height="8" x="2" y="14" rx="2" ry="2"/>
                    <line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/>
                  </svg>
                </g>
                <text y={nodeRadius + 16} textAnchor="middle" className="text-[11px] font-semibold fill-slate-700 dark:fill-slate-200 pointer-events-none">
                  {node.name.length > 14 ? node.name.slice(0, 14) + "\u2026" : node.name}
                </text>
                <text y={nodeRadius + 28} textAnchor="middle" className="text-[9px] fill-slate-500 dark:fill-slate-400 pointer-events-none">
                  {node.p95_latency}ms · {node.error_rate.toFixed(1)}%
                </text>
                <circle cx={nodeRadius - 4} cy={-nodeRadius + 4} r={5}
                  fill={node.status === "healthy" ? "#10b981" : node.status === "warning" ? "#f59e0b" : "#ef4444"}
                  stroke="white" strokeWidth={2} />
              </g>
            );
          })}</g>
        </svg>
      </div>
    </div>
  );
}

// Service List Table (sortable)
function ServiceListTable({ nodes, selectedId, onSelect, t }: {
  nodes: MeshServiceNode[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  t: MeshTabTranslations;
}) {
  const [sortKey, setSortKey] = useState<"name" | "rps" | "p95_latency" | "error_rate" | "mtls_percent">("rps");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  const toggleSort = (key: typeof sortKey) => {
    if (sortKey === key) setSortDir(d => d === "asc" ? "desc" : "asc");
    else { setSortKey(key); setSortDir("desc"); }
  };

  const sorted = useMemo(() => {
    const arr = [...nodes];
    arr.sort((a, b) => {
      const aVal = sortKey === "name" ? a.name : a[sortKey];
      const bVal = sortKey === "name" ? b.name : b[sortKey];
      if (typeof aVal === "string" && typeof bVal === "string") return sortDir === "asc" ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      return sortDir === "asc" ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
    });
    return arr;
  }, [nodes, sortKey, sortDir]);

  const SortHeader = ({ label, field }: { label: string; field: typeof sortKey }) => (
    <button onClick={() => toggleSort(field)}
      className={`flex items-center gap-1 text-[10px] font-medium uppercase tracking-wider ${sortKey === field ? "text-primary" : "text-muted hover:text-default"}`}>
      {label}
      {sortKey === field && <span className="text-[8px]">{sortDir === "asc" ? "\u25B2" : "\u25BC"}</span>}
    </button>
  );

  return (
    <div className="overflow-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-[var(--border-color)]">
            <th className="text-left py-2 px-2"><SortHeader label={t.service} field="name" /></th>
            <th className="text-right py-2 px-2"><SortHeader label={t.rps} field="rps" /></th>
            <th className="text-right py-2 px-2"><SortHeader label="P95" field="p95_latency" /></th>
            <th className="text-right py-2 px-2"><SortHeader label={t.errorRate} field="error_rate" /></th>
            <th className="text-right py-2 px-2"><SortHeader label={t.mtls} field="mtls_percent" /></th>
            <th className="text-center py-2 px-2"><span className="text-[10px] font-medium uppercase tracking-wider text-muted">{t.status}</span></th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((node) => {
            const nsColor = getNamespaceColor(node.namespace);
            return (
              <tr key={node.id} onClick={() => onSelect(node.id)}
                className={`cursor-pointer transition-colors border-b border-[var(--border-color)] ${selectedId === node.id ? "bg-primary/5 dark:bg-primary/10" : "hover:bg-[var(--hover-bg)]"}`}>
                <td className="py-2.5 px-2">
                  <div className="flex items-center gap-2">
                    <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: nsColor.fill }} />
                    <div>
                      <div className="font-medium text-default">{node.name}</div>
                      <div className="text-[10px] text-muted">{node.namespace}</div>
                    </div>
                  </div>
                </td>
                <td className="text-right py-2.5 px-2 font-medium text-default">{node.rps.toFixed(0)}<span className="text-muted">/s</span></td>
                <td className="text-right py-2.5 px-2 font-medium text-default">{node.p95_latency}<span className="text-muted">ms</span></td>
                <td className="text-right py-2.5 px-2">
                  <span className={node.error_rate > 0.5 ? "text-red-500 font-semibold" : "text-default font-medium"}>{node.error_rate.toFixed(2)}%</span>
                </td>
                <td className="text-right py-2.5 px-2">
                  <span className={`font-semibold ${node.mtls_percent >= 100 ? "text-emerald-600 dark:text-emerald-400" : node.mtls_percent >= 80 ? "text-amber-600 dark:text-amber-400" : "text-red-600 dark:text-red-400"}`}>
                    {node.mtls_percent.toFixed(0)}%
                  </span>
                </td>
                <td className="text-center py-2.5 px-2">
                  <span className={`inline-block w-2 h-2 rounded-full ${node.status === "healthy" ? "bg-emerald-500" : node.status === "warning" ? "bg-amber-500" : "bg-red-500"}`} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// Status code colors
const statusColors: Record<string, { bar: string; bg: string; text: string }> = {
  "2xx": { bar: "bg-emerald-500", bg: "bg-emerald-50 dark:bg-emerald-900/20", text: "text-emerald-700 dark:text-emerald-400" },
  "3xx": { bar: "bg-blue-500", bg: "bg-blue-50 dark:bg-blue-900/20", text: "text-blue-700 dark:text-blue-400" },
  "4xx": { bar: "bg-amber-500", bg: "bg-amber-50 dark:bg-amber-900/20", text: "text-amber-700 dark:text-amber-400" },
  "5xx": { bar: "bg-red-500", bg: "bg-red-50 dark:bg-red-900/20", text: "text-red-700 dark:text-red-400" },
};

// Service Detail Panel
function ServiceDetailPanel({ node, topology, clusterId, timeRange, t }: {
  node: MeshServiceNode;
  topology: MeshTopologyResponse;
  clusterId: string;
  timeRange: string;
  t: MeshTabTranslations;
}) {
  const nsColor = getNamespaceColor(node.namespace);
  const inbound = topology.edges.filter(e => e.target === node.id);
  const outbound = topology.edges.filter(e => e.source === node.id);
  const [detail, setDetail] = useState<MeshServiceDetailResponse | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // Lazy-load detail data when node changes (保留旧数据避免闪烁)
  useEffect(() => {
    let cancelled = false;
    setDetailLoading(true);
    getMeshServiceDetail({ clusterId, namespace: node.namespace, name: node.name, timeRange })
      .then(res => { if (!cancelled) setDetail(res.data); })
      .catch(() => { if (!cancelled) setDetail(null); })
      .finally(() => { if (!cancelled) setDetailLoading(false); });
    return () => { cancelled = true; };
  }, [node.id, clusterId, node.namespace, node.name, timeRange]);

  const statusCodes = detail?.status_codes?.filter(s => s.count > 0) ?? [];
  const latencyBuckets = detail?.latency_buckets?.filter(b => b.count > 0) ?? [];
  const totalStatusRequests = statusCodes.reduce((sum, s) => sum + s.count, 0);

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-full flex items-center justify-center text-white shadow-md" style={{ backgroundColor: nsColor.fill }}>
          <Server className="w-4 h-4" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-semibold text-default">{node.name}</span>
            <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium ${
              node.status === "healthy" ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" :
              node.status === "warning" ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400" :
              "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
            }`}>
              {node.status === "healthy" ? t.healthy : node.status === "warning" ? t.warning : t.critical}
            </span>
          </div>
          <div className="text-xs text-muted mt-0.5">{node.namespace}</div>
        </div>
      </div>

      {/* Golden Metrics */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-3">
        {[
          { label: t.rps, value: `${node.rps.toFixed(0)}`, unit: "/s" },
          { label: t.p50Latency, value: `${node.p50_latency}`, unit: "ms" },
          { label: t.p95Latency, value: `${node.p95_latency}`, unit: "ms" },
          { label: t.p99Latency, value: `${node.p99_latency}`, unit: "ms" },
          { label: t.errorRate, value: node.error_rate.toFixed(2), unit: "%", color: node.error_rate > 0.5 ? "text-red-500" : "text-emerald-500" },
          { label: t.totalRequests, value: node.total_requests.toLocaleString(), unit: "" },
        ].map((m, i) => (
          <div key={i} className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="text-[10px] text-muted mb-1">{m.label}</div>
            <div className={`text-lg font-bold ${m.color || "text-default"}`}>{m.value}<span className="text-xs font-normal text-muted">{m.unit}</span></div>
          </div>
        ))}
      </div>

      {/* Call Relations */}
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center gap-3">
          <Network className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.callRelation}</span>
        </div>
        <div className="p-4 space-y-3">
          {inbound.length > 0 && (
            <div>
              <div className="text-[10px] text-muted font-medium uppercase tracking-wider mb-2">{t.inbound} ({inbound.length})</div>
              <div className="flex flex-wrap gap-2">
                {inbound.map((edge, idx) => {
                  const srcNode = topology.nodes.find(n => n.id === edge.source);
                  if (!srcNode) return null;
                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: getNamespaceColor(srcNode.namespace).fill }} />
                      <span className="font-medium text-default">{srcNode.name}</span>
                      <ArrowRight className="w-3 h-3 text-slate-400" />
                      <span className="text-muted">{edge.rps.toFixed(0)}/s · {edge.avg_latency}ms</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
          {outbound.length > 0 && (
            <div>
              <div className="text-[10px] text-muted font-medium uppercase tracking-wider mb-2">{t.outbound} ({outbound.length})</div>
              <div className="flex flex-wrap gap-2">
                {outbound.map((edge, idx) => {
                  const tgtNode = topology.nodes.find(n => n.id === edge.target);
                  if (!tgtNode) return null;
                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      <ArrowRight className="w-3 h-3 text-cyan-600" />
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: getNamespaceColor(tgtNode.namespace).fill }} />
                      <span className="font-medium text-default">{tgtNode.name}</span>
                      <span className="text-muted">{edge.rps.toFixed(0)}/s · {edge.avg_latency}ms</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
          {inbound.length === 0 && outbound.length === 0 && (
            <div className="text-xs text-muted text-center py-4">{t.noCallData}</div>
          )}
        </div>
      </div>

      {/* Detail loading indicator */}
      {detailLoading && (
        <div className="flex items-center justify-center py-4 gap-2 text-sm text-muted">
          <Loader2 className="w-4 h-4 animate-spin" />
          {t.loading}
        </div>
      )}

      {/* Status Code + Latency Distribution — side by side */}
      <div className="flex flex-col lg:flex-row gap-4">
          {/* Status Code Distribution */}
          <div className={`bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden lg:w-[40%]`}>
            <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center gap-2">
              <BarChart3 className="w-4 h-4 text-primary flex-shrink-0" />
              <span className="text-sm font-medium text-default truncate">{t.statusCodeBreakdown}</span>
              <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400 flex-shrink-0">Linkerd</span>
            </div>
            {statusCodes.length > 0 ? (
              <div className="p-4 space-y-2">
                <div className="text-[10px] text-muted mb-2">{totalStatusRequests.toLocaleString()} {t.requests} · {timeRangeLabel(timeRange)}</div>
                {statusCodes.map((s) => {
                  const percent = totalStatusRequests > 0 ? (s.count / totalStatusRequests) * 100 : 0;
                  const maxCount = Math.max(...statusCodes.map(sc => sc.count), 1);
                  const barWidth = (s.count / maxCount) * 100;
                  const colors = statusColors[s.code] || statusColors["2xx"];
                  return (
                    <div key={s.code} className="flex items-center gap-2">
                      <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-semibold w-10 text-center ${colors.text} ${colors.bg}`}>{s.code}</span>
                      <div className="flex-1 h-4 bg-[var(--hover-bg)] rounded-sm overflow-hidden">
                        <div className={`h-full rounded-sm ${colors.bar} opacity-80`} style={{ width: `${barWidth}%` }} />
                      </div>
                      <div className="text-right flex items-center gap-1 justify-end flex-shrink-0">
                        <span className="text-xs font-medium text-default">{percent.toFixed(1)}%</span>
                        <span className="text-[10px] text-muted">({s.count.toLocaleString()})</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="p-4 text-center text-xs text-muted py-8">{t.noCallData}</div>
            )}
          </div>

          {/* Latency Distribution Histogram */}
          <div className={`bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden lg:w-[60%]`}>
            <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <BarChart3 className="w-4 h-4 text-primary flex-shrink-0" />
                <span className="text-sm font-medium text-default truncate">{t.latencyDistribution}</span>
                <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400 flex-shrink-0">Linkerd</span>
              </div>
              {latencyBuckets.length > 0 && (
                <div className="flex items-center gap-1.5 flex-shrink-0">
                  <span className="px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/30 rounded text-[9px] text-blue-700 dark:text-blue-400 font-medium">
                    P50 {node.p50_latency}ms
                  </span>
                  <span className="px-1.5 py-0.5 bg-amber-100 dark:bg-amber-900/30 rounded text-[9px] text-amber-700 dark:text-amber-400 font-medium">
                    P95 {node.p95_latency}ms
                  </span>
                  <span className="px-1.5 py-0.5 bg-red-100 dark:bg-red-900/30 rounded text-[9px] text-red-700 dark:text-red-400 font-medium">
                    P99 {node.p99_latency}ms
                  </span>
                </div>
              )}
            </div>
            {latencyBuckets.length > 0 ? (
              <div className="px-4 pt-3 pb-10">
                <MiniLatencyHistogram
                  buckets={latencyBuckets}
                  p50={node.p50_latency}
                  p95={node.p95_latency}
                  p99={node.p99_latency}
                  t={t}
                />
              </div>
            ) : (
              <div className="p-4 text-center text-xs text-muted py-8">{t.noCallData}</div>
            )}
          </div>
        </div>
    </div>
  );
}

// Histogram utilities (Kibana-style fixed axis, log scale)
// 1-2-5 log-scale tick series
const MESH_TICKS = [1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000];
function meshLogPos(ms: number, lo: number, hi: number): number {
  const v = Math.log10(Math.max(ms, 0.1));
  const a = Math.log10(Math.max(lo, 0.1));
  const b = Math.log10(Math.max(hi, 1));
  if (b <= a) return 50;
  return Math.min(100, Math.max(0, ((v - a) / (b - a)) * 100));
}
function meshAxisRange(les: number[]): [number, number] {
  if (les.length === 0) return [1, 1000];
  const minLe = Math.min(...les), maxLe = Math.max(...les);
  let lo = MESH_TICKS[0];
  for (const t of MESH_TICKS) { if (t <= minLe * 0.6) lo = t; else break; }
  let hi = MESH_TICKS[MESH_TICKS.length - 1];
  for (let i = MESH_TICKS.length - 1; i >= 0; i--) { if (MESH_TICKS[i] >= maxLe * 1.4) hi = MESH_TICKS[i]; else break; }
  return [Math.min(lo, minLe * 0.5), Math.max(hi, maxLe * 1.5)];
}
function meshVisibleTicks(lo: number, hi: number): number[] {
  return MESH_TICKS.filter(t => t >= lo && t <= hi);
}
function meshTickLabel(ms: number): string { return ms >= 1000 ? `${ms / 1000}s` : `${ms}ms`; }

// Mini Latency Histogram for service detail (Kibana-style)
function MiniLatencyHistogram({ buckets, p50, p95, p99, t }: {
  buckets: { le: number; count: number }[];
  p50: number;
  p95: number;
  p99: number;
  t: MeshTabTranslations;
}) {
  if (buckets.length === 0) return null;
  const maxCount = Math.max(...buckets.map(b => b.count), 1);
  const [lo, hi] = meshAxisRange(buckets.map(b => b.le));
  const ticks = meshVisibleTicks(lo, hi);

  const bars = buckets.map((b, i) => {
    const prev = i === 0 ? lo : buckets[i - 1].le;
    const left = meshLogPos(prev, lo, hi);
    const right = meshLogPos(b.le, lo, hi);
    const color = b.le > p99
      ? "bg-red-400/80 hover:bg-red-500"
      : b.le > p95 ? "bg-amber-400/90 hover:bg-amber-500"
      : b.le > p50 ? "bg-teal-400/80 hover:bg-teal-500 dark:bg-teal-500/70 dark:hover:bg-teal-400"
      : "bg-blue-400/80 hover:bg-blue-500 dark:bg-blue-500/70 dark:hover:bg-blue-400";
    const prevLe = i === 0 ? 0 : buckets[i - 1].le;
    return { ...b, prevLe, left, width: Math.max(right - left, 0.5), color };
  });

  return (
    <div className="flex gap-2">
      <div className="flex flex-col justify-between h-28 text-[9px] text-muted text-right w-8 flex-shrink-0">
        <span>{maxCount.toLocaleString()}</span>
        <span>{Math.round(maxCount / 2).toLocaleString()}</span>
        <span>0</span>
      </div>
      <div className="relative h-28 flex-1">
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-0 left-0 right-0 border-t border-[var(--border-color)]" />
          <div className="absolute top-1/2 left-0 right-0 border-t border-dashed border-[var(--border-color)]" />
          <div className="absolute bottom-0 left-0 right-0 border-t border-[var(--border-color)]" />
        </div>
        {/* Vertical tick grid */}
        {ticks.map(tick => (
          <div key={tick} className="absolute top-0 bottom-0 w-px bg-[var(--border-color)] opacity-30 pointer-events-none"
            style={{ left: `${meshLogPos(tick, lo, hi)}%` }} />
        ))}
        {/* P50/P95/P99 lines */}
        {[
          { value: p50, c: "blue", label: "P50" },
          { value: p95, c: "amber", label: "P95" },
          { value: p99, c: "red", label: "P99" },
        ].map(({ value, c, label }) => (
          <div key={label} className={`absolute top-0 bottom-0 w-px bg-${c}-500/70 z-20 pointer-events-none`}
            style={{ left: `${meshLogPos(value, lo, hi)}%` }}>
            <div className={`absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-${c}-500 text-white text-[8px] font-medium whitespace-nowrap rounded`}>
              {label}
            </div>
          </div>
        ))}
        {/* Bars — log scale positioning */}
        {bars.map((bar, idx) => {
          const hPct = (bar.count / maxCount) * 100;
          return (
            <div key={idx} className="absolute bottom-0 group z-10"
              style={{ left: `${bar.left}%`, width: `${bar.width}%`, height: "100%" }}>
              <div className="h-full flex items-end px-[0.5px]">
                <div className={`w-full rounded-t-sm transition-all duration-150 ${bar.color}`}
                  style={{ height: `${Math.max(hPct, 2)}%` }} />
              </div>
              <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 hidden group-hover:block z-30 pointer-events-none">
                <div className="bg-slate-900 text-white text-[10px] px-2.5 py-1.5 rounded-md shadow-xl whitespace-nowrap border border-slate-700">
                  <div className="font-medium">{bar.prevLe > 0 ? `${bar.prevLe}–${bar.le}ms` : `0–${bar.le}ms`}</div>
                  <div className="text-slate-300">{bar.count.toLocaleString()} {t.requests}</div>
                </div>
              </div>
            </div>
          );
        })}
        {/* X axis — fixed tick labels */}
        <div className="absolute -bottom-4 left-0 right-0 text-[9px] text-muted">
          {ticks.map(tick => (
            <span key={tick} className="absolute -translate-x-1/2 whitespace-nowrap"
              style={{ left: `${meshLogPos(tick, lo, hi)}%` }}>
              {meshTickLabel(tick)}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}

// Time range display label
function timeRangeLabel(tr: string): string {
  switch (tr) {
    case "1d": return "24h";
    case "7d": return "7d";
    case "30d": return "30d";
    default: return tr;
  }
}

// Main Mesh Tab
export function MeshTab({ topology, clusterId, timeRange, t }: {
  topology: MeshTopologyResponse | null;
  clusterId: string;
  timeRange: string;
  t: MeshTabTranslations;
}) {
  const [selectedServiceId, setSelectedServiceId] = useState<string | null>(null);

  if (!topology || topology.nodes.length === 0) {
    return (
      <div className="text-center py-8 text-sm text-muted">
        <Network className="w-8 h-8 mx-auto mb-2 opacity-50" />
        {t.noCallData}
      </div>
    );
  }

  // 默认选中第一个节点
  const effectiveId = selectedServiceId ?? topology.nodes[0]?.id ?? null;
  const selectedNode = effectiveId ? topology.nodes.find(n => n.id === effectiveId) : null;

  // mTLS coverage
  const totalRps = topology.nodes.reduce((sum, n) => sum + n.rps, 0);
  const overallMtls = totalRps > 0 ? topology.nodes.reduce((sum, n) => sum + n.mtls_percent * n.rps, 0) / totalRps : 0;
  const mtlsBarColor = overallMtls >= 95 ? "bg-emerald-500" : overallMtls >= 80 ? "bg-amber-500" : "bg-red-500";
  const mtlsTextColor = overallMtls >= 95 ? "text-emerald-600 dark:text-emerald-400" : overallMtls >= 80 ? "text-amber-600 dark:text-amber-400" : "text-red-600 dark:text-red-400";

  return (
    <div className="space-y-4">
      {/* Topology Graph */}
      {topology.nodes.length > 1 && (
        <ServiceTopologyView topology={topology} onSelectNode={(node) => setSelectedServiceId(node.id)} timeRange={timeRange} t={t} />
      )}

      {/* Service Mesh Overview (table + detail) */}
      <div className="rounded-xl border border-[var(--border-color)] bg-card overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-primary" />
            <span className="text-sm font-semibold text-default">{t.meshOverview}</span>
            <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400">Linkerd</span>
            <span className="text-[10px] text-muted">{topology.nodes.length} services · {timeRangeLabel(timeRange)}</span>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <Shield className="w-3.5 h-3.5 text-muted" />
              <span className="text-[10px] text-muted">{t.mtls}</span>
              <div className="w-20 h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
                <div className={`h-full rounded-full ${mtlsBarColor}`} style={{ width: `${Math.min(100, overallMtls)}%` }} />
              </div>
              <span className={`text-xs font-semibold ${mtlsTextColor}`}>{overallMtls.toFixed(1)}%</span>
            </div>
          </div>
        </div>
        <div className="flex flex-col lg:flex-row">
          <div className={`${selectedNode ? "lg:w-[400px] lg:border-r border-[var(--border-color)]" : "w-full"} p-4`}>
            <ServiceListTable nodes={topology.nodes} selectedId={effectiveId} onSelect={setSelectedServiceId} t={t} />
          </div>
          {selectedNode && (
            <div className="flex-1 p-4 bg-[var(--background)]">
              <ServiceDetailPanel node={selectedNode} topology={topology} clusterId={clusterId} timeRange={timeRange} t={t} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
