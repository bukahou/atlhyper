"use client";

import { useRef, useEffect, useState, useCallback, useMemo } from "react";
import type { DependencyGraph, EntityRisk } from "@/api/aiops";

// 节点颜色按风险等级
function riskColor(rFinal: number): string {
  if (rFinal >= 80) return "#ef4444";
  if (rFinal >= 50) return "#f97316";
  if (rFinal >= 30) return "#eab308";
  if (rFinal >= 10) return "#3b82f6";
  return "#22c55e";
}

// 节点形状按类型
const TYPE_LABELS: Record<string, string> = {
  service: "S",
  pod: "P",
  node: "N",
  ingress: "I",
};

interface LayoutNode {
  key: string;
  type: string;
  name: string;
  namespace: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  color: string;
  radius: number;
}

interface TopologyGraphProps {
  graph: DependencyGraph;
  entityRisks: Record<string, EntityRisk>;
  selectedNode: string | null;
  onNodeSelect: (key: string) => void;
}

export function TopologyGraph({ graph, entityRisks, selectedNode, onNodeSelect }: TopologyGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [nodes, setNodes] = useState<LayoutNode[]>([]);
  const [viewBox, setViewBox] = useState({ x: -400, y: -300, w: 800, h: 600 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [dragNode, setDragNode] = useState<string | null>(null);

  // 初始化节点位置
  const initialNodes = useMemo(() => {
    const keys = Object.keys(graph.nodes);
    const result: LayoutNode[] = [];
    const typeGroups: Record<string, number> = { ingress: -200, service: -70, pod: 60, node: 200 };

    let idx = 0;
    for (const key of keys) {
      const n = graph.nodes[key];
      const typeY = typeGroups[n.type] ?? 0;
      const risk = entityRisks[key];
      const rFinal = risk?.rFinal ?? 0;

      result.push({
        key,
        type: n.type,
        name: n.name,
        namespace: n.namespace,
        x: (idx % 6) * 120 - 300 + (Math.random() - 0.5) * 40,
        y: typeY + (Math.random() - 0.5) * 60,
        vx: 0,
        vy: 0,
        color: riskColor(rFinal),
        radius: n.type === "node" ? 22 : n.type === "service" ? 18 : 14,
      });
      idx++;
    }
    return result;
  }, [graph.nodes, entityRisks]);

  // 简单力导向模拟
  useEffect(() => {
    const layoutNodes = initialNodes.map((n) => ({ ...n }));
    const nodeMap = new Map(layoutNodes.map((n) => [n.key, n]));

    let animFrame: number;
    let iteration = 0;
    const maxIterations = 120;

    function simulate() {
      if (iteration >= maxIterations) {
        setNodes([...layoutNodes]);
        return;
      }

      const alpha = 0.3 * (1 - iteration / maxIterations);

      // 斥力
      for (let i = 0; i < layoutNodes.length; i++) {
        for (let j = i + 1; j < layoutNodes.length; j++) {
          const a = layoutNodes[i];
          const b = layoutNodes[j];
          const dx = b.x - a.x;
          const dy = b.y - a.y;
          const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
          const force = (3000 / (dist * dist)) * alpha;
          const fx = (dx / dist) * force;
          const fy = (dy / dist) * force;
          a.vx -= fx;
          a.vy -= fy;
          b.vx += fx;
          b.vy += fy;
        }
      }

      // 引力（边连接的节点）
      for (const edge of graph.edges) {
        const a = nodeMap.get(edge.from);
        const b = nodeMap.get(edge.to);
        if (!a || !b) continue;
        const dx = b.x - a.x;
        const dy = b.y - a.y;
        const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
        const force = (dist - 120) * 0.01 * alpha;
        const fx = (dx / dist) * force;
        const fy = (dy / dist) * force;
        a.vx += fx;
        a.vy += fy;
        b.vx -= fx;
        b.vy -= fy;
      }

      // 应用速度 + 阻尼
      for (const n of layoutNodes) {
        n.x += n.vx;
        n.y += n.vy;
        n.vx *= 0.6;
        n.vy *= 0.6;
      }

      iteration++;
      setNodes([...layoutNodes]);
      animFrame = requestAnimationFrame(simulate);
    }

    simulate();
    return () => cancelAnimationFrame(animFrame);
  }, [initialNodes, graph.edges]);

  // 节点映射
  const nodeMap = useMemo(() => new Map(nodes.map((n) => [n.key, n])), [nodes]);

  // 鼠标拖拽画布
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if ((e.target as SVGElement).closest(".graph-node")) return;
    setIsDragging(true);
    setDragStart({ x: e.clientX, y: e.clientY });
  }, []);

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (dragNode) {
        // 拖拽节点
        const svg = svgRef.current;
        if (!svg) return;
        const pt = svg.createSVGPoint();
        pt.x = e.clientX;
        pt.y = e.clientY;
        const ctm = svg.getScreenCTM();
        if (!ctm) return;
        const svgP = pt.matrixTransform(ctm.inverse());
        setNodes((prev) =>
          prev.map((n) => (n.key === dragNode ? { ...n, x: svgP.x, y: svgP.y, vx: 0, vy: 0 } : n))
        );
        return;
      }

      if (!isDragging) return;
      const dx = e.clientX - dragStart.x;
      const dy = e.clientY - dragStart.y;
      setViewBox((prev) => ({
        ...prev,
        x: prev.x - dx * (prev.w / 800),
        y: prev.y - dy * (prev.h / 600),
      }));
      setDragStart({ x: e.clientX, y: e.clientY });
    },
    [isDragging, dragStart, dragNode]
  );

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
    setDragNode(null);
  }, []);

  // 缩放
  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault();
    const scale = e.deltaY > 0 ? 1.1 : 0.9;
    setViewBox((prev) => {
      const newW = prev.w * scale;
      const newH = prev.h * scale;
      return {
        x: prev.x + (prev.w - newW) / 2,
        y: prev.y + (prev.h - newH) / 2,
        w: newW,
        h: newH,
      };
    });
  }, []);

  return (
    <svg
      ref={svgRef}
      viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.w} ${viewBox.h}`}
      className="w-full h-full bg-[var(--background)] rounded-xl border border-[var(--border-color)] cursor-grab active:cursor-grabbing"
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
      onWheel={handleWheel}
    >
      {/* 箭头标记 */}
      <defs>
        <marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto">
          <path d="M 0 0 L 10 5 L 0 10 z" fill="var(--border-color)" />
        </marker>
        <marker id="arrow-red" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto">
          <path d="M 0 0 L 10 5 L 0 10 z" fill="#ef4444" />
        </marker>
      </defs>

      {/* 边 */}
      {graph.edges.map((edge, i) => {
        const from = nodeMap.get(edge.from);
        const to = nodeMap.get(edge.to);
        if (!from || !to) return null;

        const dx = to.x - from.x;
        const dy = to.y - from.y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const nx = dx / dist;
        const ny = dy / dist;

        const x1 = from.x + nx * from.radius;
        const y1 = from.y + ny * from.radius;
        const x2 = to.x - nx * to.radius;
        const y2 = to.y - ny * to.radius;

        const isAnomaly = edge.type === "calls" && (entityRisks[edge.from]?.rFinal ?? 0) > 50;

        return (
          <line
            key={i}
            x1={x1}
            y1={y1}
            x2={x2}
            y2={y2}
            stroke={isAnomaly ? "#ef4444" : "var(--border-color)"}
            strokeWidth={isAnomaly ? 1.5 : 0.8}
            strokeOpacity={isAnomaly ? 0.7 : 0.4}
            markerEnd={isAnomaly ? "url(#arrow-red)" : "url(#arrow)"}
          />
        );
      })}

      {/* 节点 */}
      {nodes.map((node) => {
        const isSelected = selectedNode === node.key;
        return (
          <g
            key={node.key}
            className="graph-node cursor-pointer"
            transform={`translate(${node.x}, ${node.y})`}
            onClick={() => onNodeSelect(node.key)}
            onMouseDown={(e) => {
              e.stopPropagation();
              setDragNode(node.key);
            }}
          >
            {/* 选中光晕 */}
            {isSelected && <circle r={node.radius + 6} fill="none" stroke={node.color} strokeWidth={2} opacity={0.5} />}

            {/* 节点形状 */}
            {node.type === "node" ? (
              // 六边形
              <polygon
                points={hexPoints(node.radius)}
                fill={node.color}
                fillOpacity={0.15}
                stroke={node.color}
                strokeWidth={isSelected ? 2 : 1}
              />
            ) : node.type === "ingress" ? (
              // 菱形
              <polygon
                points={`0,${-node.radius} ${node.radius},0 0,${node.radius} ${-node.radius},0`}
                fill={node.color}
                fillOpacity={0.15}
                stroke={node.color}
                strokeWidth={isSelected ? 2 : 1}
              />
            ) : node.type === "service" ? (
              // 圆形
              <circle r={node.radius} fill={node.color} fillOpacity={0.15} stroke={node.color} strokeWidth={isSelected ? 2 : 1} />
            ) : (
              // 方形 (pod)
              <rect
                x={-node.radius}
                y={-node.radius}
                width={node.radius * 2}
                height={node.radius * 2}
                rx={3}
                fill={node.color}
                fillOpacity={0.15}
                stroke={node.color}
                strokeWidth={isSelected ? 2 : 1}
              />
            )}

            {/* 类型标签 */}
            <text textAnchor="middle" dy={4} fontSize={10} fontWeight="bold" fill={node.color}>
              {TYPE_LABELS[node.type] ?? "?"}
            </text>

            {/* 名称 */}
            <text textAnchor="middle" dy={node.radius + 12} fontSize={7} fill="var(--text-muted)" className="select-none">
              {node.name.length > 16 ? node.name.slice(0, 14) + ".." : node.name}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

function hexPoints(r: number): string {
  const pts = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i - Math.PI / 6;
    pts.push(`${r * Math.cos(angle)},${r * Math.sin(angle)}`);
  }
  return pts.join(" ");
}
