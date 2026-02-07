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
  ChevronLeft,
  RefreshCw,
  Settings2,
  Globe,
  Zap,
  Gauge,
  X,
  Download,
  ArrowUpRight,
  ArrowDownRight,
  Calendar,
  FileText,
  Target,
  Network,
  Clock,
  ArrowRight,
  Server,
  Database,
  Shield,
  Box,
} from "lucide-react";

// ==================== Types ====================

// 服务调用节点
interface ServiceNode {
  id: string;
  name: string;
  namespace: string;
  type: "gateway" | "service" | "database" | "cache" | "external";
  avgLatency: number;
  requestCount: number;
  errorRate: number;
  status: "healthy" | "warning" | "critical";
}

// 服务调用边
interface ServiceEdge {
  source: string;
  target: string;
  requestCount: number;
  avgLatency: number;
  errorRate: number;
}

// 服务调用拓扑
interface ServiceTopology {
  nodes: ServiceNode[];
  edges: ServiceEdge[];
}

// 调用链路追踪
interface TraceSpan {
  id: string;
  serviceName: string;
  operationName: string;
  startTime: number; // 相对于 trace 开始的毫秒数
  duration: number;  // 持续时间毫秒
  status: "success" | "error";
  children?: TraceSpan[];
}

// Trace 摘要（列表用）
interface TraceSummary {
  traceId: string;
  method: string;
  path: string;
  duration: number;
  timestamp: string;
  status: "success" | "error";
  spanCount: number;
  spans: TraceSpan[];
}

// 历史数据点
interface HistoryPoint {
  timestamp: string;
  availability: number;
  p95Latency: number;
  p99Latency: number;
  errorRate: number;
  rps: number;
  errorBudgetRemaining: number;
}

// SLO 目标
interface SLOTargets {
  availability: number;
  p95Latency: number;
  errorRate: number;
}

// 时间周期类型
type TimeRange = "1d" | "7d" | "30d";

