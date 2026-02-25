"use client";

import { useState, useMemo, useRef, useEffect } from "react";
import { Layout } from "@/components/layout/Layout";
import {
  Activity,
  AlertTriangle,
  TrendingUp,
  TrendingDown,
  Minus,
  ChevronDown,
  ChevronRight,
  RefreshCw,
  Settings2,
  Globe,
  Zap,
  Gauge,
  X,
  ArrowUpRight,
  ArrowDownRight,
  Calendar,
  Target,
  Network,
  Clock,
  ArrowRight,
  Server,
  BarChart3,
  Layers,
  Shield,
} from "lucide-react";

// ==================== Types ====================

// ---- 延迟分布（来自 Linkerd/Traefik histogram buckets）----
interface LatencyBucket {
  le: number;             // upper bound (ms)
  count: number;          // requests in this bucket
}

interface RequestBreakdown {
  method: string;         // GET, POST, PUT, DELETE
  count: number;
  errorCount: number;
}

interface StatusCodeBreakdown {
  code: string;           // "2xx", "3xx", "4xx", "5xx"
  count: number;
}

// ---- 服务拓扑（来自 Linkerd） ----
interface ServiceNode {
  id: string;
  name: string;           // deployment name
  namespace: string;
  rps: number;            // requests per second
  avgLatency: number;     // ms (latency_sum / latency_count)
  p50Latency: number;     // ms
  p95Latency: number;     // ms (histogram interpolation)
  p99Latency: number;     // ms
  errorRate: number;      // % (failure / total * 100)
  status: "healthy" | "warning" | "critical";
  mtlsEnabled: boolean;           // mTLS 是否启用
  latencyDistribution: LatencyBucket[];      // Linkerd 24桶
  requestBreakdown: RequestBreakdown[];
  statusCodeBreakdown: StatusCodeBreakdown[];
  totalRequests: number;
}

interface ServiceEdge {
  source: string;         // node id
  target: string;         // node id
  rps: number;            // requests/s
  avgLatency: number;     // ms
  errorRate: number;      // %
}

interface ServiceTopology {
  nodes: ServiceNode[];
  edges: ServiceEdge[];
}

// ---- 域名 SLO（来自 Traefik + Routes）----
type TimeRange = "1d" | "7d" | "30d";

interface SLOTargets {
  availability: number;
  p95Latency: number;
  errorRate: number;
}

interface HistoryPoint {
  timestamp: string;
  availability: number;
  p95Latency: number;
  p99Latency: number;
  errorRate: number;
  rps: number;
  errorBudgetRemaining: number;
}

interface DomainSLO {
  id: string;
  host: string;
  ingressName: string;
  namespace: string;
  tls: boolean;
  targets: Record<TimeRange, SLOTargets>;
  current: {
    availability: number;
    p95Latency: number;
    p50Latency: number;
    p99Latency: number;
    errorRate: number;
    requestsPerSec: number;
    totalRequests: number;
  };
  previous: {
    availability: number;
    p95Latency: number;
    errorRate: number;
  };
  errorBudgetRemaining: number;
  status: "healthy" | "warning" | "critical";
  trend: "up" | "down" | "stable";
  history: HistoryPoint[];
  latencyDistribution: LatencyBucket[];
  requestBreakdown: RequestBreakdown[];
  statusCodeBreakdown: StatusCodeBreakdown[];
  backendServices: string[];              // 关联的服务 node ID 列表
}

// ==================== Namespace Colors ====================

const namespaceColors: Record<string, { fill: string; stroke: string; light: string }> = {
  "kube-system": { fill: "#7c3aed", stroke: "#6d28d9", light: "#a78bfa" },
  "geass":       { fill: "#0891b2", stroke: "#0e7490", light: "#22d3ee" },
  "elastic":     { fill: "#d97706", stroke: "#b45309", light: "#fbbf24" },
  "atlhyper":    { fill: "#059669", stroke: "#047857", light: "#34d399" },
  "default":     { fill: "#4b5563", stroke: "#374151", light: "#9ca3af" },
};

function getNamespaceColor(ns: string) {
  return namespaceColors[ns] || namespaceColors["default"];
}

// ==================== Mock Data ====================

// Linkerd 24 桶边界 (ms)
const linkerdBucketBounds = [1, 2, 3, 4, 5, 10, 20, 30, 40, 50, 100, 200, 300, 400, 500, 1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000];

// 生成延迟分布 mock（正态偏右分布）
function generateLatencyDistribution(peakMs: number): LatencyBucket[] {
  return linkerdBucketBounds.map((le) => {
    const ratio = le / peakMs;
    let count: number;
    if (ratio < 0.3) count = Math.floor(Math.random() * 50 + 10);
    else if (ratio < 0.7) count = Math.floor(Math.random() * 200 + 150);
    else if (ratio < 1.0) count = Math.floor(Math.random() * 500 + 400);
    else if (ratio < 1.5) count = Math.floor(Math.random() * 400 + 200);
    else if (ratio < 2.5) count = Math.floor(Math.random() * 150 + 50);
    else if (ratio < 5.0) count = Math.floor(Math.random() * 40 + 5);
    else count = Math.floor(Math.random() * 10);
    return { le, count };
  });
}

// 全局拓扑
const mockGlobalTopology: ServiceTopology = {
  nodes: [
    { id: "traefik",        name: "traefik",        namespace: "kube-system", rps: 5200, avgLatency: 3,   p50Latency: 2,   p95Latency: 8,   p99Latency: 15,  errorRate: 0.02, status: "healthy",  mtlsEnabled: true,  latencyDistribution: generateLatencyDistribution(5),   requestBreakdown: [{ method: "GET", count: 38000, errorCount: 8 }, { method: "POST", count: 12000, errorCount: 3 }], statusCodeBreakdown: [{ code: "2xx", count: 48500 }, { code: "3xx", count: 800 }, { code: "4xx", count: 600 }, { code: "5xx", count: 100 }], totalRequests: 50000 },
    { id: "geass-gateway",  name: "geass-gateway",  namespace: "geass",       rps: 1240, avgLatency: 18,  p50Latency: 12,  p95Latency: 45,  p99Latency: 88,  errorRate: 0.12, status: "healthy",  mtlsEnabled: true, latencyDistribution: generateLatencyDistribution(25),  requestBreakdown: [{ method: "GET", count: 8200, errorCount: 5 }, { method: "POST", count: 3100, errorCount: 4 }, { method: "PUT", count: 680, errorCount: 1 }], statusCodeBreakdown: [{ code: "2xx", count: 11200 }, { code: "3xx", count: 280 }, { code: "4xx", count: 420 }, { code: "5xx", count: 80 }], totalRequests: 11980 },
    { id: "geass-auth",     name: "geass-auth",     namespace: "geass",       rps: 890,  avgLatency: 8,   p50Latency: 5,   p95Latency: 12,  p99Latency: 25,  errorRate: 0.05, status: "healthy",  mtlsEnabled: true, latencyDistribution: generateLatencyDistribution(10),  requestBreakdown: [{ method: "POST", count: 7500, errorCount: 4 }, { method: "GET", count: 1200, errorCount: 0 }], statusCodeBreakdown: [{ code: "2xx", count: 8300 }, { code: "4xx", count: 350 }, { code: "5xx", count: 50 }], totalRequests: 8700 },
    { id: "geass-user",     name: "geass-user",     namespace: "geass",       rps: 560,  avgLatency: 85,  p50Latency: 60,  p95Latency: 180, p99Latency: 350, errorRate: 1.2,  status: "warning",  mtlsEnabled: true,  latencyDistribution: generateLatencyDistribution(100), requestBreakdown: [{ method: "GET", count: 3500, errorCount: 25 }, { method: "POST", count: 1200, errorCount: 18 }, { method: "PUT", count: 400, errorCount: 5 }, { method: "DELETE", count: 100, errorCount: 2 }], statusCodeBreakdown: [{ code: "2xx", count: 4900 }, { code: "3xx", count: 80 }, { code: "4xx", count: 160 }, { code: "5xx", count: 60 }], totalRequests: 5200 },
    { id: "geass-web",      name: "geass-web",      namespace: "geass",       rps: 3200, avgLatency: 12,  p50Latency: 8,   p95Latency: 35,  p99Latency: 65,  errorRate: 0.03, status: "healthy",  mtlsEnabled: true, latencyDistribution: generateLatencyDistribution(15),  requestBreakdown: [{ method: "GET", count: 28000, errorCount: 5 }, { method: "POST", count: 2800, errorCount: 3 }], statusCodeBreakdown: [{ code: "2xx", count: 29800 }, { code: "3xx", count: 600 }, { code: "4xx", count: 350 }, { code: "5xx", count: 50 }], totalRequests: 30800 },
    { id: "geass-media",    name: "geass-media",    namespace: "geass",       rps: 420,  avgLatency: 45,  p50Latency: 30,  p95Latency: 120, p99Latency: 230, errorRate: 0.15, status: "healthy",  mtlsEnabled: true,  latencyDistribution: generateLatencyDistribution(60),  requestBreakdown: [{ method: "GET", count: 3200, errorCount: 3 }, { method: "POST", count: 850, errorCount: 2 }], statusCodeBreakdown: [{ code: "2xx", count: 3800 }, { code: "3xx", count: 120 }, { code: "4xx", count: 100 }, { code: "5xx", count: 30 }], totalRequests: 4050 },
    { id: "elasticsearch",  name: "elasticsearch",  namespace: "elastic",     rps: 340,  avgLatency: 22,  p50Latency: 15,  p95Latency: 35,  p99Latency: 72,  errorRate: 0.02, status: "healthy",  mtlsEnabled: true,  latencyDistribution: generateLatencyDistribution(28),  requestBreakdown: [{ method: "GET", count: 2400, errorCount: 1 }, { method: "POST", count: 800, errorCount: 0 }], statusCodeBreakdown: [{ code: "2xx", count: 3100 }, { code: "4xx", count: 80 }, { code: "5xx", count: 20 }], totalRequests: 3200 },
    { id: "atlhyper-web",   name: "atlhyper-web",   namespace: "atlhyper",    rps: 150,  avgLatency: 15,  p50Latency: 10,  p95Latency: 42,  p99Latency: 80,  errorRate: 0.01, status: "healthy",  mtlsEnabled: true, latencyDistribution: generateLatencyDistribution(20),  requestBreakdown: [{ method: "GET", count: 1200, errorCount: 0 }, { method: "POST", count: 250, errorCount: 1 }], statusCodeBreakdown: [{ code: "2xx", count: 1380 }, { code: "3xx", count: 40 }, { code: "4xx", count: 25 }, { code: "5xx", count: 5 }], totalRequests: 1450 },
  ],
  edges: [
    { source: "traefik",       target: "geass-gateway", rps: 1240, avgLatency: 3,  errorRate: 0.02 },
    { source: "traefik",       target: "geass-web",     rps: 3200, avgLatency: 2,  errorRate: 0.01 },
    { source: "traefik",       target: "atlhyper-web",  rps: 150,  avgLatency: 3,  errorRate: 0.01 },
    { source: "geass-gateway", target: "geass-auth",    rps: 890,  avgLatency: 8,  errorRate: 0.05 },
    { source: "geass-gateway", target: "geass-user",    rps: 560,  avgLatency: 18, errorRate: 0.12 },
    { source: "geass-gateway", target: "geass-media",   rps: 420,  avgLatency: 15, errorRate: 0.08 },
    { source: "geass-user",    target: "elasticsearch", rps: 340,  avgLatency: 22, errorRate: 0.02 },
  ],
};

