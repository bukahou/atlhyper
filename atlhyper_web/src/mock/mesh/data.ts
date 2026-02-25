/**
 * Mesh Mock 数据
 *
 * 严格基于 docs/design/active/clickhouse-otel-data-reference.md
 * 6 个 geass 微服务 (Linkerd mesh) + 调用拓扑 (OTel Traces §4.2)
 *
 * 数据源:
 *   - 节点: Linkerd response_latency_ms (§6.1, §6.3), Traces Span 数 (§3.1)
 *   - 边: Trace 样本 §4.2 的调用关系
 *   - 状态码: Linkerd status_code (§6.3): 200, 404, 500
 *   - 延迟桶: Linkerd le 桶 [1, 2, 4, 10, 40, 40000] ms (§6.3)
 */

import type {
  MeshServiceNode,
  MeshServiceEdge,
  MeshTopologyResponse,
  MeshServiceDetailResponse,
  MeshServiceHistoryPoint,
  MeshStatusCodeBreakdown,
  MeshLatencyBucket,
} from "@/types/mesh";

// ============================================================================
// 真实数据常量
// ============================================================================

// Linkerd le 桶 (§6.3)
const LINKERD_LE_BUCKETS = [1, 2, 4, 10, 40, 40000];

// Namespace
const NS = "geass";

// ============================================================================
// 6 个 Geass 微服务节点 (基于 §3.1 Traces + §6.3 Linkerd)
// ============================================================================

// Span 数 (§3.1): gateway=235, media=237, auth=127, favorites=118, history=114, user=109
// 延迟特征 (§4.2 Trace 样本):
//   gateway: 87ms 总耗时 (聚合多个下游调用)
//   auth: 2ms (Token 验证)
//   media: 26ms (多个 DB 查询, 单个 1-3ms)
//   history: 32ms (DB 查询 2-3ms)
//   favorites: 类似 history
//   user: 登录验证 + DB 写入

const MOCK_NODES: MeshServiceNode[] = [
  {
    id: "geass-gateway",
    name: "geass-gateway",
    namespace: NS,
    rps: 28,       // 网关入口, 最高 RPS
    avgLatency: 42, // 聚合多个下游调用
    p50Latency: 35,
    p95Latency: 87,
    p99Latency: 120,
    errorRate: 0.03,
    availability: 99.97,
    status: "healthy",
    mtlsEnabled: true,
    totalRequests: 2419200, // 28 * 86400
  },
  {
    id: "geass-media",
    name: "geass-media",
    namespace: NS,
    rps: 22,       // 媒体服务 Span 最多
    avgLatency: 8,  // 多个 DB 查询 1-3ms 聚合
    p50Latency: 5,
    p95Latency: 18,
    p99Latency: 45,
    errorRate: 0.01,
    availability: 99.99,
    status: "healthy",
    mtlsEnabled: true,
    totalRequests: 1900800,
  },
  {
    id: "geass-auth",
    name: "geass-auth",
    namespace: NS,
    rps: 26,       // 每个请求都需要 Token 验证
    avgLatency: 2,  // §4.2: Duration 2ms
    p50Latency: 1.5,
    p95Latency: 4,
    p99Latency: 8,
    errorRate: 0.02,
    availability: 99.98,
    status: "healthy",
    mtlsEnabled: true,
    totalRequests: 2246400,
  },
  {
    id: "geass-favorites",
    name: "geass-favorites",
    namespace: NS,
    rps: 8,
    avgLatency: 6,
    p50Latency: 4,
    p95Latency: 15,
    p99Latency: 35,
    errorRate: 0.04,
    availability: 99.96,
    status: "healthy",
    mtlsEnabled: true,
    totalRequests: 691200,
  },
  {
    id: "geass-history",
    name: "geass-history",
    namespace: NS,
    rps: 10,
    avgLatency: 7,  // §4.2: 32ms 包含 DB 查询
    p50Latency: 5,
    p95Latency: 16,
    p99Latency: 40,
    errorRate: 0.02,
    availability: 99.98,
    status: "healthy",
    mtlsEnabled: true,
    totalRequests: 864000,
  },
  {
    id: "geass-user",
    name: "geass-user",
    namespace: NS,
    rps: 3,        // 登录频率低
    avgLatency: 5,
    p50Latency: 3,
    p95Latency: 12,
    p99Latency: 30,
    errorRate: 0.08,
    availability: 99.92,
    status: "healthy",
    mtlsEnabled: true,
    totalRequests: 259200,
  },
];