// 域名级别的 SLO 数据
interface DomainSLO {
  id: string;
  host: string;
  ingressName: string;
  ingressClass: string;
  namespace: string;
  tls: boolean;
  targets: {
    "1d": SLOTargets;
    "7d": SLOTargets;
    "30d": SLOTargets;
  };
  current: {
    availability: number;
    p95Latency: number;
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
  topology: ServiceTopology;
  traces: TraceSummary[];
}

// ==================== Mock Data ====================

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

// Mock 服务拓扑数据
const mockTopologies: Record<string, ServiceTopology> = {
  "api.example.com": {
    nodes: [
      { id: "traefik", name: "Traefik", namespace: "kube-system", type: "gateway", avgLatency: 2, requestCount: 50000, errorRate: 0.01, status: "healthy" },
      { id: "api-gateway", name: "api-gateway", namespace: "production", type: "service", avgLatency: 15, requestCount: 50000, errorRate: 0.05, status: "healthy" },
      { id: "user-service", name: "user-service", namespace: "production", type: "service", avgLatency: 25, requestCount: 20000, errorRate: 0.1, status: "healthy" },
      { id: "order-service", name: "order-service", namespace: "production", type: "service", avgLatency: 45, requestCount: 15000, errorRate: 0.2, status: "warning" },
      { id: "payment-service", name: "payment-service", namespace: "production", type: "service", avgLatency: 120, requestCount: 8000, errorRate: 0.05, status: "healthy" },
      { id: "postgres", name: "PostgreSQL", namespace: "database", type: "database", avgLatency: 5, requestCount: 30000, errorRate: 0.01, status: "healthy" },
      { id: "redis", name: "Redis", namespace: "cache", type: "cache", avgLatency: 1, requestCount: 80000, errorRate: 0, status: "healthy" },
    ],
    edges: [
      { source: "traefik", target: "api-gateway", requestCount: 50000, avgLatency: 2, errorRate: 0.01 },
      { source: "api-gateway", target: "user-service", requestCount: 20000, avgLatency: 15, errorRate: 0.05 },
      { source: "api-gateway", target: "order-service", requestCount: 15000, avgLatency: 20, errorRate: 0.1 },
      { source: "order-service", target: "payment-service", requestCount: 8000, avgLatency: 25, errorRate: 0.05 },
      { source: "user-service", target: "postgres", requestCount: 15000, avgLatency: 5, errorRate: 0.01 },
      { source: "order-service", target: "postgres", requestCount: 12000, avgLatency: 5, errorRate: 0.02 },
      { source: "user-service", target: "redis", requestCount: 40000, avgLatency: 1, errorRate: 0 },
      { source: "order-service", target: "redis", requestCount: 30000, avgLatency: 1, errorRate: 0 },
    ],
  },
  "pay.example.com": {
    nodes: [
      { id: "traefik", name: "Traefik", namespace: "kube-system", type: "gateway", avgLatency: 2, requestCount: 10000, errorRate: 0.01, status: "healthy" },
      { id: "payment-gateway", name: "payment-gateway", namespace: "finance", type: "service", avgLatency: 20, requestCount: 10000, errorRate: 0.02, status: "healthy" },
      { id: "payment-core", name: "payment-core", namespace: "finance", type: "service", avgLatency: 80, requestCount: 10000, errorRate: 0.05, status: "warning" },
      { id: "risk-engine", name: "risk-engine", namespace: "finance", type: "service", avgLatency: 30, requestCount: 10000, errorRate: 0.01, status: "healthy" },
      { id: "mysql", name: "MySQL", namespace: "database", type: "database", avgLatency: 8, requestCount: 25000, errorRate: 0.01, status: "healthy" },
    ],
    edges: [
      { source: "traefik", target: "payment-gateway", requestCount: 10000, avgLatency: 2, errorRate: 0.01 },
      { source: "payment-gateway", target: "payment-core", requestCount: 10000, avgLatency: 20, errorRate: 0.02 },
      { source: "payment-gateway", target: "risk-engine", requestCount: 10000, avgLatency: 15, errorRate: 0.01 },
      { source: "payment-core", target: "mysql", requestCount: 20000, avgLatency: 8, errorRate: 0.01 },
      { source: "risk-engine", target: "mysql", requestCount: 5000, avgLatency: 5, errorRate: 0.01 },
    ],
  },
};

// Mock 调用链路数据（多个 trace）
const mockTraces: Record<string, TraceSummary[]> = {
  "api.example.com": [
    {
      traceId: "trace-001",
      method: "GET",
      path: "/api/users/123",
      duration: 226,
      timestamp: "2026-02-06T10:32:15Z",
      status: "success",
      spanCount: 8,
      spans: [
        {
          id: "span-1",
          serviceName: "Traefik",
          operationName: "HTTP GET /api/users/123",
          startTime: 0,
          duration: 226,
          status: "success",
          children: [
            {
              id: "span-2",
              serviceName: "api-gateway",
              operationName: "handleRequest",
              startTime: 2,
              duration: 220,
              status: "success",
              children: [
                {
                  id: "span-3",
                  serviceName: "user-service",
                  operationName: "getUser",
                  startTime: 5,
                  duration: 180,
                  status: "success",
                  children: [
                    { id: "span-4", serviceName: "Redis", operationName: "GET user:123", startTime: 8, duration: 2, status: "success" },
                    { id: "span-5", serviceName: "PostgreSQL", operationName: "SELECT * FROM users", startTime: 15, duration: 45, status: "success" },
                    { id: "span-6", serviceName: "Redis", operationName: "SET user:123", startTime: 65, duration: 1, status: "success" },
                  ],
                },
                {
                  id: "span-7",
                  serviceName: "order-service",
                  operationName: "getRecentOrders",
                  startTime: 190,
                  duration: 25,
                  status: "success",
                  children: [
                    { id: "span-8", serviceName: "Redis", operationName: "GET orders:user:123", startTime: 192, duration: 1, status: "success" },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
    {
      traceId: "trace-002",
      method: "POST",
      path: "/api/orders",
      duration: 380,
      timestamp: "2026-02-06T10:31:42Z",
      status: "success",
      spanCount: 6,
      spans: [
        {
          id: "span-1",
          serviceName: "Traefik",
          operationName: "HTTP POST /api/orders",
          startTime: 0,
          duration: 380,
          status: "success",
          children: [
            {
              id: "span-2",
              serviceName: "api-gateway",
              operationName: "handleRequest",
              startTime: 2,
              duration: 375,
              status: "success",
              children: [
                {
                  id: "span-3",
                  serviceName: "order-service",
                  operationName: "createOrder",
                  startTime: 5,
                  duration: 365,
                  status: "success",
                  children: [
                    { id: "span-4", serviceName: "PostgreSQL", operationName: "INSERT INTO orders", startTime: 10, duration: 35, status: "success" },
                    { id: "span-5", serviceName: "Redis", operationName: "DEL orders:user:*", startTime: 50, duration: 2, status: "success" },
                    { id: "span-6", serviceName: "payment-service", operationName: "reservePayment", startTime: 55, duration: 280, status: "success" },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
    {
      traceId: "trace-003",
      method: "GET",
      path: "/api/products/456",
      duration: 95,
      timestamp: "2026-02-06T10:30:58Z",
      status: "success",
      spanCount: 4,
      spans: [
        {
          id: "span-1",
          serviceName: "Traefik",
          operationName: "HTTP GET /api/products/456",
          startTime: 0,
          duration: 95,
          status: "success",
          children: [
            {
              id: "span-2",
              serviceName: "api-gateway",
              operationName: "handleRequest",
              startTime: 2,
              duration: 90,
              status: "success",
              children: [
                {
                  id: "span-3",
                  serviceName: "product-service",
                  operationName: "getProduct",
                  startTime: 5,
                  duration: 82,
                  status: "success",
                  children: [
                    { id: "span-4", serviceName: "Redis", operationName: "GET product:456", startTime: 8, duration: 1, status: "success" },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
    {
      traceId: "trace-004",
      method: "DELETE",
      path: "/api/users/789",
      duration: 520,
      timestamp: "2026-02-06T10:29:30Z",
      status: "error",
      spanCount: 5,
      spans: [
        {
          id: "span-1",
          serviceName: "Traefik",
          operationName: "HTTP DELETE /api/users/789",
          startTime: 0,
          duration: 520,
          status: "error",
          children: [
            {
              id: "span-2",
              serviceName: "api-gateway",
              operationName: "handleRequest",
              startTime: 2,
              duration: 515,
              status: "error",
              children: [
                {
                  id: "span-3",
                  serviceName: "user-service",
                  operationName: "deleteUser",
                  startTime: 5,
                  duration: 505,
                  status: "error",
                  children: [
                    { id: "span-4", serviceName: "PostgreSQL", operationName: "DELETE FROM users", startTime: 10, duration: 480, status: "error" },
                    { id: "span-5", serviceName: "Redis", operationName: "DEL user:789", startTime: 495, duration: 2, status: "success" },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
  ],
  "pay.example.com": [
    {
      traceId: "trace-101",
      method: "POST",
      path: "/api/pay",
      duration: 125,
      timestamp: "2026-02-06T10:32:00Z",
      status: "success",
      spanCount: 7,
      spans: [
        {
          id: "span-1",
          serviceName: "Traefik",
          operationName: "HTTP POST /api/pay",
          startTime: 0,
          duration: 125,
          status: "success",
          children: [
            {
              id: "span-2",
              serviceName: "payment-gateway",
              operationName: "processPayment",
              startTime: 2,
              duration: 120,
              status: "success",
              children: [
                {
                  id: "span-3",
                  serviceName: "risk-engine",
                  operationName: "checkRisk",
                  startTime: 5,
                  duration: 28,
                  status: "success",
                  children: [
                    { id: "span-4", serviceName: "MySQL", operationName: "SELECT risk_rules", startTime: 8, duration: 5, status: "success" },
                  ],
                },
                {
                  id: "span-5",
                  serviceName: "payment-core",
                  operationName: "executePayment",
                  startTime: 38,
                  duration: 80,
                  status: "success",
                  children: [
                    { id: "span-6", serviceName: "MySQL", operationName: "INSERT transaction", startTime: 42, duration: 12, status: "success" },
                    { id: "span-7", serviceName: "MySQL", operationName: "UPDATE account", startTime: 58, duration: 8, status: "success" },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
    {
      traceId: "trace-102",
      method: "POST",
      path: "/api/refund",
      duration: 340,
      timestamp: "2026-02-06T10:28:15Z",
      status: "error",
      spanCount: 5,
      spans: [
        {
          id: "span-1",
          serviceName: "Traefik",
          operationName: "HTTP POST /api/refund",
          startTime: 0,
          duration: 340,
          status: "error",
          children: [
            {
              id: "span-2",
              serviceName: "payment-gateway",
              operationName: "processRefund",
              startTime: 2,
              duration: 335,
              status: "error",
              children: [
                {
                  id: "span-3",
                  serviceName: "payment-core",
                  operationName: "executeRefund",
                  startTime: 5,
                  duration: 325,
                  status: "error",
                  children: [
                    { id: "span-4", serviceName: "MySQL", operationName: "SELECT transaction", startTime: 10, duration: 8, status: "success" },
                    { id: "span-5", serviceName: "MySQL", operationName: "UPDATE balance", startTime: 25, duration: 295, status: "error" },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
  ],
};

const mockDomainSLOs: DomainSLO[] = [
  {
    id: "1",
    host: "api.example.com",
    ingressName: "api-gateway",
    ingressClass: "traefik",
    namespace: "production",
    tls: true,
    targets: {
      "1d": { availability: 95, p95Latency: 300, errorRate: 5 },
      "7d": { availability: 96, p95Latency: 280, errorRate: 4 },
      "30d": { availability: 97, p95Latency: 250, errorRate: 3 },
    },
    current: { availability: 99.74, p95Latency: 226, p99Latency: 520, errorRate: 0.26, requestsPerSec: 1790, totalRequests: 77600000 },
    previous: { availability: 99.82, p95Latency: 198, errorRate: 0.18 },
    errorBudgetRemaining: 65,
    status: "warning",
    trend: "stable",
    history: generateHistory(7, 99.74, 226),
    topology: mockTopologies["api.example.com"],
    traces: mockTraces["api.example.com"],
  },
  {
    id: "2",
    host: "pay.example.com",
    ingressName: "payment-gateway",
    ingressClass: "traefik",
    namespace: "finance",
    tls: true,
    targets: {
      "1d": { availability: 99, p95Latency: 100, errorRate: 1 },
      "7d": { availability: 99.5, p95Latency: 100, errorRate: 0.5 },
      "30d": { availability: 99.9, p95Latency: 100, errorRate: 0.1 },
    },
    current: { availability: 99.92, p95Latency: 125, p99Latency: 245, errorRate: 0.08, requestsPerSec: 95, totalRequests: 4120000 },
    previous: { availability: 99.96, p95Latency: 95, errorRate: 0.04 },
    errorBudgetRemaining: 12,
    status: "critical",
    trend: "down",
    history: generateHistory(7, 99.92, 125),
    topology: mockTopologies["pay.example.com"],
    traces: mockTraces["pay.example.com"],
  },
  {
    id: "3",
    host: "www.example.com",
    ingressName: "frontend",
    ingressClass: "traefik",
    namespace: "production",
    tls: true,
    targets: {
      "1d": { availability: 95, p95Latency: 500, errorRate: 5 },
      "7d": { availability: 96, p95Latency: 450, errorRate: 4 },
      "30d": { availability: 97, p95Latency: 400, errorRate: 3 },
    },
    current: { availability: 99.95, p95Latency: 292, p99Latency: 546, errorRate: 0.05, requestsPerSec: 4250, totalRequests: 184200000 },
    previous: { availability: 99.91, p95Latency: 310, errorRate: 0.09 },
    errorBudgetRemaining: 85,
    status: "healthy",
    trend: "up",
    history: generateHistory(7, 99.95, 292),
    topology: mockTopologies["api.example.com"],
    traces: mockTraces["api.example.com"],
  },
];

// ==================== Components ====================

// 服务节点图标
function ServiceIcon({ type }: { type: ServiceNode["type"] }) {
  const iconClass = "w-4 h-4";
  switch (type) {
    case "gateway": return <Shield className={iconClass} />;
    case "service": return <Box className={iconClass} />;
    case "database": return <Database className={iconClass} />;
    case "cache": return <Zap className={iconClass} />;
    case "external": return <Globe className={iconClass} />;
  }
}

// 服务节点颜色
function getNodeColor(status: ServiceNode["status"]) {
  switch (status) {
    case "healthy": return { bg: "bg-emerald-500/10", border: "border-emerald-500/30", text: "text-emerald-600 dark:text-emerald-400" };
    case "warning": return { bg: "bg-amber-500/10", border: "border-amber-500/30", text: "text-amber-600 dark:text-amber-400" };
    case "critical": return { bg: "bg-red-500/10", border: "border-red-500/30", text: "text-red-600 dark:text-red-400" };
  }
}

// 节点位置状态
interface NodePosition {
  id: string;
  x: number;
  y: number;
  node: ServiceNode;
}

// 服务拓扑图组件（圆形节点 + 可拖拽）
function ServiceTopologyView({ topology, onSelectNode }: { topology: ServiceTopology; onSelectNode?: (node: ServiceNode) => void }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [draggingNode, setDraggingNode] = useState<string | null>(null);
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
  const [containerWidth, setContainerWidth] = useState(1000);
  const [timeRange, setTimeRange] = useState<"15m" | "1h" | "1d">("15m");

  // 节点半径
  const nodeRadius = 32;
  const svgHeight = 500;

  // 监听容器宽度
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

  // 根据调用链拓扑排序计算节点层级
  const nodeLayers = useMemo(() => {
    const nodeIds = topology.nodes.map(n => n.id);
    const inDegree: Record<string, number> = {};
    const outDegree: Record<string, number> = {};
    const outEdges: Record<string, string[]> = {};
    const inEdges: Record<string, string[]> = {};

    // 初始化
    nodeIds.forEach(id => {
      inDegree[id] = 0;
      outDegree[id] = 0;
      outEdges[id] = [];
      inEdges[id] = [];
    });

    // 计算入度和出度
    topology.edges.forEach(edge => {
      if (nodeIds.includes(edge.source) && nodeIds.includes(edge.target)) {
        inDegree[edge.target]++;
        outDegree[edge.source]++;
        outEdges[edge.source].push(edge.target);
        inEdges[edge.target].push(edge.source);
      }
    });

    // BFS 拓扑排序计算层级
    const level: Record<string, number> = {};
    const queue: string[] = [];

    // 入口节点（入度为0）从第0层开始
    nodeIds.forEach(id => {
      if (inDegree[id] === 0) {
        level[id] = 0;
        queue.push(id);
      }
    });

    // BFS
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

    // 处理没有入边的孤立节点
    nodeIds.forEach(id => {
      if (level[id] === undefined) {
        level[id] = 0;
      }
    });

    // 按层级分组
    const maxLevel = Math.max(...Object.values(level));
    const layers: string[][] = Array.from({ length: maxLevel + 1 }, () => []);
    nodeIds.forEach(id => {
      layers[level[id]].push(id);
    });

    return layers;
  }, [topology]);

  // 初始化节点位置
  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({});
  const [initialized, setInitialized] = useState(false);

  // 根据拓扑层级计算初始位置
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

  // 节点颜色 - 深色扁平风格
  const getNodeColors = (type: ServiceNode["type"]) => {
    const colors: Record<ServiceNode["type"], {
      fill: string;
      stroke: string;
      icon: string;
      light: string;
    }> = {
      gateway: {
        fill: "#7c3aed",
        stroke: "#6d28d9",
        icon: "#ffffff",
        light: "#a78bfa"
      },
      service: {
        fill: "#0891b2",
        stroke: "#0e7490",
        icon: "#ffffff",
        light: "#22d3ee"
      },
      database: {
        fill: "#d97706",
        stroke: "#b45309",
        icon: "#ffffff",
        light: "#fbbf24"
      },
      cache: {
        fill: "#dc2626",
        stroke: "#b91c1c",
        icon: "#ffffff",
        light: "#f87171"
      },
      external: {
        fill: "#4b5563",
        stroke: "#374151",
        icon: "#ffffff",
        light: "#9ca3af"
      },
    };
    return colors[type];
  };

  // 计算贝塞尔曲线
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

  // 边标签位置
  const getEdgeLabelPos = (sourceId: string, targetId: string) => {
    const source = positions[sourceId];
    const target = positions[targetId];
    if (!source || !target) return { x: 0, y: 0 };
    return {
      x: (source.x + target.x) / 2,
      y: (source.y + target.y) / 2,
    };
  };

  // 拖拽处理
  const handleMouseDown = (e: React.MouseEvent, nodeId: string) => {
    if (e.button !== 0) return; // 只响应左键
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

    setPositions(prev => ({
      ...prev,
      [draggingNode]: { x: newX, y: newY },
    }));
  };

  const handleMouseUp = () => {
    setDraggingNode(null);
  };

  const selectedNodeData = selectedNode ? topology.nodes.find(n => n.id === selectedNode) : null;

  return (
    <div className="p-4 bg-[var(--hover-bg)] rounded-lg">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <Network className="w-4 h-4 text-primary" />
            <span className="text-sm font-medium text-default">服务调用拓扑</span>
          </div>
          <span className="text-[10px] px-2 py-0.5 rounded-full bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400">
            P95 数据
          </span>
          <div className="flex items-center gap-1 p-0.5 rounded-lg bg-slate-200/50 dark:bg-slate-700/50">
            {[
              { value: "15m", label: "15分钟" },
              { value: "1h", label: "1小时" },
              { value: "1d", label: "1天" },
            ].map(opt => (
              <button
                key={opt.value}
                onClick={() => setTimeRange(opt.value as typeof timeRange)}
                className={`px-2 py-0.5 text-[10px] rounded transition-colors ${
                  timeRange === opt.value
                    ? "bg-white dark:bg-slate-600 text-default shadow-sm"
                    : "text-muted hover:text-default"
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-4 text-[11px]">
          {[
            { label: "Gateway", color: "#7c3aed" },
            { label: "Service", color: "#0891b2" },
            { label: "Database", color: "#d97706" },
            { label: "Cache", color: "#dc2626" },
          ].map(item => (
            <div key={item.label} className="flex items-center gap-1.5">
              <span
                className="w-3 h-3 rounded-full border-2"
                style={{ backgroundColor: item.color, borderColor: item.color }}
              />
              <span className="text-muted">{item.label}</span>
            </div>
          ))}
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
            {/* 箭头 */}
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

            {/* 扁平阴影 */}
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
                                    selectedNode === edge.source || selectedNode === edge.target;
              const markerId = isHighlighted
                ? (edge.errorRate > 1 ? "arrow-red" : edge.errorRate > 0.1 ? "arrow-amber" : "arrow-blue")
                : "arrow-gray";

              let strokeColor = "#cbd5e1";
              if (edge.errorRate > 1) strokeColor = isHighlighted ? "#ef4444" : "#fca5a5";
              else if (edge.errorRate > 0.1) strokeColor = isHighlighted ? "#f59e0b" : "#fcd34d";
              else if (isHighlighted) strokeColor = "#0ea5e9";

              const labelPos = getEdgeLabelPos(edge.source, edge.target);

              return (
                <g key={idx}>
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
                      <rect x="-28" y="-10" width="56" height="20" rx="4" fill="white" className="dark:fill-slate-800" />
                      <text textAnchor="middle" y="4" className="text-[10px] font-medium fill-slate-600 dark:fill-slate-300">
                        {edge.avgLatency}ms
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

              const colors = getNodeColors(node.type);
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

                  {/* 主圆 - 深色扁平 */}
                  <circle
                    r={nodeRadius}
                    fill={colors.fill}
                    stroke={colors.stroke}
                    strokeWidth={2}
                    filter="url(#shadow)"
                  />

                  {/* 图标 */}
                  <g className="text-white" style={{ transform: "translate(-10px, -10px)" }}>
                    {node.type === "gateway" && (
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/>
                      </svg>
                    )}
                    {node.type === "service" && (
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
                      </svg>
                    )}
                    {node.type === "database" && (
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <ellipse cx="12" cy="5" rx="9" ry="3"/>
                        <path d="M3 5V19a9 3 0 0 0 18 0V5"/>
                        <path d="M3 12a9 3 0 0 0 18 0"/>
                      </svg>
                    )}
                    {node.type === "cache" && (
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
                      </svg>
                    )}
                  </g>

                  {/* 服务名 */}
                  <text
                    y={nodeRadius + 16}
                    textAnchor="middle"
                    className="text-[11px] font-semibold fill-slate-700 dark:fill-slate-200 pointer-events-none"
                  >
                    {node.name.length > 12 ? node.name.slice(0, 12) + "…" : node.name}
                  </text>

                  {/* 指标 */}
                  <text
                    y={nodeRadius + 28}
                    textAnchor="middle"
                    className="text-[9px] fill-slate-500 dark:fill-slate-400 pointer-events-none"
                  >
                    {node.avgLatency}ms · {node.errorRate.toFixed(1)}%
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
                style={{ backgroundColor: getNodeColors(selectedNodeData.type).fill }}
              >
                <ServiceIcon type={selectedNodeData.type} />
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
                <div className="text-xs text-muted mt-0.5">{selectedNodeData.namespace} · {selectedNodeData.type}</div>
              </div>
            </div>
            <div className="grid grid-cols-3 gap-6 text-center">
              <div>
                <div className="text-xl font-bold text-default">{selectedNodeData.avgLatency}<span className="text-xs font-normal text-muted">ms</span></div>
                <div className="text-[10px] text-muted">延迟</div>
              </div>
              <div>
                <div className="text-xl font-bold text-default">{(selectedNodeData.requestCount / 1000).toFixed(1)}<span className="text-xs font-normal text-muted">k</span></div>
                <div className="text-[10px] text-muted">请求</div>
              </div>
              <div>
                <div className={`text-xl font-bold ${selectedNodeData.errorRate > 0.1 ? "text-red-500" : "text-emerald-500"}`}>
                  {selectedNodeData.errorRate.toFixed(2)}<span className="text-xs font-normal">%</span>
                </div>
                <div className="text-[10px] text-muted">错误</div>
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
                  const colors = getNodeColors(otherNode.type);

                  return (
                    <div key={idx} className="flex items-center gap-2 px-2.5 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                      {!isOutgoing && (
                        <>
                          <span
                            className="w-2.5 h-2.5 rounded-full"
                            style={{ backgroundColor: colors.fill }}
                          />
                          <span className="font-medium text-default">{otherNode.name}</span>
                        </>
                      )}
                      <ArrowRight className={`w-3 h-3 ${isOutgoing ? "text-cyan-600" : "text-slate-400"}`} />
                      {isOutgoing && (
                        <>
                          <span
                            className="w-2.5 h-2.5 rounded-full"
                            style={{ backgroundColor: colors.fill }}
                          />
                          <span className="font-medium text-default">{otherNode.name}</span>
                        </>
                      )}
                      <span className="text-muted">{edge.avgLatency}ms</span>
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

// 调用链路时序图组件（Kibana 风格 - 时间柱状图 + 拖拽选择）
type TimeRangeOption = "15m" | "1h" | "1d";
const timeRangeLabels: Record<TimeRangeOption, string> = { "15m": "15 分钟", "1h": "1 小时", "1d": "1 天" };
const timeRangeDurations: Record<TimeRangeOption, number> = {
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "1d": 24 * 60 * 60 * 1000,
};

function CallTimelineView({ traces }: { traces: TraceSummary[] }) {
  const chartRef = useRef<HTMLDivElement>(null);
  const [expandedSpans, setExpandedSpans] = useState<Set<string>>(new Set(["span-1", "span-2", "span-3"]));
  const [selection, setSelection] = useState<{ start: number; end: number } | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 1; // 每次显示1个链路
  const [timeRange, setTimeRange] = useState<TimeRangeOption>("15m");

  // 生成延迟分布直方图数据（按延迟分桶，X轴=延迟，Y轴=数量）
  const { buckets, maxCount, minLatency, maxLatency, p95, allRequests } = useMemo(() => {
    const now = Date.now();
    const duration = timeRangeDurations[timeRange];
    const start = now - duration;

    // 模拟请求数据
    const requests: { time: number; latency: number; trace: TraceSummary }[] = [];
    traces.forEach(trace => {
      for (let i = 0; i < 15 + Math.floor(Math.random() * 20); i++) {
        // 延迟分布：大部分在基础值附近，少数高延迟
        const baseLatency = trace.duration;
        const rand = Math.random();
        let latency: number;
        if (rand < 0.7) {
          latency = baseLatency * (0.5 + Math.random() * 0.5); // 70% 在 50%-100%
        } else if (rand < 0.95) {
          latency = baseLatency * (1 + Math.random() * 0.8); // 25% 在 100%-180%
        } else {
          latency = baseLatency * (2 + Math.random() * 2); // 5% 高延迟 200%-400%
        }
        requests.push({
          time: start + Math.random() * duration,
          latency: Math.round(latency),
          trace,
        });
      }
    });

    // 计算 P95
    const sortedLatencies = requests.map(r => r.latency).sort((a, b) => a - b);
    const p95Index = Math.floor(sortedLatencies.length * 0.95);
    const p95Value = sortedLatencies[p95Index] || 0;

    // 按延迟分桶
    const minLat = Math.min(...requests.map(r => r.latency));
    const maxLat = Math.max(...requests.map(r => r.latency));
    const bucketCount = 50;
    const latencyRange = maxLat - minLat || 100;
    const bucketSize = latencyRange / bucketCount;

    const bucketData: { latencyStart: number; latencyEnd: number; count: number; requests: typeof requests }[] = [];
    for (let i = 0; i < bucketCount; i++) {
      const bucketStart = minLat + i * bucketSize;
      const bucketEnd = bucketStart + bucketSize;
      const inBucket = requests.filter(r => r.latency >= bucketStart && r.latency < bucketEnd);
      bucketData.push({
        latencyStart: Math.round(bucketStart),
        latencyEnd: Math.round(bucketEnd),
        count: inBucket.length,
        requests: inBucket,
      });
    }

    return {
      buckets: bucketData,
      maxCount: Math.max(...bucketData.map(b => b.count), 1),
      minLatency: Math.round(minLat),
      maxLatency: Math.round(maxLat),
      p95: Math.round(p95Value),
      allRequests: requests,
    };
  }, [traces, timeRange]);

  // 显示的请求列表：有选择范围时显示范围内的，否则显示全部（按时间排序，最新在前）
  const displayRequests = useMemo(() => {
    if (selection) {
      return buckets
        .filter(b => b.latencyEnd >= selection.start && b.latencyStart <= selection.end)
        .flatMap(b => b.requests)
        .filter(r => r.latency >= selection.start && r.latency <= selection.end)
        .sort((a, b) => b.time - a.time);
    }
    // 默认显示全部请求，按时间排序（最新在前）
    return [...allRequests].sort((a, b) => b.time - a.time);
  }, [buckets, selection, allRequests]);

  // 分页后的请求
  const totalPages = Math.max(1, Math.ceil(displayRequests.length / pageSize));
  const paginatedRequests = displayRequests.slice((page - 1) * pageSize, page * pageSize);

  // 选择变化时重置页码
  useEffect(() => {
    setPage(1);
  }, [selection]);

  // 处理拖拽选择
  const handleMouseDown = (e: React.MouseEvent) => {
    if (!chartRef.current) return;
    const rect = chartRef.current.getBoundingClientRect();
    const x = (e.clientX - rect.left) / rect.width;
    const latency = minLatency + x * (maxLatency - minLatency);
    setDragStart(latency);
    setIsDragging(true);
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging || dragStart === null || !chartRef.current) return;
    const rect = chartRef.current.getBoundingClientRect();
    const x = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    const latency = minLatency + x * (maxLatency - minLatency);
    setSelection({
      start: Math.min(dragStart, latency),
      end: Math.max(dragStart, latency),
    });
  };

  const handleMouseUp = () => {
    setIsDragging(false);
    setDragStart(null);
  };

  // 服务颜色
  const getServiceColor = (name: string, status: "success" | "error") => {
    if (status === "error") return "bg-red-500";
    if (name.includes("Traefik")) return "bg-violet-500";
    if (name.includes("gateway")) return "bg-blue-500";
    if (name.includes("service") || name.includes("engine") || name.includes("core")) return "bg-cyan-600";
    if (name.includes("Redis")) return "bg-rose-500";
    if (name.includes("Postgres") || name.includes("MySQL")) return "bg-amber-500";
    return "bg-slate-500";
  };

  // 格式化时间
  // 递归渲染 span (接收 traceDuration 用于计算宽度)
  const renderSpan = (span: TraceSpan, traceDuration: number, depth: number = 0) => {
    const isExpanded = expandedSpans.has(span.id);
    const hasChildren = span.children && span.children.length > 0;
    const widthPercent = (span.duration / traceDuration) * 100;
    const leftPercent = (span.startTime / traceDuration) * 100;

    return (
      <div key={span.id}>
        <div
          className="flex items-center gap-2 py-1 hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
          style={{ paddingLeft: depth * 16 + 4 }}
        >
          <button
            onClick={() => {
              const newSet = new Set(expandedSpans);
              if (isExpanded) newSet.delete(span.id);
              else newSet.add(span.id);
              setExpandedSpans(newSet);
            }}
            className="w-4 h-4 flex items-center justify-center flex-shrink-0"
            disabled={!hasChildren}
          >
            {hasChildren ? (
              isExpanded ? <ChevronDown className="w-3 h-3 text-muted" /> : <ChevronRight className="w-3 h-3 text-muted" />
            ) : (
              <span className="w-1 h-1 rounded-full bg-slate-300 dark:bg-slate-600" />
            )}
          </button>

          <div className="w-28 flex-shrink-0 flex items-center gap-1.5">
            <span className={`w-2 h-2 rounded-full flex-shrink-0 ${getServiceColor(span.serviceName, span.status)}`} />
            <span className="text-[11px] font-medium text-default truncate">{span.serviceName}</span>
          </div>

          <div className="flex-1 h-5 relative bg-slate-100 dark:bg-slate-800 rounded-sm overflow-hidden">
            <div
              className={`absolute h-full ${getServiceColor(span.serviceName, span.status)} rounded-sm flex items-center px-1.5`}
              style={{ left: `${leftPercent}%`, width: `${Math.max(widthPercent, 2)}%`, opacity: 0.9 }}
            >
              {widthPercent > 8 && <span className="text-[9px] text-white font-medium">{span.duration}ms</span>}
            </div>
          </div>

          <div className="w-16 flex-shrink-0 text-right">
            <span className="text-[10px] text-muted">{span.duration}ms</span>
          </div>
        </div>
        {hasChildren && isExpanded && span.children?.map(child => renderSpan(child, traceDuration, depth + 1))}
      </div>
    );
  };

  return (
    <div className="space-y-4">
      {/* 时间序列柱状图 */}
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Clock className="w-4 h-4 text-primary" />
            <span className="text-sm font-medium text-default">Traefik 延迟分布</span>
            {/* 时间范围选择器 */}
            <div className="flex items-center gap-1 bg-slate-100 dark:bg-slate-800 rounded-md p-0.5">
              {(["15m", "1h", "1d"] as TimeRangeOption[]).map(opt => (
                <button
                  key={opt}
                  onClick={() => { setTimeRange(opt); setSelection(null); }}
                  className={`px-2 py-0.5 text-[10px] rounded transition-colors ${
                    timeRange === opt
                      ? "bg-white dark:bg-slate-700 text-default shadow-sm font-medium"
                      : "text-muted hover:text-default"
                  }`}
                >
                  {timeRangeLabels[opt]}
                </button>
              ))}
            </div>
            {/* 总请求数 */}
            <span className="text-[10px] text-muted">
              共 <span className="font-medium text-default">{allRequests.length}</span> 个请求
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
            {/* P95 标签 */}
            <div className="flex items-center gap-1.5 px-2 py-0.5 bg-amber-100 dark:bg-amber-900/30 rounded text-[10px] text-amber-700 dark:text-amber-400 font-medium">
              <span>P95</span>
              <span>{p95}ms</span>
            </div>
          </div>
        </div>

        {/* 柱状图区域 */}
        <div className="px-4 pt-4 pb-6">
          <div className="flex gap-2">
            {/* Y轴刻度 */}
            <div className="flex flex-col justify-between h-32 text-[9px] text-muted text-right w-8 flex-shrink-0">
              <span>{maxCount}</span>
              <span>{Math.round(maxCount / 2)}</span>
              <span>0</span>
            </div>

            {/* 柱状图主体 */}
            <div
              ref={chartRef}
              className="relative h-32 cursor-crosshair select-none flex-1"
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

              {/* 选择区域高亮 - 只显示边框 */}
              {selection && (
                <div
                  className="absolute top-0 bottom-0 border-l-2 border-r-2 border-blue-500 pointer-events-none"
                  style={{
                    left: `${((selection.start - minLatency) / (maxLatency - minLatency)) * 100}%`,
                    width: `${((selection.end - selection.start) / (maxLatency - minLatency)) * 100}%`,
                  }}
                />
              )}

              {/* P95 垂直线 */}
              <div
                className="absolute top-0 bottom-0 w-px bg-amber-500/70 z-20 pointer-events-none"
                style={{ left: `${((p95 - minLatency) / (maxLatency - minLatency)) * 100}%` }}
              >
                <div className="absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-amber-500 text-white text-[8px] font-medium whitespace-nowrap rounded">
                  P95
                </div>
              </div>

              {/* 柱状图 - 延迟分布 */}
              <div className="flex items-end h-full gap-[1px] relative z-10">
                {buckets.map((bucket, idx) => {
                  const heightPercent = (bucket.count / maxCount) * 100;
                  const isInSelection = selection &&
                    bucket.latencyEnd >= selection.start &&
                    bucket.latencyStart <= selection.end;
                  const isNearP95 = bucket.latencyStart <= p95 && bucket.latencyEnd >= p95;

                  return (
                    <div
                      key={idx}
                      className="flex-1 flex flex-col items-center justify-end group relative"
                      style={{ height: "100%" }}
                    >
                      <div
                        className={`w-full max-w-[6px] rounded-t-sm transition-all duration-150 ${
                          isInSelection
                            ? "bg-blue-500"
                            : isNearP95
                              ? "bg-amber-400/90 group-hover:bg-amber-500"
                              : bucket.count > 0
                                ? "bg-teal-400/80 group-hover:bg-teal-500 dark:bg-teal-500/70 dark:group-hover:bg-teal-400"
                                : "bg-transparent"
                        }`}
                        style={{
                          height: bucket.count > 0 ? `${Math.max(heightPercent, 3)}%` : "0",
                          zIndex: isInSelection ? 30 : undefined,
                        }}
                      />
                      {/* Tooltip */}
                      {bucket.count > 0 && (
                        <div className="absolute bottom-full mb-2 hidden group-hover:block z-20 pointer-events-none">
                          <div className="bg-slate-900 text-white text-[10px] px-2.5 py-1.5 rounded-md shadow-xl whitespace-nowrap border border-slate-700">
                            <div className="font-medium">{bucket.latencyStart}-{bucket.latencyEnd}ms</div>
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
                <span>{minLatency}ms</span>
                <span>{Math.round(minLatency + (maxLatency - minLatency) * 0.25)}ms</span>
                <span>{Math.round(minLatency + (maxLatency - minLatency) * 0.5)}ms</span>
                <span>{Math.round(minLatency + (maxLatency - minLatency) * 0.75)}ms</span>
                <span>{maxLatency}ms</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 链路图 */}
      {paginatedRequests.length > 0 && (() => {
        const currentReq = paginatedRequests[0];
        const currentTrace = currentReq.trace;
        const currentDuration = currentTrace.duration;
        return (
          <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
            {/* 头部：分页控制 + 请求信息 */}
            <div className="px-4 py-2.5 border-b border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-800 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="text-xs font-medium text-default">跟踪样例</span>
                {selection ? (
                  <span className="text-[10px] px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded">
                    {Math.round(selection.start)}-{Math.round(selection.end)}ms
                  </span>
                ) : (
                  <span className="text-[10px] text-muted">最新</span>
                )}
                <span className={`w-1.5 h-1.5 rounded-full ${currentTrace.status === "success" ? "bg-emerald-500" : "bg-red-500"}`} />
                <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded ${
                  currentTrace.method === "GET" ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400" :
                  currentTrace.method === "POST" ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" :
                  "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                }`}>
                  {currentTrace.method}
                </span>
                <span className="text-xs text-default">{currentTrace.path}</span>
                <span className="text-xs font-semibold text-default">{currentTrace.duration}ms</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                  className="p-1 rounded hover:bg-slate-200 dark:hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  <ChevronLeft className="w-4 h-4 text-muted" />
                </button>
                <span className="text-xs text-default min-w-[80px] text-center">
                  {page} / {totalPages}
                </span>
                <button
                  onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                  className="p-1 rounded hover:bg-slate-200 dark:hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  <ChevronRight className="w-4 h-4 text-muted" />
                </button>
              </div>
            </div>

            {/* 时间刻度 */}
            <div className="flex items-center gap-2 px-4 py-2 text-[9px] text-muted border-b border-slate-50 dark:border-slate-800/50 bg-slate-50/50 dark:bg-slate-800/30">
              <div className="w-4" />
              <div className="w-28 flex-shrink-0 font-medium">服务</div>
              <div className="flex-1 flex justify-between">
                {[0, 0.25, 0.5, 0.75, 1].map((ratio, i) => (
                  <span key={i}>{Math.round(currentDuration * ratio)}ms</span>
                ))}
              </div>
              <div className="w-16" />
            </div>

            {/* Span 时间线 */}
            <div className="p-2 max-h-[280px] overflow-y-auto">
              {currentTrace.spans.map(span => renderSpan(span, currentDuration))}
            </div>

            {/* 底部图例 */}
            <div className="flex items-center gap-4 px-4 py-2 border-t border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/50">
              {[
                { name: "Gateway", color: "bg-violet-500" },
                { name: "API", color: "bg-blue-500" },
                { name: "Service", color: "bg-cyan-600" },
                { name: "Cache", color: "bg-rose-500" },
                { name: "DB", color: "bg-amber-500" },
              ].map(item => (
                <div key={item.name} className="flex items-center gap-1">
                  <span className={`w-2 h-2 rounded ${item.color}`} />
                  <span className="text-[10px] text-muted">{item.name}</span>
                </div>
              ))}
            </div>
          </div>
        );
      })()}

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
function DomainCard({ domain, expanded, onToggle, timeRange, onEditTargets }: {
  domain: DomainSLO;
  expanded: boolean;
  onToggle: () => void;
  timeRange: TimeRange;
  onEditTargets: () => void;
}) {
  const [activeTab, setActiveTab] = useState<"overview" | "topology" | "trace" | "compare">("overview");
  const targets = domain.targets[timeRange];

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
              <span className="text-gray-400">·</span>
              <span>{domain.ingressClass}</span>
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
              { id: "topology", label: "服务拓扑", icon: Network },
              { id: "trace", label: "链路追踪", icon: Clock },
              { id: "compare", label: "周期对比", icon: Calendar },
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
              </div>
            )}

            {/* 服务拓扑 Tab */}
            {activeTab === "topology" && (
              <ServiceTopologyView topology={domain.topology} />
            )}

            {/* 链路追踪 Tab */}
            {activeTab === "trace" && (
              <CallTimelineView traces={domain.traces} />
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
    const healthyCount = domains.filter(d => d.status === "healthy").length;
    const warningCount = domains.filter(d => d.status === "warning").length;
    const criticalCount = domains.filter(d => d.status === "critical").length;
    const totalRPS = domains.reduce((sum, d) => sum + d.current.requestsPerSec, 0);
    const avgAvailability = domains.reduce((sum, d) => sum + d.current.availability, 0) / totalDomains;
    const avgErrorBudget = domains.reduce((sum, d) => sum + d.errorBudgetRemaining, 0) / totalDomains;

    return { totalDomains, healthyCount, warningCount, criticalCount, totalRPS, avgAvailability, avgErrorBudget };
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
                <p className="text-xs text-muted">域名级 SLO 监控 · 服务拓扑 · 链路追踪</p>
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
            <SummaryCard icon={Globe} label="监控域名" value={summary.totalDomains.toString()} subValue={`${summary.healthyCount} 健康`} color="bg-blue-500/10 text-blue-500" />
            <SummaryCard icon={Activity} label="平均可用性" value={`${summary.avgAvailability.toFixed(2)}%`} color="bg-emerald-500/10 text-emerald-500" />
            <SummaryCard icon={Gauge} label="错误预算剩余" value={`${summary.avgErrorBudget.toFixed(0)}%`} subValue="平均剩余" color={summary.avgErrorBudget > 50 ? "bg-emerald-500/10 text-emerald-500" : summary.avgErrorBudget > 20 ? "bg-amber-500/10 text-amber-500" : "bg-red-500/10 text-red-500"} />
            <SummaryCard icon={Zap} label="总吞吐量" value={formatNumber(summary.totalRPS)} subValue="req/s" color="bg-violet-500/10 text-violet-500" />
            <SummaryCard icon={AlertTriangle} label="告警中" value={summary.warningCount.toString()} subValue="需要关注" color="bg-amber-500/10 text-amber-500" />
            <SummaryCard icon={AlertTriangle} label="严重问题" value={summary.criticalCount.toString()} subValue="需立即处理" color="bg-red-500/10 text-red-500" />
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
                />
              ))}
            </div>
          </div>

          {/* 说明 */}
          <div className="p-4 rounded-xl bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
            <div className="flex items-start gap-3">
              <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-900/50">
                <Activity className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="text-sm">
                <p className="font-medium text-blue-800 dark:text-blue-200 mb-1">数据来源说明</p>
                <p className="text-blue-700 dark:text-blue-300 text-xs leading-relaxed">
                  SLO 指标来源于 Traefik Ingress 流量统计，服务拓扑和链路追踪数据来源于 Linkerd 服务网格。
                  系统采集 <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded">request_total</code>、
                  <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded ml-1">response_latency_ms</code> 等指标计算 SLI/SLO。
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