// 生成历史数据
function generateHistory(days: number, baseAvail: number, baseLatency: number): HistoryPoint[] {
  const points: HistoryPoint[] = [];
  const now = new Date();
  for (let i = days * 24; i >= 0; i -= 4) {
    const timestamp = new Date(now.getTime() - i * 60 * 60 * 1000).toISOString();
    const noise = Math.random() * 0.3 - 0.15;
    const latencyNoise = Math.random() * 50 - 25;
    points.push({
      timestamp,
      availability: Math.min(100, Math.max(95, baseAvail + noise)),
      p95Latency: Math.max(10, baseLatency + latencyNoise),
      p99Latency: Math.max(20, baseLatency * 1.8 + latencyNoise * 2),
      errorRate: Math.max(0, (100 - baseAvail - noise) * 0.8),
      rps: Math.floor(Math.random() * 500 + 1500),
      errorBudgetRemaining: Math.max(0, 70 + Math.random() * 30 - i * 0.02),
    });
  }
  return points;
}

const mockDomainSLOs: DomainSLO[] = [
  {
    id: "1",
    host: "geass.example.com",
    ingressName: "geass-gateway-ingress",
    namespace: "geass",
    tls: true,
    targets: {
      "1d": { availability: 95, p95Latency: 300, errorRate: 5 },
      "7d": { availability: 96, p95Latency: 280, errorRate: 4 },
      "30d": { availability: 97, p95Latency: 250, errorRate: 3 },
    },
    current: { availability: 99.74, p50Latency: 85, p95Latency: 226, p99Latency: 520, errorRate: 0.26, requestsPerSec: 1240, totalRequests: 53700000 },
    previous: { availability: 99.82, p95Latency: 198, errorRate: 0.18 },
    errorBudgetRemaining: 65,
    status: "warning",
    trend: "stable",
    history: generateHistory(7, 99.74, 226),
    latencyDistribution: generateLatencyDistribution(200),
    requestBreakdown: [
      { method: "GET",    count: 12340, errorCount: 15 },
      { method: "POST",   count: 2150,  errorCount: 8 },
      { method: "PUT",    count: 890,   errorCount: 3 },
      { method: "DELETE", count: 320,   errorCount: 6 },
    ],
    statusCodeBreakdown: [
      { code: "2xx", count: 14820 },
      { code: "3xx", count: 330 },
      { code: "4xx", count: 488 },
      { code: "5xx", count: 94 },
    ],
    backendServices: ["geass-gateway", "geass-auth", "geass-user", "geass-media"],
  },
  {
    id: "2",
    host: "monitor.example.com",
    ingressName: "atlhyper-web-ingress",
    namespace: "atlhyper",
    tls: true,
    targets: {
      "1d": { availability: 99, p95Latency: 100, errorRate: 1 },
      "7d": { availability: 99.5, p95Latency: 100, errorRate: 0.5 },
      "30d": { availability: 99.9, p95Latency: 100, errorRate: 0.1 },
    },
    current: { availability: 99.92, p50Latency: 18, p95Latency: 42, p99Latency: 85, errorRate: 0.08, requestsPerSec: 150, totalRequests: 6500000 },
    previous: { availability: 99.96, p95Latency: 38, errorRate: 0.04 },
    errorBudgetRemaining: 12,
    status: "critical",
    trend: "down",
    history: generateHistory(7, 99.92, 42),
    latencyDistribution: generateLatencyDistribution(30),
    requestBreakdown: [
      { method: "GET",  count: 5800, errorCount: 3 },
      { method: "POST", count: 450,  errorCount: 2 },
    ],
    statusCodeBreakdown: [
      { code: "2xx", count: 5920 },
      { code: "3xx", count: 180 },
      { code: "4xx", count: 120 },
      { code: "5xx", count: 30 },
    ],
    backendServices: ["atlhyper-web"],
  },
  {
    id: "3",
    host: "media.example.com",
    ingressName: "geass-media-ingress",
    namespace: "geass",
    tls: true,
    targets: {
      "1d": { availability: 95, p95Latency: 500, errorRate: 5 },
      "7d": { availability: 96, p95Latency: 450, errorRate: 4 },
      "30d": { availability: 97, p95Latency: 400, errorRate: 3 },
    },
    current: { availability: 99.95, p50Latency: 110, p95Latency: 292, p99Latency: 546, errorRate: 0.05, requestsPerSec: 3620, totalRequests: 156800000 },
    previous: { availability: 99.91, p95Latency: 310, errorRate: 0.09 },
    errorBudgetRemaining: 85,
    status: "healthy",
    trend: "up",
    history: generateHistory(7, 99.95, 292),
    latencyDistribution: generateLatencyDistribution(250),
    requestBreakdown: [
      { method: "GET",    count: 28500, errorCount: 8 },
      { method: "POST",   count: 4200,  errorCount: 5 },
      { method: "PUT",    count: 1100,  errorCount: 1 },
      { method: "DELETE", count: 200,   errorCount: 0 },
    ],
    statusCodeBreakdown: [
      { code: "2xx", count: 32100 },
      { code: "3xx", count: 850 },
      { code: "4xx", count: 920 },
      { code: "5xx", count: 130 },
    ],
    backendServices: ["geass-web", "geass-media"],
  },
];

// ==================== Components ====================

// 节点位置状态
interface NodePosition {
  id: string;
  x: number;
  y: number;
  node: ServiceNode;
}