// ============================================================================
// 调用边 (基于 §4.2 Trace 样本调用拓扑)
//
// gateway → auth   (每个请求先验证 Token)
// gateway → media  (批量获取)
// gateway → history (历史列表)
// gateway → favorites (收藏列表)
// gateway → user   (用户信息/登录)
// history → media  (batch/fetch 获取详情)
// favorites → media (获取媒体详情)
// ============================================================================

const MOCK_EDGES: MeshServiceEdge[] = [
  { source: "geass-gateway", target: "geass-auth", rps: 26, avgLatency: 2, errorRate: 0.02 },
  { source: "geass-gateway", target: "geass-media", rps: 12, avgLatency: 8, errorRate: 0.01 },
  { source: "geass-gateway", target: "geass-history", rps: 8, avgLatency: 7, errorRate: 0.02 },
  { source: "geass-gateway", target: "geass-favorites", rps: 6, avgLatency: 6, errorRate: 0.04 },
  { source: "geass-gateway", target: "geass-user", rps: 2, avgLatency: 5, errorRate: 0.08 },
  { source: "geass-history", target: "geass-media", rps: 5, avgLatency: 4, errorRate: 0.01 },
  { source: "geass-favorites", target: "geass-media", rps: 4, avgLatency: 3, errorRate: 0.01 },
];

export const MOCK_MESH_TOPOLOGY: MeshTopologyResponse = {
  nodes: MOCK_NODES,
  edges: MOCK_EDGES,
};

// ============================================================================
// 服务详情数据
// ============================================================================

/** 生成 Linkerd 延迟桶分布 */
function generateLatencyBuckets(p50: number, p95: number, totalRequests: number): MeshLatencyBucket[] {
  // 基于 Linkerd le 桶 [1, 2, 4, 10, 40, 40000] 构造分布
  const ratios: number[] = LINKERD_LE_BUCKETS.map(le => {
    if (le <= p50 * 0.5) return 0.15;
    if (le <= p50) return 0.35;
    if (le <= p95) return 0.35;
    if (le <= p95 * 3) return 0.10;
    return 0.05;
  });
  const sum = ratios.reduce((a, b) => a + b, 0);
  return LINKERD_LE_BUCKETS.map((le, i) => ({
    le,
    count: Math.round((ratios[i] / sum) * totalRequests),
  }));
}

/** 生成 Linkerd 状态码分布 (§6.3: 200, 404, 500) */
function generateStatusCodes(totalRequests: number, errorRate: number): MeshStatusCodeBreakdown[] {
  const errorCount = Math.round(totalRequests * (errorRate / 100));
  const okCount = totalRequests - errorCount;
  // Linkerd 状态码: 200 (绝大多数), 404 (少量), 500 (极少)
  const code404 = Math.round(errorCount * 0.7);
  const code500 = errorCount - code404;
  return [
    { code: "2xx", count: okCount },
    { code: "4xx", count: code404 },
    { code: "5xx", count: code500 },
  ].filter(s => s.count > 0);
}

/** 生成服务历史数据 (24h) */
function generateServiceHistory(node: MeshServiceNode): MeshServiceHistoryPoint[] {
  const points: MeshServiceHistoryPoint[] = [];
  const now = Date.now();
  for (let i = 24; i >= 0; i--) {
    const jitter = () => (Math.random() - 0.5) * 2;
    points.push({
      timestamp: new Date(now - i * 3600000).toISOString(),
      rps: Math.max(0, node.rps + jitter() * node.rps * 0.2),
      p95Latency: Math.max(1, node.p95Latency + jitter() * node.p95Latency * 0.15),
      errorRate: Math.max(0, node.errorRate + jitter() * 0.05),
      availability: Math.min(100, Math.max(99, node.availability + jitter() * 0.1)),
      mtlsEnabled: node.mtlsEnabled,
    });
  }
  return points;
}

/** 构建单个服务的详情响应 */
function buildServiceDetail(node: MeshServiceNode): MeshServiceDetailResponse {
  const inbound = MOCK_EDGES.filter(e => e.target === node.id);
  const outbound = MOCK_EDGES.filter(e => e.source === node.id);

  return {
    ...node,
    history: generateServiceHistory(node),
    upstreams: inbound,
    downstreams: outbound,
    statusCodes: generateStatusCodes(node.totalRequests, node.errorRate),
    latencyBuckets: generateLatencyBuckets(node.p50Latency, node.p95Latency, node.totalRequests),
  };
}

/** 获取服务详情 (按 namespace + name 查找) */
export function getMockServiceDetail(namespace: string, name: string): MeshServiceDetailResponse | null {
  const node = MOCK_NODES.find(n => n.namespace === namespace && n.name === name);
  if (!node) return null;
  return buildServiceDetail(node);
}
