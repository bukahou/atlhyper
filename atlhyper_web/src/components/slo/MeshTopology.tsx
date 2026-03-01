"use client";

import { useState, useMemo, useRef, useEffect } from "react";
import { Network, ZoomIn, ZoomOut, Shrink } from "lucide-react";
import { getNamespaceColor, formatLatency, formatRPS } from "./common";
import type { MeshServiceNode, MeshTopologyResponse } from "@/types/mesh";
import type { MeshTabTranslations } from "./MeshTypes";
import { timeRangeLabel } from "./MeshTypes";
import {
  NODE_RADIUS, SVG_HEIGHT, MIN_ZOOM, MAX_ZOOM,
  computeNodeLayers, groupNodesByNamespace, computeSwimLaneLayout,
  computeFitZoom, getEdgePath, getEdgeLabelPos,
} from "./MeshTopologyUtils";

// Service Topology View (SVG) with zoom & pan
export function ServiceTopologyView({ topology, onSelectNode, timeRange, t }: {
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

  const nsGroups = useMemo(() => groupNodesByNamespace(topology), [topology]);
  const sortedNamespaces = useMemo(() => Object.keys(nsGroups).sort(), [nsGroups]);
  const [nsLaneBounds, setNsLaneBounds] = useState<Record<string, { minX: number; minY: number; maxX: number; maxY: number }>>({});

  useEffect(() => {
    const updateWidth = () => { if (containerRef.current) setContainerWidth(containerRef.current.offsetWidth); };
    updateWidth();
    window.addEventListener("resize", updateWidth);
    return () => window.removeEventListener("resize", updateWidth);
  }, []);

  const nodeLayers = useMemo(() => computeNodeLayers(topology), [topology]);

  // Initialize swim-lane layout
  useEffect(() => {
    if (initialized || containerWidth < 100) return;
    const layout = computeSwimLaneLayout(containerWidth, nodeLayers, nsGroups, sortedNamespaces);
    setPositions(layout.positions);
    setNsLaneBounds(layout.bounds);

    const fit = computeFitZoom(layout.positions, containerWidth);
    if (fit) {
      setZoom(fit.zoom);
      setViewOrigin(fit.viewOrigin);
    }
    setInitialized(true);
  }, [containerWidth, nodeLayers, nsGroups, sortedNamespaces, initialized]);

  const viewBox = `${viewOrigin.x} ${viewOrigin.y} ${containerWidth / zoom} ${SVG_HEIGHT / zoom}`;

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
      const cy = SVG_HEIGHT / 2;
      setViewOrigin(vo => ({
        x: vo.x + cx * (1 / prev - 1 / newZoom),
        y: vo.y + cy * (1 / prev - 1 / newZoom),
      }));
      return newZoom;
    });
  };

  const fitToContent = () => {
    const fit = computeFitZoom(positions, containerWidth);
    if (fit) { setZoom(fit.zoom); setViewOrigin(fit.viewOrigin); }
    else { setZoom(1); setViewOrigin({ x: 0, y: 0 }); }
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

        <svg ref={svgRef} width={containerWidth} height={SVG_HEIGHT} viewBox={viewBox}
          className={`select-none ${isPanning ? "cursor-grabbing" : "cursor-grab"}`}
          onMouseDown={handleSvgMouseDown} onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp} onMouseLeave={handleMouseUp}>
          <defs>
            <marker id="arrow-gray" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#94a3b8" /></marker>
            <marker id="arrow-blue" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#0891b2" /></marker>
            <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%"><feDropShadow dx="0" dy="2" stdDeviation="3" floodOpacity="0.2" /></filter>
          </defs>
          <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#e2e8f0" strokeWidth="0.5" className="dark:stroke-slate-700" /></pattern>
          <rect x={viewOrigin.x} y={viewOrigin.y} width={containerWidth / zoom} height={SVG_HEIGHT / zoom} fill="url(#grid)" opacity="0.5" />

          {/* Namespace labels */}
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
            if (edge.errorRate > 1) strokeColor = isHighlighted ? "#ef4444" : "#fca5a5";
            else if (edge.errorRate > 0.1) strokeColor = isHighlighted ? "#f59e0b" : "#fcd34d";
            else if (isHighlighted) strokeColor = "#0ea5e9";
            const labelPos = getEdgeLabelPos(edge.source, edge.target, positions);
            return (
              <g key={idx} onMouseEnter={() => setHoveredEdge(idx)} onMouseLeave={() => setHoveredEdge(null)}>
                <path d={getEdgePath(edge.source, edge.target, positions)} fill="none" stroke="transparent" strokeWidth={12} style={{ cursor: "pointer" }} />
                <path d={getEdgePath(edge.source, edge.target, positions)} fill="none" stroke={strokeColor} strokeWidth={isHighlighted ? 3 : 2} markerEnd={`url(#${isHighlighted ? "arrow-blue" : "arrow-gray"})`} className="transition-colors duration-200" />
                {isHighlighted && (
                  <g transform={`translate(${labelPos.x}, ${labelPos.y})`}>
                    <rect x="-42" y="-12" width="84" height="24" rx="4" fill="white" className="dark:fill-slate-800" stroke="#e2e8f0" strokeWidth="1" />
                    <text textAnchor="middle" y="4" className="text-[10px] font-medium fill-slate-600 dark:fill-slate-300">{formatRPS(edge.rps)}/s · avg {formatLatency(edge.avgLatency)}</text>
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
                {(isSelected || isHovered) && <circle r={NODE_RADIUS + 5} fill="none" stroke={colors.light} strokeWidth={2} strokeOpacity={isSelected ? 0.8 : 0.5} />}
                <circle r={NODE_RADIUS} fill={colors.fill} stroke={colors.stroke} strokeWidth={2} filter="url(#shadow)" />
                <g style={{ transform: "translate(-10px, -10px)" }}>
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <rect width="20" height="8" x="2" y="2" rx="2" ry="2"/><rect width="20" height="8" x="2" y="14" rx="2" ry="2"/>
                    <line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/>
                  </svg>
                </g>
                <text y={NODE_RADIUS + 16} textAnchor="middle" className="text-[11px] font-semibold fill-slate-700 dark:fill-slate-200 pointer-events-none">
                  {node.name.length > 14 ? node.name.slice(0, 14) + "\u2026" : node.name}
                </text>
                <text y={NODE_RADIUS + 28} textAnchor="middle" className="text-[9px] fill-slate-500 dark:fill-slate-400 pointer-events-none">
                  P95 {formatLatency(node.p95Latency)} · {node.errorRate.toFixed(1)}%
                </text>
                <circle cx={NODE_RADIUS - 4} cy={-NODE_RADIUS + 4} r={5}
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