// 服务拓扑图组件（namespace 着色 + 统一 Server 图标）
function ServiceTopologyView({ topology, onSelectNode }: { topology: ServiceTopology; onSelectNode?: (node: ServiceNode) => void }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [hoveredEdge, setHoveredEdge] = useState<number | null>(null);
  const [draggingNode, setDraggingNode] = useState<string | null>(null);
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
  const [containerWidth, setContainerWidth] = useState(1000);

  const nodeRadius = 32;
  const svgHeight = 500;

  useEffect(() => {
    const updateWidth = () => {
      if (containerRef.current) {
        setContainerWidth(containerRef.current.offsetWidth);
      }
    };
    updateWidth();
    window.addEventListener("resize", updateWidth);
    return () => window.removeEventListener("resize", updateWidth);
  }, []);

  // 拓扑排序计算节点层级
  const nodeLayers = useMemo(() => {
    const nodeIds = topology.nodes.map(n => n.id);
    const inDegree: Record<string, number> = {};
    const outEdges: Record<string, string[]> = {};

    nodeIds.forEach(id => {
      inDegree[id] = 0;
      outEdges[id] = [];
    });

    topology.edges.forEach(edge => {
      if (nodeIds.includes(edge.source) && nodeIds.includes(edge.target)) {
        inDegree[edge.target]++;
        outEdges[edge.source].push(edge.target);
      }
    });

    const level: Record<string, number> = {};
    const queue: string[] = [];

    nodeIds.forEach(id => {
      if (inDegree[id] === 0) {
        level[id] = 0;
        queue.push(id);
      }
    });

    while (queue.length > 0) {
      const current = queue.shift()!;
      outEdges[current].forEach(next => {
        const newLevel = level[current] + 1;
        if (level[next] === undefined || level[next] < newLevel) {
          level[next] = newLevel;
        }
        inDegree[next]--;
        if (inDegree[next] === 0) {
          queue.push(next);
        }
      });
    }

    nodeIds.forEach(id => {
      if (level[id] === undefined) level[id] = 0;
    });

    const maxLevel = Math.max(...Object.values(level));
    const layers: string[][] = Array.from({ length: maxLevel + 1 }, () => []);
    nodeIds.forEach(id => {
      layers[level[id]].push(id);
    });

    return layers;
  }, [topology]);

  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({});
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (initialized || containerWidth < 100) return;

    const paddingX = 100;
    const usableWidth = containerWidth - paddingX * 2;
    const layerCount = nodeLayers.length;
    const layerGap = layerCount > 1 ? usableWidth / (layerCount - 1) : 0;
    const nodeGap = 95;

    const pos: Record<string, { x: number; y: number }> = {};

    nodeLayers.forEach((layer, layerIndex) => {
      const layerX = layerCount > 1 ? paddingX + layerIndex * layerGap : containerWidth / 2;
      const layerHeight = (layer.length - 1) * nodeGap;
      const startY = (svgHeight - layerHeight) / 2;

      layer.forEach((nodeId, nodeIndex) => {
        pos[nodeId] = { x: layerX, y: startY + nodeIndex * nodeGap };
      });
    });

    setPositions(pos);
    setInitialized(true);
  }, [containerWidth, nodeLayers, initialized]);

  // 贝塞尔曲线
  const getEdgePath = (sourceId: string, targetId: string) => {
    const source = positions[sourceId];
    const target = positions[targetId];
    if (!source || !target) return "";

    const startX = source.x + nodeRadius;
    const startY = source.y;
    const endX = target.x - nodeRadius;
    const endY = target.y;
    const midX = (startX + endX) / 2;

    return `M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`;
  };

  const getEdgeLabelPos = (sourceId: string, targetId: string) => {
    const source = positions[sourceId];
    const target = positions[targetId];
    if (!source || !target) return { x: 0, y: 0 };
    return { x: (source.x + target.x) / 2, y: (source.y + target.y) / 2 };
  };

  // 拖拽
  const handleMouseDown = (e: React.MouseEvent, nodeId: string) => {
    if (e.button !== 0) return;
    e.stopPropagation();
    const svg = svgRef.current;
    if (!svg) return;
    const pt = svg.createSVGPoint();
    pt.x = e.clientX;
    pt.y = e.clientY;
    const svgP = pt.matrixTransform(svg.getScreenCTM()?.inverse());
    const pos = positions[nodeId];
    setDragOffset({ x: svgP.x - pos.x, y: svgP.y - pos.y });
    setDraggingNode(nodeId);
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!draggingNode) return;
    const svg = svgRef.current;
    if (!svg) return;
    const pt = svg.createSVGPoint();
    pt.x = e.clientX;
    pt.y = e.clientY;
    const svgP = pt.matrixTransform(svg.getScreenCTM()?.inverse());
    const newX = Math.max(nodeRadius + 10, Math.min(containerWidth - nodeRadius - 10, svgP.x - dragOffset.x));
    const newY = Math.max(nodeRadius + 30, Math.min(svgHeight - nodeRadius - 40, svgP.y - dragOffset.y));
    setPositions(prev => ({ ...prev, [draggingNode]: { x: newX, y: newY } }));
  };

  const handleMouseUp = () => {
    setDraggingNode(null);
  };

  const selectedNodeData = selectedNode ? topology.nodes.find(n => n.id === selectedNode) : null;

  // 收集所有出现的 namespace 用于图例
  const namespacesInTopology = useMemo(() => {
    const nsSet = new Set(topology.nodes.map(n => n.namespace));
    return Array.from(nsSet);
  }, [topology]);

  return (
    <div className="p-4 bg-[var(--hover-bg)] rounded-lg">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <Network className="w-4 h-4 text-primary" />
            <span className="text-sm font-medium text-default">服务调用拓扑</span>
          </div>
        </div>
        <div className="flex items-center gap-4 text-[11px]">
          {namespacesInTopology.map(ns => {
            const colors = getNamespaceColor(ns);
            return (
              <div key={ns} className="flex items-center gap-1.5">
                <span
                  className="w-3 h-3 rounded-full border-2"
                  style={{ backgroundColor: colors.fill, borderColor: colors.fill }}
                />
                <span className="text-muted">{ns}</span>
              </div>
            );
          })}
        </div>
      </div>

      {/* SVG 拓扑图 */}
      <div
        ref={containerRef}
        className="relative rounded-xl bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800 border border-slate-200 dark:border-slate-700 overflow-hidden"
      >
        <svg
          ref={svgRef}
          width={containerWidth}
          height={svgHeight}
          className="select-none"
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
        >
          <defs>
            <marker id="arrow-gray" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto">
              <path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#94a3b8" />
            </marker>
            <marker id="arrow-blue" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto">
              <path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#0891b2" />
            </marker>
            <marker id="arrow-amber" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto">
              <path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#d97706" />
            </marker>
            <marker id="arrow-red" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto">
              <path d="M0,0 L10,4 L0,8 L3,4 Z" fill="#dc2626" />
            </marker>
            <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%">
              <feDropShadow dx="0" dy="2" stdDeviation="3" floodOpacity="0.2" />
            </filter>
          </defs>

          {/* 背景网格 */}
          <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
            <path d="M 40 0 L 0 0 0 40" fill="none" stroke="#e2e8f0" strokeWidth="0.5" className="dark:stroke-slate-700" />
          </pattern>
          <rect width="100%" height="100%" fill="url(#grid)" opacity="0.5" />

          {/* 连接线 */}
          <g className="edges">
            {topology.edges.map((edge, idx) => {
              const isHighlighted = hoveredNode === edge.source || hoveredNode === edge.target ||
                                    selectedNode === edge.source || selectedNode === edge.target ||
                                    hoveredEdge === idx;
              const markerId = isHighlighted
                ? (edge.errorRate > 1 ? "arrow-red" : edge.errorRate > 0.1 ? "arrow-amber" : "arrow-blue")
                : "arrow-gray";

              let strokeColor = "#cbd5e1";
              if (edge.errorRate > 1) strokeColor = isHighlighted ? "#ef4444" : "#fca5a5";
              else if (edge.errorRate > 0.1) strokeColor = isHighlighted ? "#f59e0b" : "#fcd34d";
              else if (isHighlighted) strokeColor = "#0ea5e9";

              const labelPos = getEdgeLabelPos(edge.source, edge.target);

              return (
                <g
                  key={idx}
                  onMouseEnter={() => setHoveredEdge(idx)}
                  onMouseLeave={() => setHoveredEdge(null)}
                >
                  {/* 宽透明命中区 */}
                  <path
                    d={getEdgePath(edge.source, edge.target)}
                    fill="none"
                    stroke="transparent"
                    strokeWidth={12}
                    style={{ cursor: "pointer" }}
                  />
                  <path
                    d={getEdgePath(edge.source, edge.target)}
                    fill="none"
                    stroke={strokeColor}
                    strokeWidth={isHighlighted ? 3 : 2}
                    markerEnd={`url(#${markerId})`}
                    className="transition-colors duration-200"
                  />
                  {isHighlighted && (
                    <g transform={`translate(${labelPos.x}, ${labelPos.y})`}>
                      <rect x="-42" y="-12" width="84" height="24" rx="4" fill="white" className="dark:fill-slate-800" stroke="#e2e8f0" strokeWidth="1" />
                      <text textAnchor="middle" y="4" className="text-[10px] font-medium fill-slate-600 dark:fill-slate-300">
                        {edge.rps}/s · {edge.avgLatency}ms
                      </text>
                    </g>
                  )}
                </g>
              );
            })}
          </g>

          {/* 节点 */}
          <g className="nodes">
            {topology.nodes.map((node) => {
              const pos = positions[node.id];
              if (!pos) return null;

              const colors = getNamespaceColor(node.namespace);
              const isSelected = selectedNode === node.id;
              const isHovered = hoveredNode === node.id;
              const isDragging = draggingNode === node.id;

              return (
                <g
                  key={node.id}
                  transform={`translate(${pos.x}, ${pos.y})`}
                  onMouseDown={(e) => handleMouseDown(e, node.id)}
                  onMouseEnter={() => !draggingNode && setHoveredNode(node.id)}
                  onMouseLeave={() => !draggingNode && setHoveredNode(null)}
                  onClick={() => {
                    if (!isDragging) {
                      setSelectedNode(isSelected ? null : node.id);
                      onSelectNode?.(node);
                    }
                  }}
                  style={{ cursor: isDragging ? "grabbing" : "grab" }}
                >
                  {/* 悬停/选中外环 */}
                  {(isSelected || isHovered) && (
                    <circle
                      r={nodeRadius + 5}
                      fill="none"
                      stroke={colors.light}
                      strokeWidth={2}
                      strokeOpacity={isSelected ? 0.8 : 0.5}
                    />
                  )}

                  {/* 主圆 - namespace 着色 */}
                  <circle
                    r={nodeRadius}
                    fill={colors.fill}
                    stroke={colors.stroke}
                    strokeWidth={2}
                    filter="url(#shadow)"
                  />

                  {/* 统一 Server 图标 */}
                  <g style={{ transform: "translate(-10px, -10px)" }}>
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <rect width="20" height="8" x="2" y="2" rx="2" ry="2"/>
                      <rect width="20" height="8" x="2" y="14" rx="2" ry="2"/>
                      <line x1="6" x2="6.01" y1="6" y2="6"/>
                      <line x1="6" x2="6.01" y1="18" y2="18"/>
                    </svg>
                  </g>

                  {/* 服务名 */}
                  <text
                    y={nodeRadius + 16}
                    textAnchor="middle"
                    className="text-[11px] font-semibold fill-slate-700 dark:fill-slate-200 pointer-events-none"
                  >
                    {node.name.length > 14 ? node.name.slice(0, 14) + "…" : node.name}
                  </text>

                  {/* p95 · errorRate */}
                  <text
                    y={nodeRadius + 28}
                    textAnchor="middle"
                    className="text-[9px] fill-slate-500 dark:fill-slate-400 pointer-events-none"
                  >
                    {node.p95Latency}ms · {node.errorRate.toFixed(1)}%
                  </text>

                  {/* 状态点 */}
                  <circle
                    cx={nodeRadius - 4}
                    cy={-nodeRadius + 4}
                    r={5}
                    fill={node.status === "healthy" ? "#10b981" : node.status === "warning" ? "#f59e0b" : "#ef4444"}
                    stroke="white"
                    strokeWidth={2}
                  />
                </g>
              );
            })}
          </g>
        </svg>
      </div>

      {/* 选中节点详情 */}
      {selectedNodeData && (
        <div className="mt-4 p-4 rounded-xl bg-card border border-[var(--border-color)]">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3">
              <div
                className="w-10 h-10 rounded-full flex items-center justify-center text-white shadow-md"
                style={{ backgroundColor: getNamespaceColor(selectedNodeData.namespace).fill }}
              >
                <Server className="w-4 h-4" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-default">{selectedNodeData.name}</span>
                  <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium ${
                    selectedNodeData.status === "healthy" ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" :
                    selectedNodeData.status === "warning" ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400" :
                    "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                  }`}>
                    {selectedNodeData.status === "healthy" ? "健康" : selectedNodeData.status === "warning" ? "告警" : "严重"}
                  </span>
                </div>
                <div className="text-xs text-muted mt-0.5">{selectedNodeData.namespace}</div>
              </div>
            </div>
            <div className="grid grid-cols-4 gap-6 text-center">
              <div>
                <div className="text-xl font-bold text-default">{selectedNodeData.rps}<span className="text-xs font-normal text-muted">/s</span></div>
                <div className="text-[10px] text-muted">RPS</div>
              </div>
              <div>
                <div className="text-xl font-bold text-default">{selectedNodeData.avgLatency}<span className="text-xs font-normal text-muted">ms</span></div>
                <div className="text-[10px] text-muted">Avg延迟</div>
              </div>
              <div>
                <div className="text-xl font-bold text-default">{selectedNodeData.p95Latency}<span className="text-xs font-normal text-muted">ms</span></div>
                <div className="text-[10px] text-muted">P95</div>
              </div>
              <div>
                <div className={`text-xl font-bold ${selectedNodeData.errorRate > 0.1 ? "text-red-500" : "text-emerald-500"}`}>
                  {selectedNodeData.errorRate.toFixed(2)}<span className="text-xs font-normal">%</span>
                </div>
                <div className="text-[10px] text-muted">错误率</div>
              </div>
            </div>
          </div>

          {/* 调用关系 */}
          <div className="mt-3 pt-3 border-t border-[var(--border-color)]">
            <div className="text-xs font-medium text-muted mb-2">调用关系</div>
            <div className="flex flex-wrap gap-2">
              {topology.edges
                .filter(e => e.source === selectedNodeData.id || e.target === selectedNodeData.id)
                .map((edge, idx) => {
                  const isOutgoing = edge.source === selectedNodeData.id;
                  const otherNode = topology.nodes.find(n => n.id === (isOutgoing ? edge.target : edge.source));
                  if (!otherNode) return null;
                  const colors = getNamespaceColor(otherNode.namespace);

                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      {!isOutgoing && (
                        <>
                          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: colors.fill }} />
                          <span className="font-medium text-default">{otherNode.name}</span>
                        </>
                      )}
                      <ArrowRight className={`w-3 h-3 ${isOutgoing ? "text-cyan-600" : "text-slate-400"}`} />
                      {isOutgoing && (
                        <>
                          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: colors.fill }} />
                          <span className="font-medium text-default">{otherNode.name}</span>
                        </>
                      )}
                      <span className="text-muted">{edge.rps}/s · {edge.avgLatency}ms</span>
                    </div>
                  );
                })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// 延迟分布直方图组件
function LatencyDistributionView({ buckets, p50, p95, p99, requestBreakdown, sourceLabel }: {
  buckets: LatencyBucket[];
  p50: number;
  p95: number;
  p99: number;
  requestBreakdown: RequestBreakdown[];
  sourceLabel?: string;
}) {
  const chartRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<{ start: number; end: number } | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState<number | null>(null);

  const maxCount = Math.max(...buckets.map(b => b.count), 1);
  const totalRequests = requestBreakdown.reduce((sum, r) => sum + r.count, 0);
  const maxMethodCount = Math.max(...requestBreakdown.map(r => r.count), 1);

  // 筛选有数据的桶
  const activeBuckets = buckets.filter(b => b.count > 0);
  const minLe = activeBuckets.length > 0 ? activeBuckets[0].le : 0;
  const maxLe = activeBuckets.length > 0 ? activeBuckets[activeBuckets.length - 1].le : 1000;

  const handleMouseDown = (e: React.MouseEvent) => {
    if (!chartRef.current) return;
    const rect = chartRef.current.getBoundingClientRect();
    const x = (e.clientX - rect.left) / rect.width;
    const le = minLe + x * (maxLe - minLe);
    setDragStart(le);
    setIsDragging(true);
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging || dragStart === null || !chartRef.current) return;
    const rect = chartRef.current.getBoundingClientRect();
    const x = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    const le = minLe + x * (maxLe - minLe);
    setSelection({
      start: Math.min(dragStart, le),
      end: Math.max(dragStart, le),
    });
  };

  const handleMouseUp = () => {
    setIsDragging(false);
    setDragStart(null);
  };

  // HTTP 方法颜色
  const methodColors: Record<string, string> = {
    GET: "bg-blue-500",
    POST: "bg-emerald-500",
    PUT: "bg-amber-500",
    DELETE: "bg-red-500",
    PATCH: "bg-violet-500",
  };

  return (
    <div className="space-y-4">
      {/* 延迟分布直方图 */}
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <BarChart3 className="w-4 h-4 text-primary" />
            <span className="text-sm font-medium text-default">延迟分布</span>
            <span className="text-[10px] text-muted">
              {sourceLabel || "Linkerd 24 桶直方图"}
            </span>
          </div>
          <div className="flex items-center gap-3">
            {selection && (
              <button
                onClick={() => setSelection(null)}
                className="text-[10px] text-muted hover:text-default flex items-center gap-1"
              >
                <X className="w-3 h-3" />
                清除选择
              </button>
            )}
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1.5 px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 rounded text-[10px] text-blue-700 dark:text-blue-400 font-medium">
                <span>P50</span>
                <span>{p50}ms</span>
              </div>
              <div className="flex items-center gap-1.5 px-2 py-0.5 bg-amber-100 dark:bg-amber-900/30 rounded text-[10px] text-amber-700 dark:text-amber-400 font-medium">
                <span>P95</span>
                <span>{p95}ms</span>
              </div>
              <div className="flex items-center gap-1.5 px-2 py-0.5 bg-red-100 dark:bg-red-900/30 rounded text-[10px] text-red-700 dark:text-red-400 font-medium">
                <span>P99</span>
                <span>{p99}ms</span>
              </div>
            </div>
          </div>
        </div>

        {/* 直方图区域 */}
        <div className="px-4 pt-4 pb-6">
          <div className="flex gap-2">
            {/* Y轴刻度 */}
            <div className="flex flex-col justify-between h-36 text-[9px] text-muted text-right w-8 flex-shrink-0">
              <span>{maxCount}</span>
              <span>{Math.round(maxCount / 2)}</span>
              <span>0</span>
            </div>

            {/* 直方图主体 */}
            <div
              ref={chartRef}
              className="relative h-36 cursor-crosshair select-none flex-1"
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
            >
              {/* Y轴网格线 */}
              <div className="absolute inset-0 pointer-events-none">
                <div className="absolute top-0 left-0 right-0 border-t border-slate-200 dark:border-slate-700" />
                <div className="absolute top-1/2 left-0 right-0 border-t border-dashed border-slate-200 dark:border-slate-700" />
                <div className="absolute bottom-0 left-0 right-0 border-t border-slate-200 dark:border-slate-700" />
              </div>

              {/* 选择区域高亮 */}
              {selection && (
                <div
                  className="absolute top-0 bottom-0 border-l-2 border-r-2 border-blue-500 bg-blue-500/5 pointer-events-none"
                  style={{
                    left: `${((selection.start - minLe) / (maxLe - minLe)) * 100}%`,
                    width: `${((selection.end - selection.start) / (maxLe - minLe)) * 100}%`,
                  }}
                />
              )}

              {/* P50 垂直线 */}
              {p50 >= minLe && p50 <= maxLe && (
                <div
                  className="absolute top-0 bottom-0 w-px bg-blue-500/70 z-20 pointer-events-none"
                  style={{ left: `${((p50 - minLe) / (maxLe - minLe)) * 100}%` }}
                >
                  <div className="absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-blue-500 text-white text-[8px] font-medium whitespace-nowrap rounded">
                    P50
                  </div>
                </div>
              )}

              {/* P95 垂直线 */}
              {p95 >= minLe && p95 <= maxLe && (
                <div
                  className="absolute top-0 bottom-0 w-px bg-amber-500/70 z-20 pointer-events-none"
                  style={{ left: `${((p95 - minLe) / (maxLe - minLe)) * 100}%` }}
                >
                  <div className="absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-amber-500 text-white text-[8px] font-medium whitespace-nowrap rounded">
                    P95
                  </div>
                </div>
              )}

              {/* P99 垂直线 */}
              {p99 >= minLe && p99 <= maxLe && (
                <div
                  className="absolute top-0 bottom-0 w-px bg-red-500/70 z-20 pointer-events-none"
                  style={{ left: `${((p99 - minLe) / (maxLe - minLe)) * 100}%` }}
                >
                  <div className="absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-red-500 text-white text-[8px] font-medium whitespace-nowrap rounded">
                    P99
                  </div>
                </div>
              )}

              {/* 条形图 */}
              <div className="flex items-end h-full gap-[2px] relative z-10">
                {buckets.map((bucket, idx) => {
                  const heightPercent = (bucket.count / maxCount) * 100;
                  const isInSelection = selection &&
                    bucket.le >= selection.start &&
                    bucket.le <= selection.end;
                  const isNearP50 = Math.abs(bucket.le - p50) < (maxLe - minLe) * 0.05;
                  const isNearP95 = Math.abs(bucket.le - p95) < (maxLe - minLe) * 0.05;
                  const isNearP99 = Math.abs(bucket.le - p99) < (maxLe - minLe) * 0.05;

                  return (
                    <div
                      key={idx}
                      className="flex-1 flex flex-col items-center justify-end group relative"
                      style={{ height: "100%" }}
                    >
                      <div
                        className={`w-full rounded-t-sm transition-all duration-150 ${
                          isInSelection
                            ? "bg-blue-500"
                            : isNearP99
                              ? "bg-red-400/80 group-hover:bg-red-500"
                              : isNearP95
                                ? "bg-amber-400/90 group-hover:bg-amber-500"
                                : isNearP50
                                  ? "bg-blue-400/80 group-hover:bg-blue-500"
                                  : bucket.count > 0
                                    ? "bg-teal-400/80 group-hover:bg-teal-500 dark:bg-teal-500/70 dark:group-hover:bg-teal-400"
                                    : "bg-transparent"
                        }`}
                        style={{
                          height: bucket.count > 0 ? `${Math.max(heightPercent, 3)}%` : "0",
                        }}
                      />
                      {/* Tooltip */}
                      {bucket.count > 0 && (
                        <div className="absolute bottom-full mb-2 hidden group-hover:block z-30 pointer-events-none">
                          <div className="bg-slate-900 text-white text-[10px] px-2.5 py-1.5 rounded-md shadow-xl whitespace-nowrap border border-slate-700">
                            <div className="font-medium">≤ {bucket.le}ms</div>
                            <div className="text-slate-300">{bucket.count} 请求</div>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* X轴延迟标签 */}
              <div className="absolute -bottom-5 left-0 right-0 flex justify-between text-[9px] text-muted">
                <span>{minLe}ms</span>
                <span>{Math.round(minLe + (maxLe - minLe) * 0.25)}ms</span>
                <span>{Math.round(minLe + (maxLe - minLe) * 0.5)}ms</span>
                <span>{Math.round(minLe + (maxLe - minLe) * 0.75)}ms</span>
                <span>{maxLe}ms</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* HTTP 方法请求分布 */}
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center gap-3">
          <Layers className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">请求方法分布</span>
          <span className="text-[10px] text-muted">
            共 {totalRequests.toLocaleString()} 请求
          </span>
        </div>
        <div className="p-4 space-y-2.5">
          {requestBreakdown.map((rb) => {
            const percent = totalRequests > 0 ? (rb.count / totalRequests) * 100 : 0;
            const barWidth = (rb.count / maxMethodCount) * 100;
            const errorPercent = rb.count > 0 ? (rb.errorCount / rb.count) * 100 : 0;
            return (
              <div key={rb.method} className="flex items-center gap-3">
                <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-medium w-14 text-center text-white ${methodColors[rb.method] || "bg-slate-500"}`}>
                  {rb.method}
                </span>
                <div className="flex-1 h-5 bg-slate-100 dark:bg-slate-800 rounded-sm overflow-hidden relative">
                  <div
                    className={`h-full rounded-sm ${methodColors[rb.method] || "bg-slate-500"} opacity-80`}
                    style={{ width: `${barWidth}%` }}
                  />
                </div>
                <div className="w-28 text-right flex items-center gap-2 justify-end">
                  <span className="text-xs font-medium text-default">{rb.count.toLocaleString()}</span>
                  <span className="text-[10px] text-muted">({percent.toFixed(0)}%)</span>
                </div>
                {errorPercent > 0 && (
                  <span className="text-[10px] text-red-500">{errorPercent.toFixed(2)}% err</span>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// 后端服务 golden metrics 网格
function BackendServicesView({ serviceIds, topology }: {
  serviceIds: string[];
  topology: ServiceTopology;
}) {
  const services = serviceIds
    .map(id => topology.nodes.find(n => n.id === id))
    .filter((n): n is ServiceNode => n !== undefined);

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 text-xs text-muted">
        <Server className="w-4 h-4" />
        <span>后端服务 ({services.length})</span>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {services.map((svc) => {
          const colors = getNamespaceColor(svc.namespace);
          return (
            <div key={svc.id} className="p-4 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <div
                    className="w-8 h-8 rounded-full flex items-center justify-center text-white"
                    style={{ backgroundColor: colors.fill }}
                  >
                    <Server className="w-3.5 h-3.5" />
                  </div>
                  <div>
                    <div className="text-sm font-semibold text-default">{svc.name}</div>
                    <div className="text-[10px] text-muted">{svc.namespace}</div>
                  </div>
                </div>
                <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium ${
                  svc.status === "healthy" ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" :
                  svc.status === "warning" ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400" :
                  "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                }`}>
                  {svc.status === "healthy" ? "健康" : svc.status === "warning" ? "告警" : "严重"}
                </span>
              </div>
              <div className="grid grid-cols-3 gap-3 text-center">
                <div className="p-2 rounded-lg bg-slate-50 dark:bg-slate-800">
                  <div className="text-sm font-bold text-default">{svc.rps.toLocaleString()}<span className="text-[10px] font-normal text-muted">/s</span></div>
                  <div className="text-[10px] text-muted">RPS</div>
                </div>
                <div className="p-2 rounded-lg bg-slate-50 dark:bg-slate-800">
                  <div className="text-sm font-bold text-default">{svc.p95Latency}<span className="text-[10px] font-normal text-muted">ms</span></div>
                  <div className="text-[10px] text-muted">P95</div>
                </div>
                <div className="p-2 rounded-lg bg-slate-50 dark:bg-slate-800">
                  <div className={`text-sm font-bold ${svc.errorRate > 0.5 ? "text-red-500" : "text-emerald-500"}`}>
                    {svc.errorRate.toFixed(2)}<span className="text-[10px] font-normal">%</span>
                  </div>
                  <div className="text-[10px] text-muted">错误率</div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// SLO 趋势图（SVG 面积图）
function HistoryChartView({ history, targets }: {
  history: HistoryPoint[];
  targets: SLOTargets;
}) {
  const [activeMetric, setActiveMetric] = useState<"p95Latency" | "errorRate">("p95Latency");
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  const metrics = [
    { id: "p95Latency" as const, label: "P95 延迟", unit: "ms", color: "#0891b2", targetKey: "p95Latency" as const },
    { id: "errorRate" as const, label: "错误率", unit: "%", color: "#ef4444", targetKey: null },
  ];

  const currentMetric = metrics.find(m => m.id === activeMetric)!;

  const values = history.map(p => {
    switch (activeMetric) {
      case "p95Latency": return p.p95Latency;
      case "errorRate": return p.errorRate;
    }
  });

  // SLO 目标值（P95 延迟有目标线）
  let targetVal: number | null = null;
  if (activeMetric === "p95Latency") targetVal = targets.p95Latency;

  // Y 轴范围需包含 SLO 目标值，否则目标线会超出图表
  let rawMin = Math.min(...values);
  let rawMax = Math.max(...values);
  if (targetVal !== null) {
    rawMin = Math.min(rawMin, targetVal);
    rawMax = Math.max(rawMax, targetVal);
  }
  const minVal = rawMin - (rawMax - rawMin) * 0.05;
  const maxVal = rawMax + (rawMax - rawMin) * 0.05;
  const range = maxVal - minVal || 1;

  const width = 660;
  const height = 180;
  const padLeft = 55;
  const padRight = 5;
  const padTop = 10;
  const padBottom = 25;
  const chartH = height - padTop - padBottom;
  const chartW = width - padLeft - padRight;

  const points = values.map((v, i) => ({
    x: padLeft + (i / Math.max(values.length - 1, 1)) * chartW,
    y: padTop + (1 - (v - minVal) / range) * chartH,
    value: v,
    timestamp: history[i].timestamp,
  }));

  const linePath = points.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");
  const areaPath = `${linePath} L ${points[points.length - 1]?.x ?? 0} ${padTop + chartH} L ${points[0]?.x ?? 0} ${padTop + chartH} Z`;

  const targetY = targetVal !== null ? padTop + (1 - (targetVal - minVal) / range) * chartH : null;

  // X 轴标签
  const xLabels: { x: number; label: string }[] = [];
  if (history.length > 0) {
    const step = Math.max(1, Math.floor(history.length / 6));
    for (let i = 0; i < history.length; i += step) {
      const d = new Date(history[i].timestamp);
      xLabels.push({
        x: padLeft + (i / Math.max(history.length - 1, 1)) * chartW,
        label: `${d.getMonth() + 1}/${d.getDate()}`,
      });
    }
  }

  const formatValue = (v: number) => {
    if (activeMetric === "p95Latency") return Math.round(v) + "ms";
    return v.toFixed(3) + "%";
  };

  const gradientId = `history-grad-${activeMetric}`;

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <TrendingUp className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">SLO 趋势</span>
        </div>
        <div className="flex items-center gap-1 p-0.5 rounded-lg bg-slate-100 dark:bg-slate-800">
          {metrics.map(m => (
            <button
              key={m.id}
              onClick={() => setActiveMetric(m.id)}
              className={`px-2.5 py-1 text-[10px] rounded-md transition-colors ${
                activeMetric === m.id
                  ? "bg-white dark:bg-slate-700 text-default shadow-sm font-medium"
                  : "text-muted hover:text-default"
              }`}
            >{m.label}</button>
          ))}
        </div>
      </div>
      <div className="p-4">
        <svg
          ref={svgRef}
          viewBox={`0 0 ${width} ${height}`}
          className="w-full h-auto"
          onMouseLeave={() => setHoveredIndex(null)}
          onMouseMove={(e) => {
            const svg = svgRef.current;
            if (!svg) return;
            const rect = svg.getBoundingClientRect();
            const mouseX = ((e.clientX - rect.left) / rect.width) * width;
            let closest = 0;
            let minDist = Infinity;
            points.forEach((p, i) => {
              const dist = Math.abs(p.x - mouseX);
              if (dist < minDist) { minDist = dist; closest = i; }
            });
            setHoveredIndex(closest);
          }}
        >
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={currentMetric.color} stopOpacity="0.3" />
              <stop offset="100%" stopColor={currentMetric.color} stopOpacity="0.02" />
            </linearGradient>
          </defs>

          {/* Y 轴网格 + 左侧刻度标签 */}
          {[0, 0.25, 0.5, 0.75, 1].map((r, i) => {
            const y = padTop + r * chartH;
            const val = maxVal - r * range;
            return (
              <g key={i}>
                <line x1={padLeft} y1={y} x2={padLeft + chartW} y2={y} stroke="#e2e8f0" strokeWidth="0.5" strokeDasharray={i === 0 || i === 4 ? "0" : "3 3"} className="dark:stroke-slate-700" />
                <text x={padLeft - 6} y={y + 3} textAnchor="end" className="text-[9px] fill-slate-400">{formatValue(val)}</text>
              </g>
            );
          })}

          {/* SLO 目标线 */}
          {targetY !== null && targetY >= padTop && targetY <= padTop + chartH && (
            <g>
              <line x1={padLeft} y1={targetY} x2={padLeft + chartW} y2={targetY} stroke="#f59e0b" strokeWidth="1.5" strokeDasharray="6 3" />
              <text x={padLeft + 4} y={targetY - 4} className="text-[9px] fill-amber-500 font-medium">SLO 目标: {targetVal}{currentMetric.unit}</text>
            </g>
          )}

          {/* 面积填充 */}
          {points.length > 1 && (
            <path d={areaPath} fill={`url(#${gradientId})`} />
          )}

          {/* 趋势线 */}
          {points.length > 1 && (
            <path d={linePath} fill="none" stroke={currentMetric.color} strokeWidth="2" strokeLinejoin="round" />
          )}

          {/* 数据点 */}
          {points.map((p, i) => (
            <circle
              key={i}
              cx={p.x}
              cy={p.y}
              r={hoveredIndex === i ? 4 : 0}
              fill={currentMetric.color}
              stroke="white"
              strokeWidth="2"
            />
          ))}

          {/* X 轴标签 */}
          {xLabels.map((l, i) => (
            <text key={i} x={l.x} y={height - 4} textAnchor="middle" className="text-[9px] fill-slate-400">{l.label}</text>
          ))}

          {/* Hover tooltip */}
          {hoveredIndex !== null && points[hoveredIndex] && (
            <g>
              <line x1={points[hoveredIndex].x} y1={padTop} x2={points[hoveredIndex].x} y2={padTop + chartH} stroke="#94a3b8" strokeWidth="0.5" strokeDasharray="3 3" />
              <rect
                x={Math.min(points[hoveredIndex].x - 50, width - 105)}
                y={Math.max(points[hoveredIndex].y - 38, padTop)}
                width="100"
                height="30"
                rx="4"
                fill="#1e293b"
                opacity="0.95"
              />
              <text
                x={Math.min(points[hoveredIndex].x - 50, width - 105) + 50}
                y={Math.max(points[hoveredIndex].y - 38, padTop) + 13}
                textAnchor="middle"
                className="text-[9px] fill-white font-medium"
              >
                {formatValue(points[hoveredIndex].value)}
              </text>
              <text
                x={Math.min(points[hoveredIndex].x - 50, width - 105) + 50}
                y={Math.max(points[hoveredIndex].y - 38, padTop) + 24}
                textAnchor="middle"
                className="text-[8px] fill-slate-400"
              >
                {new Date(points[hoveredIndex].timestamp).toLocaleString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" })}
              </text>
            </g>
          )}
        </svg>
      </div>
    </div>
  );
}

// 错误预算消耗图
function ErrorBudgetBurnView({ history }: { history: HistoryPoint[] }) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  const values = history.map(p => p.errorBudgetRemaining);
  const width = 600;
  const height = 160;
  const padX = 0;
  const padTop = 10;
  const padBottom = 25;
  const chartH = height - padTop - padBottom;
  const chartW = width - padX * 2;

  const points = values.map((v, i) => ({
    x: padX + (i / Math.max(values.length - 1, 1)) * chartW,
    y: padTop + (1 - v / 100) * chartH,
    value: v,
    timestamp: history[i].timestamp,
  }));

  const linePath = points.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");

  // 预测耗尽：线性回归
  const n = values.length;
  let exhaustDate = "";
  if (n > 2) {
    const first = values[0];
    const last = values[n - 1];
    const rate = (first - last) / n; // 每点消耗
    if (rate > 0) {
      const pointsToZero = last / rate;
      const hoursPerPoint = 4;
      const hoursLeft = pointsToZero * hoursPerPoint;
      const d = new Date(history[n - 1].timestamp);
      d.setHours(d.getHours() + hoursLeft);
      exhaustDate = `${d.getMonth() + 1}/${d.getDate()}`;
    }
  }

  // 当前值
  const currentBudget = values[values.length - 1] ?? 0;
  const budgetColor = currentBudget > 50 ? "#10b981" : currentBudget > 20 ? "#f59e0b" : "#ef4444";

  // X 轴标签
  const xLabels: { x: number; label: string }[] = [];
  if (history.length > 0) {
    const step = Math.max(1, Math.floor(history.length / 6));
    for (let i = 0; i < history.length; i += step) {
      const d = new Date(history[i].timestamp);
      xLabels.push({
        x: padX + (i / Math.max(history.length - 1, 1)) * chartW,
        label: `${d.getMonth() + 1}/${d.getDate()}`,
      });
    }
  }

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Target className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">错误预算消耗</span>
        </div>
        <div className="flex items-center gap-3 text-[11px]">
          <span className="text-muted">当前</span>
          <span className="font-semibold" style={{ color: budgetColor }}>{currentBudget.toFixed(1)}%</span>
          {exhaustDate && (
            <>
              <span className="text-muted">预计耗尽</span>
              <span className="font-semibold text-red-500">~{exhaustDate}</span>
            </>
          )}
        </div>
      </div>
      <div className="p-4">
        <svg
          ref={svgRef}
          viewBox={`0 0 ${width} ${height}`}
          className="w-full h-auto"
          onMouseLeave={() => setHoveredIndex(null)}
          onMouseMove={(e) => {
            const svg = svgRef.current;
            if (!svg) return;
            const rect = svg.getBoundingClientRect();
            const mouseX = ((e.clientX - rect.left) / rect.width) * width;
            let closest = 0;
            let minDist = Infinity;
            points.forEach((p, i) => {
              const dist = Math.abs(p.x - mouseX);
              if (dist < minDist) { minDist = dist; closest = i; }
            });
            setHoveredIndex(closest);
          }}
        >
          <defs>
            <linearGradient id="budget-grad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#10b981" stopOpacity="0.15" />
              <stop offset="50%" stopColor="#f59e0b" stopOpacity="0.15" />
              <stop offset="100%" stopColor="#ef4444" stopOpacity="0.15" />
            </linearGradient>
          </defs>

          {/* 背景渐变 */}
          <rect x={padX} y={padTop} width={chartW} height={chartH} fill="url(#budget-grad)" rx="4" />

          {/* 网格线 */}
          {[0, 25, 50, 75, 100].map((v, i) => {
            const y = padTop + (1 - v / 100) * chartH;
            return (
              <g key={i}>
                <line x1={padX} y1={y} x2={padX + chartW} y2={y} stroke="#e2e8f0" strokeWidth="0.5" strokeDasharray="3 3" className="dark:stroke-slate-700" />
                <text x={padX + chartW + 4} y={y + 3} className="text-[9px] fill-slate-400">{v}%</text>
              </g>
            );
          })}

          {/* 消耗线 */}
          {points.length > 1 && (
            <path d={linePath} fill="none" stroke={budgetColor} strokeWidth="2.5" strokeLinejoin="round" />
          )}

          {/* 预测虚线延伸 */}
          {exhaustDate && points.length > 1 && (
            <line
              x1={points[points.length - 1].x}
              y1={points[points.length - 1].y}
              x2={padX + chartW}
              y2={padTop + chartH}
              stroke="#ef4444"
              strokeWidth="1.5"
              strokeDasharray="6 4"
              opacity="0.6"
            />
          )}

          {/* 数据点 */}
          {points.map((p, i) => (
            <circle
              key={i}
              cx={p.x}
              cy={p.y}
              r={hoveredIndex === i ? 4 : 0}
              fill={budgetColor}
              stroke="white"
              strokeWidth="2"
            />
          ))}

          {/* X 轴标签 */}
          {xLabels.map((l, i) => (
            <text key={i} x={l.x} y={height - 4} textAnchor="middle" className="text-[9px] fill-slate-400">{l.label}</text>
          ))}

          {/* Hover tooltip */}
          {hoveredIndex !== null && points[hoveredIndex] && (
            <g>
              <line x1={points[hoveredIndex].x} y1={padTop} x2={points[hoveredIndex].x} y2={padTop + chartH} stroke="#94a3b8" strokeWidth="0.5" strokeDasharray="3 3" />
              <rect
                x={Math.min(points[hoveredIndex].x - 40, width - 85)}
                y={Math.max(points[hoveredIndex].y - 38, padTop)}
                width="80"
                height="30"
                rx="4"
                fill="#1e293b"
                opacity="0.95"
              />
              <text
                x={Math.min(points[hoveredIndex].x - 40, width - 85) + 40}
                y={Math.max(points[hoveredIndex].y - 38, padTop) + 13}
                textAnchor="middle"
                className="text-[9px] fill-white font-medium"
              >
                {points[hoveredIndex].value.toFixed(1)}%
              </text>
              <text
                x={Math.min(points[hoveredIndex].x - 40, width - 85) + 40}
                y={Math.max(points[hoveredIndex].y - 38, padTop) + 24}
                textAnchor="middle"
                className="text-[8px] fill-slate-400"
              >
                {new Date(points[hoveredIndex].timestamp).toLocaleString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" })}
              </text>
            </g>
          )}
        </svg>
      </div>
    </div>
  );
}

// 状态码分布横条图
function StatusCodeBreakdownView({ breakdown = [] }: { breakdown: StatusCodeBreakdown[] }) {
  if (breakdown.length === 0) return null;
  const total = breakdown.reduce((sum, b) => sum + b.count, 0);
  const maxCount = Math.max(...breakdown.map(b => b.count), 1);

  const codeColors: Record<string, { bar: string; bg: string; text: string }> = {
    "2xx": { bar: "bg-emerald-500", bg: "bg-emerald-50 dark:bg-emerald-900/20", text: "text-emerald-700 dark:text-emerald-400" },
    "3xx": { bar: "bg-blue-500", bg: "bg-blue-50 dark:bg-blue-900/20", text: "text-blue-700 dark:text-blue-400" },
    "4xx": { bar: "bg-amber-500", bg: "bg-amber-50 dark:bg-amber-900/20", text: "text-amber-700 dark:text-amber-400" },
    "5xx": { bar: "bg-red-500", bg: "bg-red-50 dark:bg-red-900/20", text: "text-red-700 dark:text-red-400" },
  };

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center gap-3">
        <BarChart3 className="w-4 h-4 text-primary" />
        <span className="text-sm font-medium text-default">状态码分布</span>
        <span className="text-[10px] text-muted">共 {total.toLocaleString()} 请求</span>
      </div>
      <div className="p-4 space-y-2.5">
        {breakdown.map((b) => {
          const percent = total > 0 ? (b.count / total) * 100 : 0;
          const barWidth = (b.count / maxCount) * 100;
          const colors = codeColors[b.code] || codeColors["2xx"];
          return (
            <div key={b.code} className="flex items-center gap-3">
              <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-semibold w-10 text-center ${colors.text} ${colors.bg}`}>
                {b.code}
              </span>
              <div className="flex-1 h-5 bg-slate-100 dark:bg-slate-800 rounded-sm overflow-hidden relative">
                <div
                  className={`h-full rounded-sm ${colors.bar} opacity-80`}
                  style={{ width: `${barWidth}%` }}
                />
              </div>
              <div className="w-32 text-right flex items-center gap-2 justify-end">
                <span className="text-xs font-medium text-default">{percent.toFixed(1)}%</span>
                <span className="text-[10px] text-muted">{b.count.toLocaleString()}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// 服务列表表格（可排序）
function ServiceListTable({ nodes, selectedId, onSelect }: {
  nodes: ServiceNode[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const [sortKey, setSortKey] = useState<"name" | "rps" | "p95Latency" | "errorRate">("rps");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  const toggleSort = (key: typeof sortKey) => {
    if (sortKey === key) {
      setSortDir(d => d === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  };

  const sorted = useMemo(() => {
    const arr = [...nodes];
    arr.sort((a, b) => {
      const aVal = sortKey === "name" ? a.name : a[sortKey];
      const bVal = sortKey === "name" ? b.name : b[sortKey];
      if (typeof aVal === "string" && typeof bVal === "string") {
        return sortDir === "asc" ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      }
      return sortDir === "asc" ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
    });
    return arr;
  }, [nodes, sortKey, sortDir]);

  const SortHeader = ({ label, field }: { label: string; field: typeof sortKey }) => (
    <button
      onClick={() => toggleSort(field)}
      className={`flex items-center gap-1 text-[10px] font-medium uppercase tracking-wider ${
        sortKey === field ? "text-primary" : "text-muted hover:text-default"
      }`}
    >
      {label}
      {sortKey === field && (
        <span className="text-[8px]">{sortDir === "asc" ? "▲" : "▼"}</span>
      )}
    </button>
  );

  return (
    <div className="overflow-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-[var(--border-color)]">
            <th className="text-left py-2 px-2"><SortHeader label="服务" field="name" /></th>
            <th className="text-right py-2 px-2"><SortHeader label="RPS" field="rps" /></th>
            <th className="text-right py-2 px-2"><SortHeader label="P95" field="p95Latency" /></th>
            <th className="text-right py-2 px-2"><SortHeader label="错误率" field="errorRate" /></th>
            <th className="text-right py-2 px-2"><span className="text-[10px] font-medium uppercase tracking-wider text-muted">mTLS</span></th>
            <th className="text-center py-2 px-2"><span className="text-[10px] font-medium uppercase tracking-wider text-muted">状态</span></th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((node) => {
            const nsColor = getNamespaceColor(node.namespace);
            const isSelected = selectedId === node.id;
            return (
              <tr
                key={node.id}
                onClick={() => onSelect(node.id)}
                className={`cursor-pointer transition-colors border-b border-[var(--border-color)] ${
                  isSelected
                    ? "bg-primary/5 dark:bg-primary/10"
                    : "hover:bg-[var(--hover-bg)]"
                }`}
              >
                <td className="py-2.5 px-2">
                  <div className="flex items-center gap-2">
                    <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: nsColor.fill }} />
                    <div>
                      <div className="font-medium text-default">{node.name}</div>
                      <div className="text-[10px] text-muted">{node.namespace}</div>
                    </div>
                  </div>
                </td>
                <td className="text-right py-2.5 px-2 font-medium text-default">{node.rps.toLocaleString()}<span className="text-muted">/s</span></td>
                <td className="text-right py-2.5 px-2 font-medium text-default">{node.p95Latency}<span className="text-muted">ms</span></td>
                <td className="text-right py-2.5 px-2">
                  <span className={node.errorRate > 0.5 ? "text-red-500 font-semibold" : "text-default font-medium"}>
                    {node.errorRate.toFixed(2)}%
                  </span>
                </td>
                <td className="text-right py-2.5 px-2">
                  <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                    node.mtlsEnabled
                      ? "bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400"
                      : "bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400"
                  }`}>
                    {node.mtlsEnabled ? "ON" : "OFF"}
                  </span>
                </td>
                <td className="text-center py-2.5 px-2">
                  <span className={`inline-block w-2 h-2 rounded-full ${
                    node.status === "healthy" ? "bg-emerald-500" :
                    node.status === "warning" ? "bg-amber-500" : "bg-red-500"
                  }`} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// 服务详情面板
function ServiceDetailPanel({ node, topology }: {
  node: ServiceNode;
  topology: ServiceTopology;
}) {
  const nsColor = getNamespaceColor(node.namespace);

  // 调用关系
  const inbound = topology.edges.filter(e => e.target === node.id);
  const outbound = topology.edges.filter(e => e.source === node.id);

  return (
    <div className="space-y-4">
      {/* 标题 */}
      <div className="flex items-center gap-3">
        <div
          className="w-10 h-10 rounded-full flex items-center justify-center text-white shadow-md"
          style={{ backgroundColor: nsColor.fill }}
        >
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
              {node.status === "healthy" ? "健康" : node.status === "warning" ? "告警" : "严重"}
            </span>
          </div>
          <div className="text-xs text-muted mt-0.5">{node.namespace}</div>
        </div>
      </div>

      {/* Golden Metrics 网格 */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-3">
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-[10px] text-muted mb-1">RPS</div>
          <div className="text-lg font-bold text-default">{node.rps.toLocaleString()}<span className="text-xs font-normal text-muted">/s</span></div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-[10px] text-muted mb-1">P50 延迟</div>
          <div className="text-lg font-bold text-default">{node.p50Latency}<span className="text-xs font-normal text-muted">ms</span></div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-[10px] text-muted mb-1">P95 延迟</div>
          <div className="text-lg font-bold text-default">{node.p95Latency}<span className="text-xs font-normal text-muted">ms</span></div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-[10px] text-muted mb-1">P99 延迟</div>
          <div className="text-lg font-bold text-default">{node.p99Latency}<span className="text-xs font-normal text-muted">ms</span></div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-[10px] text-muted mb-1">错误率</div>
          <div className={`text-lg font-bold ${node.errorRate > 0.5 ? "text-red-500" : "text-emerald-500"}`}>
            {node.errorRate.toFixed(2)}<span className="text-xs font-normal">%</span>
          </div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-[10px] text-muted mb-1">总请求</div>
          <div className="text-lg font-bold text-default">{node.totalRequests.toLocaleString()}</div>
        </div>
      </div>

      {/* Linkerd 服务延迟分布 */}
      <LatencyDistributionView
        buckets={node.latencyDistribution}
        p50={node.p50Latency}
        p95={node.p95Latency}
        p99={node.p99Latency}
        requestBreakdown={node.requestBreakdown}
        sourceLabel="Linkerd 服务延迟分布"
      />

      {/* 调用关系 */}
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center gap-3">
          <Network className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">调用关系</span>
        </div>
        <div className="p-4 space-y-3">
          {inbound.length > 0 && (
            <div>
              <div className="text-[10px] text-muted font-medium uppercase tracking-wider mb-2">Inbound ({inbound.length})</div>
              <div className="flex flex-wrap gap-2">
                {inbound.map((edge, idx) => {
                  const srcNode = topology.nodes.find(n => n.id === edge.source);
                  if (!srcNode) return null;
                  const srcColor = getNamespaceColor(srcNode.namespace);
                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: srcColor.fill }} />
                      <span className="font-medium text-default">{srcNode.name}</span>
                      <ArrowRight className="w-3 h-3 text-slate-400" />
                      <span className="text-muted">{edge.rps}/s · {edge.avgLatency}ms</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
          {outbound.length > 0 && (
            <div>
              <div className="text-[10px] text-muted font-medium uppercase tracking-wider mb-2">Outbound ({outbound.length})</div>
              <div className="flex flex-wrap gap-2">
                {outbound.map((edge, idx) => {
                  const tgtNode = topology.nodes.find(n => n.id === edge.target);
                  if (!tgtNode) return null;
                  const tgtColor = getNamespaceColor(tgtNode.namespace);
                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      <ArrowRight className="w-3 h-3 text-cyan-600" />
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: tgtColor.fill }} />
                      <span className="font-medium text-default">{tgtNode.name}</span>
                      <span className="text-muted">{edge.rps}/s · {edge.avgLatency}ms</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
          {inbound.length === 0 && outbound.length === 0 && (
            <div className="text-xs text-muted text-center py-4">暂无调用关系数据</div>
          )}
        </div>
      </div>

      {/* 状态码分布 */}
      <StatusCodeBreakdownView breakdown={node.statusCodeBreakdown} />
    </div>
  );
}

// 服务网格概览（双栏容器）
function ServiceMeshOverview({ topology, selectedServiceId, onSelectService }: {
  topology: ServiceTopology;
  selectedServiceId: string | null;
  onSelectService: (id: string) => void;
}) {
  const selectedNode = selectedServiceId
    ? topology.nodes.find(n => n.id === selectedServiceId)
    : null;

  // mTLS 启用状态
  const mtlsEnabledCount = topology.nodes.filter(n => n.mtlsEnabled).length;
  const allMtlsEnabled = mtlsEnabledCount === topology.nodes.length;

  return (
    <div className="rounded-xl border border-[var(--border-color)] bg-card overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Layers className="w-4 h-4 text-primary" />
          <span className="text-sm font-semibold text-default">服务网格概览</span>
          <span className="text-[10px] text-muted">Linkerd mesh · {topology.nodes.length} 个服务</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <Shield className="w-3.5 h-3.5 text-muted" />
            <span className="text-[10px] text-muted">mTLS</span>
            <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold ${
              allMtlsEnabled
                ? "bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400"
                : "bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400"
            }`}>
              {allMtlsEnabled ? "ON" : `${mtlsEnabledCount}/${topology.nodes.length}`}
            </span>
          </div>
        </div>
      </div>
      <div className="flex flex-col lg:flex-row">
        {/* 左栏：服务列表 */}
        <div className={`${selectedNode ? "lg:w-[400px] lg:border-r border-[var(--border-color)]" : "w-full"} p-4`}>
          <ServiceListTable
            nodes={topology.nodes}
            selectedId={selectedServiceId}
            onSelect={onSelectService}
          />
        </div>
        {/* 右栏：服务详情（条件渲染） */}
        {selectedNode && (
          <div className="flex-1 p-4 bg-[var(--background)]">
            <ServiceDetailPanel node={selectedNode} topology={topology} />
          </div>
        )}
      </div>
    </div>
  );
}

// mTLS 状态卡片
function MtlsCoverageView({ topology }: { topology: ServiceTopology }) {
  const nodes = topology.nodes;

  const mtlsEnabledCount = nodes.filter(n => n.mtlsEnabled).length;
  const allEnabled = mtlsEnabledCount === nodes.length;

  return (
    <div className="rounded-xl border border-[var(--border-color)] bg-card overflow-hidden">
      <div className="px-4 py-3">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Shield className="w-4 h-4 text-primary" />
            <span className="text-sm font-medium text-default">mTLS</span>
          </div>
          <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold ${
            allEnabled
              ? "bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400"
              : "bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400"
          }`}>
            {allEnabled ? "ON" : `${mtlsEnabledCount}/${nodes.length} ON`}
          </span>
        </div>

        {/* Per-service 药丸 */}
        <div className="flex flex-wrap gap-1.5">
          {nodes.map((node) => {
            const nsColor = getNamespaceColor(node.namespace);
            return (
              <div
                key={node.id}
                className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-[10px] font-medium ${
                  node.mtlsEnabled
                    ? "bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400"
                    : "bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400"
                }`}
              >
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: nsColor.fill }} />
                <span>{node.name}</span>
                <span className="font-bold">{node.mtlsEnabled ? "ON" : "OFF"}</span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// 趋势图标
function TrendIcon({ trend }: { trend: "up" | "down" | "stable" }) {
  if (trend === "up") return <TrendingUp className="w-4 h-4 text-emerald-500" />;
  if (trend === "down") return <TrendingDown className="w-4 h-4 text-red-500" />;
  return <Minus className="w-4 h-4 text-gray-400" />;
}

// 状态徽章
function StatusBadge({ status }: { status: "healthy" | "warning" | "critical" }) {
  const config = {
    healthy: { bg: "bg-emerald-500/10", text: "text-emerald-500", dot: "bg-emerald-500" },
    warning: { bg: "bg-amber-500/10", text: "text-amber-500", dot: "bg-amber-500" },
    critical: { bg: "bg-red-500/10", text: "text-red-500", dot: "bg-red-500" },
  };
  const c = config[status];
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${c.bg} ${c.text}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot}`} />
      {status === "healthy" ? "健康" : status === "warning" ? "告警" : "严重"}
    </span>
  );
}

// 错误预算条
function ErrorBudgetBar({ percent }: { percent: number }) {
  const isHealthy = percent > 50;
  const isWarning = percent > 20 && percent <= 50;
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${
            isHealthy ? "bg-emerald-500" : isWarning ? "bg-amber-500" : "bg-red-500"
          }`}
          style={{ width: `${Math.max(0, Math.min(100, percent))}%` }}
        />
      </div>
      <span className={`text-xs font-medium w-10 text-right ${
        isHealthy ? "text-emerald-500" : isWarning ? "text-amber-500" : "text-red-500"
      }`}>
        {percent.toFixed(0)}%
      </span>
    </div>
  );
}

// 对比指标组件
function CompareMetric({ label, current, previous, unit, inverse = false }: {
  label: string;
  current: number;
  previous: number;
  unit: string;
  inverse?: boolean;
}) {
  const diff = current - previous;
  const percentDiff = previous !== 0 ? (diff / previous) * 100 : 0;
  const isImproved = inverse ? diff < 0 : diff > 0;
  const isWorsened = inverse ? diff > 0 : diff < 0;

  return (
    <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
      <div className="text-xs text-muted mb-1">{label}</div>
      <div className="flex items-end gap-2">
        <span className="text-lg font-bold text-default">{current.toFixed(2)}{unit}</span>
        <div className={`flex items-center text-xs ${isImproved ? "text-emerald-500" : isWorsened ? "text-red-500" : "text-gray-400"}`}>
          {isImproved ? <ArrowUpRight className="w-3 h-3" /> : isWorsened ? <ArrowDownRight className="w-3 h-3" /> : <Minus className="w-3 h-3" />}
          <span>{Math.abs(percentDiff).toFixed(1)}%</span>
        </div>
      </div>
      <div className="text-xs text-muted mt-0.5">上周期: {previous.toFixed(2)}{unit}</div>
    </div>
  );
}

// 格式化请求数
function formatNumber(num: number): string {
  return num.toLocaleString();
}

// 域名卡片
function DomainCard({ domain, expanded, onToggle, timeRange, onEditTargets, globalTopology }: {
  domain: DomainSLO;
  expanded: boolean;
  onToggle: () => void;
  timeRange: TimeRange;
  onEditTargets: () => void;
  globalTopology: ServiceTopology;
}) {
  const [activeTab, setActiveTab] = useState<"overview" | "mesh" | "latency" | "compare">("overview");
  const [meshSelectedServiceId, setMeshSelectedServiceId] = useState<string | null>(null);
  const targets = domain.targets[timeRange];

  // 从全局拓扑提取该域名关联的子拓扑
  const subTopology = useMemo((): ServiceTopology => {
    const nodeSet = new Set(domain.backendServices);
    const nodes = globalTopology.nodes.filter(n => nodeSet.has(n.id));
    const edges = globalTopology.edges.filter(e => nodeSet.has(e.source) && nodeSet.has(e.target));
    return { nodes, edges };
  }, [domain.backendServices, globalTopology]);

  return (
    <div className="border border-[var(--border-color)] rounded-xl overflow-hidden bg-card">
      {/* 域名摘要行 */}
      <button
        onClick={onToggle}
        className="w-full px-4 py-3 flex items-center gap-4 hover:bg-[var(--hover-bg)] transition-colors"
      >
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className={`p-2 rounded-lg ${
            domain.status === "healthy" ? "bg-emerald-500/10" :
            domain.status === "warning" ? "bg-amber-500/10" : "bg-red-500/10"
          }`}>
            <Globe className={`w-4 h-4 ${
              domain.status === "healthy" ? "text-emerald-500" :
              domain.status === "warning" ? "text-amber-500" : "text-red-500"
            }`} />
          </div>
          <div className="text-left min-w-0">
            <div className="flex items-center gap-2">
              {domain.tls && <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">HTTPS</span>}
              <span className="font-medium text-default truncate">{domain.host}</span>
              <StatusBadge status={domain.status} />
            </div>
            <div className="text-xs text-muted flex items-center gap-2 mt-0.5">
              <span>{domain.namespace}/{domain.ingressName}</span>
            </div>
          </div>
        </div>

        <div className="hidden lg:flex items-center gap-5">
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">可用性</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                domain.current.availability >= targets.availability ? "text-emerald-500" : "text-red-500"
              }`}>{domain.current.availability.toFixed(2)}%</span>
              <span className="text-xs text-muted">/ {targets.availability}%</span>
            </div>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">P95 延迟</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                domain.current.p95Latency <= targets.p95Latency ? "text-emerald-500" : "text-amber-500"
              }`}>{domain.current.p95Latency}ms</span>
              <span className="text-xs text-muted">/ {targets.p95Latency}ms</span>
            </div>
          </div>
          <div className="w-28">
            <div className="text-[10px] text-muted mb-0.5">错误率</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                domain.current.errorRate <= targets.errorRate ? "text-emerald-500" : "text-red-500"
              }`}>{domain.current.errorRate.toFixed(2)}%</span>
              <span className="text-xs text-muted">/ {targets.errorRate}%</span>
            </div>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">错误预算</div>
            <ErrorBudgetBar percent={domain.errorBudgetRemaining} />
          </div>
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">吞吐量</div>
            <span className="text-sm font-semibold text-default">{formatNumber(domain.current.requestsPerSec)}/s</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <TrendIcon trend={domain.trend} />
          {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
        </div>
      </button>

      {/* 展开详情 */}
      {expanded && (
        <div className="border-t border-[var(--border-color)]">
          {/* Tab 切换 */}
          <div className="flex items-center gap-1 px-4 pt-3 pb-2 border-b border-[var(--border-color)]">
            {[
              { id: "overview", label: "SLO 概览", icon: Activity },
              { id: "latency",  label: "入口延迟分布", icon: BarChart3 },
              { id: "mesh",     label: "服务网格", icon: Network },
              { id: "compare",  label: "周期对比", icon: Calendar },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as typeof activeTab)}
                className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg transition-colors ${
                  activeTab === tab.id
                    ? "bg-primary/10 text-primary"
                    : "text-muted hover:text-default hover:bg-[var(--hover-bg)]"
                }`}
              >
                <tab.icon className="w-3.5 h-3.5" />
                {tab.label}
              </button>
            ))}
            <div className="flex-1" />
            <button
              onClick={onEditTargets}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg text-muted hover:text-default hover:bg-[var(--hover-bg)] transition-colors"
            >
              <Settings2 className="w-3.5 h-3.5" />
              编辑目标
            </button>
          </div>

          <div className="p-4 bg-[var(--background)]">
            {/* SLO 概览 Tab */}
            {activeTab === "overview" && (
              <div className="space-y-4">
                <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">可用性</div>
                    <div className="text-lg font-bold text-default">{domain.current.availability.toFixed(3)}%</div>
                    <div className="text-xs text-muted mt-1">目标: {targets.availability}%</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">P95 / P99 延迟</div>
                    <div className="text-lg font-bold text-default">{domain.current.p95Latency}ms / {domain.current.p99Latency}ms</div>
                    <div className="text-xs text-muted mt-1">目标 P95: {targets.p95Latency}ms</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">错误率</div>
                    <div className="text-lg font-bold text-default">{domain.current.errorRate.toFixed(3)}%</div>
                    <div className="text-xs text-muted mt-1">目标: {targets.errorRate}%</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">总请求数</div>
                    <div className="text-lg font-bold text-default">{formatNumber(domain.current.totalRequests)}</div>
                    <div className="text-xs text-muted mt-1">{formatNumber(domain.current.requestsPerSec)} req/s</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">错误预算剩余</div>
                    <div className={`text-lg font-bold ${
                      domain.errorBudgetRemaining > 50 ? "text-emerald-500" :
                      domain.errorBudgetRemaining > 20 ? "text-amber-500" : "text-red-500"
                    }`}>{domain.errorBudgetRemaining.toFixed(1)}%</div>
                    <div className="mt-1"><ErrorBudgetBar percent={domain.errorBudgetRemaining} /></div>
                  </div>
                </div>
                {/* SLO 趋势图 + 错误预算消耗图 */}
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                  <HistoryChartView history={domain.history} targets={targets} />
                  <ErrorBudgetBurnView history={domain.history} />
                </div>
              </div>
            )}

            {/* 服务网格 Tab */}
            {activeTab === "mesh" && (
              <div className="space-y-4">
                {/* 该域名的服务拓扑 */}
                {subTopology.nodes.length > 1 && (
                  <ServiceTopologyView
                    topology={subTopology}
                    onSelectNode={(node) => setMeshSelectedServiceId(node.id)}
                  />
                )}
                {/* 服务列表 + 详情双栏 */}
                <ServiceMeshOverview
                  topology={subTopology}
                  selectedServiceId={meshSelectedServiceId}
                  onSelectService={setMeshSelectedServiceId}
                />
              </div>
            )}

            {/* 入口延迟分布 Tab */}
            {activeTab === "latency" && (
              <div className="space-y-4">
              <LatencyDistributionView
                buckets={domain.latencyDistribution}
                p50={domain.current.p50Latency}
                p95={domain.current.p95Latency}
                p99={domain.current.p99Latency}
                requestBreakdown={domain.requestBreakdown}
                sourceLabel="Traefik 入口延迟分布"
              />
              <StatusCodeBreakdownView breakdown={domain.statusCodeBreakdown} />
              </div>
            )}

            {/* 周期对比 Tab */}
            {activeTab === "compare" && (
              <div className="space-y-4">
                <div className="flex items-center gap-2 text-xs text-muted">
                  <Calendar className="w-4 h-4" />
                  <span>本周期 vs 上周期对比</span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <CompareMetric label="可用性" current={domain.current.availability} previous={domain.previous.availability} unit="%" inverse={false} />
                  <CompareMetric label="P95 延迟" current={domain.current.p95Latency} previous={domain.previous.p95Latency} unit="ms" inverse={true} />
                  <CompareMetric label="错误率" current={domain.current.errorRate} previous={domain.previous.errorRate} unit="%" inverse={true} />
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// 汇总卡片
function SummaryCard({ icon: Icon, label, value, subValue, color }: {
  icon: typeof Activity;
  label: string;
  value: string;
  subValue?: string;
  color: string;
}) {
  return (
    <div className="p-4 rounded-xl bg-card border border-[var(--border-color)]">
      <div className="flex items-center gap-3">
        <div className={`p-2 rounded-lg ${color}`}><Icon className="w-5 h-5" /></div>
        <div>
          <div className="text-xs text-muted">{label}</div>
          <div className="text-xl font-bold text-default">{value}</div>
          {subValue && <div className="text-xs text-muted">{subValue}</div>}
        </div>
      </div>
    </div>
  );
}

// SLO 配置弹窗
function SLOConfigModal({ domain, onClose, onSave, currentTimeRange }: {
  domain: DomainSLO;
  onClose: () => void;
  onSave: (timeRange: TimeRange, targets: SLOTargets) => void;
  currentTimeRange: TimeRange;
}) {
  const [selectedRange, setSelectedRange] = useState<TimeRange>(currentTimeRange);
  const [targets, setTargets] = useState(domain.targets[selectedRange]);

  const handleRangeChange = (range: TimeRange) => {
    setSelectedRange(range);
    setTargets(domain.targets[range]);
  };

  const timeRangeLabels: Record<TimeRange, string> = { "1d": "天", "7d": "周", "30d": "月" };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-md bg-card rounded-2xl shadow-xl border border-[var(--border-color)] overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <Target className="w-5 h-5 text-primary" />
            <h3 className="font-semibold text-default">配置 SLO 目标</h3>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--hover-bg)]">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div className="text-sm text-muted mb-2">
            <Globe className="w-4 h-4 inline mr-1" />{domain.host}
          </div>

          <div>
            <label className="text-sm font-medium text-default mb-2 block">选择周期</label>
            <div className="flex gap-2">
              {(["1d", "7d", "30d"] as TimeRange[]).map((range) => (
                <button
                  key={range}
                  onClick={() => handleRangeChange(range)}
                  className={`flex-1 px-3 py-2 text-sm rounded-lg border transition-colors ${
                    selectedRange === range ? "border-primary bg-primary/10 text-primary" : "border-[var(--border-color)] text-muted hover:text-default"
                  }`}
                >{timeRangeLabels[range]}</button>
              ))}
            </div>
          </div>

          <div>
            <label className="text-sm font-medium text-default">可用性目标 (%)</label>
            <input
              type="number" step="0.01" min="90" max="100" value={targets.availability}
              onChange={(e) => setTargets({ ...targets, availability: parseFloat(e.target.value) })}
              className="w-full mt-1 px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default"
            />
          </div>

          <div>
            <label className="text-sm font-medium text-default">P95 延迟阈值 (ms)</label>
            <input
              type="number" step="10" min="10" max="5000" value={targets.p95Latency}
              onChange={(e) => setTargets({ ...targets, p95Latency: parseInt(e.target.value) })}
              className="w-full mt-1 px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default"
            />
          </div>

          <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted">错误率阈值</span>
              <span className="text-sm font-medium text-default">{(100 - targets.availability).toFixed(2)}%</span>
            </div>
            <div className="text-xs text-muted mt-1">自动计算: 100% - 可用性目标</div>
          </div>
        </div>

        <div className="flex items-center gap-2 p-4 border-t border-[var(--border-color)] bg-[var(--hover-bg)]">
          <button onClick={onClose} className="flex-1 px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-muted hover:text-default transition-colors">取消</button>
          <button
            onClick={() => onSave(selectedRange, { ...targets, errorRate: 100 - targets.availability })}
            className="flex-1 px-4 py-2 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 transition-colors"
          >保存 ({timeRangeLabels[selectedRange]})</button>
        </div>
      </div>
    </div>
  );
}

// ==================== Main Page ====================

export default function StylePreviewPage() {
  const [expandedId, setExpandedId] = useState<string | null>("1");
  const [timeRange, setTimeRange] = useState<TimeRange>("1d");
  const [showConfigModal, setShowConfigModal] = useState<DomainSLO | null>(null);
  const [domains, setDomains] = useState(mockDomainSLOs);

  const summary = useMemo(() => {
    const totalDomains = domains.length;
    const totalServices = mockGlobalTopology.nodes.length;
    const healthyCount = domains.filter(d => d.status === "healthy").length;
    const warningCount = domains.filter(d => d.status === "warning").length;
    const criticalCount = domains.filter(d => d.status === "critical").length;
    const totalRPS = domains.reduce((sum, d) => sum + d.current.requestsPerSec, 0);
    const avgAvailability = domains.reduce((sum, d) => sum + d.current.availability, 0) / totalDomains;
    const avgP95 = domains.reduce((sum, d) => sum + d.current.p95Latency, 0) / totalDomains;

    return { totalDomains, totalServices, healthyCount, warningCount, criticalCount, totalRPS, avgAvailability, avgP95 };
  }, [domains]);

  const handleSaveConfig = (domainId: string, range: TimeRange, newTargets: SLOTargets) => {
    setDomains(prev => prev.map(d =>
      d.id === domainId ? { ...d, targets: { ...d.targets, [range]: newTargets } } : d
    ));
    setShowConfigModal(null);
  };

  return (
    <Layout>
      <div className="-m-6 min-h-[calc(100vh-3.5rem)] bg-[var(--background)]">
        {/* 头部 */}
        <div className="px-6 py-4 border-b border-[var(--border-color)] bg-card">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-xl bg-gradient-to-br from-violet-100 to-indigo-100 dark:from-violet-900/30 dark:to-indigo-900/30">
                <Activity className="w-6 h-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <h1 className="text-lg font-semibold text-default">SLO 服务监控</h1>
                <p className="text-xs text-muted">服务网格拓扑 · per-service 延迟分布 · 域名 SLO</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1 p-1 rounded-lg bg-[var(--hover-bg)]">
                {([{ value: "1d", label: "天" }, { value: "7d", label: "周" }, { value: "30d", label: "月" }] as const).map((range) => (
                  <button
                    key={range.value}
                    onClick={() => setTimeRange(range.value)}
                    className={`px-3 py-1 text-xs rounded-md transition-colors ${
                      timeRange === range.value ? "bg-card text-default shadow-sm" : "text-muted hover:text-default"
                    }`}
                  >{range.label}</button>
                ))}
              </div>
              <button className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors">
                <RefreshCw className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        <div className="p-6 space-y-6">
          {/* 汇总卡片 */}
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            <SummaryCard icon={Server} label="服务总数" value={summary.totalServices.toString()} subValue="Linkerd mesh" color="bg-blue-500/10 text-blue-500" />
            <SummaryCard icon={Globe} label="域名数" value={summary.totalDomains.toString()} subValue={`${summary.healthyCount} 健康`} color="bg-violet-500/10 text-violet-500" />
            <SummaryCard icon={Activity} label="平均可用性" value={`${summary.avgAvailability.toFixed(2)}%`} color="bg-emerald-500/10 text-emerald-500" />
            <SummaryCard icon={Gauge} label="平均 P95" value={`${Math.round(summary.avgP95)}ms`} color="bg-cyan-500/10 text-cyan-500" />
            <SummaryCard icon={Zap} label="总 RPS" value={formatNumber(summary.totalRPS)} subValue="req/s" color="bg-amber-500/10 text-amber-500" />
            <SummaryCard icon={AlertTriangle} label="告警数" value={(summary.warningCount + summary.criticalCount).toString()} subValue={`${summary.criticalCount} 严重`} color={summary.criticalCount > 0 ? "bg-red-500/10 text-red-500" : "bg-amber-500/10 text-amber-500"} />
          </div>

          {/* 域名 SLO 列表 */}
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-semibold text-default">
                域名 SLO 状态
                <span className="ml-2 text-xs font-normal text-muted">({summary.totalDomains} 个域名)</span>
              </h2>
              <button className="text-xs text-primary hover:underline">+ 添加 SLO 目标</button>
            </div>
            <div className="space-y-3">
              {domains.map((domain) => (
                <DomainCard
                  key={domain.id}
                  domain={domain}
                  expanded={expandedId === domain.id}
                  onToggle={() => setExpandedId(expandedId === domain.id ? null : domain.id)}
                  timeRange={timeRange}
                  onEditTargets={() => setShowConfigModal(domain)}
                  globalTopology={mockGlobalTopology}
                />
              ))}
            </div>
          </div>

          {/* 数据来源说明 */}
          <div className="p-4 rounded-xl bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
            <div className="flex items-start gap-3">
              <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-900/50">
                <Activity className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="text-sm">
                <p className="font-medium text-blue-800 dark:text-blue-200 mb-1">数据来源说明</p>
                <p className="text-blue-700 dark:text-blue-300 text-xs leading-relaxed">
                  <strong>服务网格层（Linkerd）：</strong>服务调用拓扑、per-service golden metrics（RPS / 延迟百分位 / 错误率）和 24 桶延迟直方图来源于
                  <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded ml-1">otel_response_total</code> +
                  <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded ml-1">otel_response_latency_ms</code>。
                  mTLS 状态通过 <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded">response_total</code> 的
                  <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded ml-1">tls</code> 标签判断。
                </p>
                <p className="text-blue-700 dark:text-blue-300 text-xs leading-relaxed mt-1">
                  <strong>入口层（Traefik）：</strong>域名级 SLO（可用性 / 延迟 / 错误预算）来源于 Traefik 入口指标。
                  所有数据经 OTel Collector 统一采集，Agent 端完成 per-pod delta 计算与 service 聚合。
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {showConfigModal && (
        <SLOConfigModal
          domain={showConfigModal}
          onClose={() => setShowConfigModal(null)}
          onSave={(range, newTargets) => handleSaveConfig(showConfigModal.id, range, newTargets)}
          currentTimeRange={timeRange}
        />
      )}
    </Layout>
  );
}
