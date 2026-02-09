/**
 * 服务网格拓扑相关类型定义
 * 与后端 atlhyper_master_v2/gateway/handler/slo_mesh.go 保持一致
 */

// 服务节点
export interface MeshServiceNode {
  id: string;
  name: string;
  namespace: string;
  rps: number;
  avg_latency: number;
  p50_latency: number;
  p95_latency: number;
  p99_latency: number;
  error_rate: number;
  availability: number;
  status: "healthy" | "warning" | "critical";
  mtls_percent: number;
  total_requests: number;
}

// 服务调用边
export interface MeshServiceEdge {
  source: string;
  target: string;
  rps: number;
  avg_latency: number;
  error_rate: number;
}

// 拓扑响应
export interface MeshTopologyResponse {
  nodes: MeshServiceNode[];
  edges: MeshServiceEdge[];
}

// 服务详情历史点
export interface MeshServiceHistoryPoint {
  timestamp: string;
  rps: number;
  p95_latency: number;
  error_rate: number;
  availability: number;
  mtls_percent: number;
}

// 状态码分布
export interface MeshStatusCodeBreakdown {
  code: string;  // "2xx", "3xx", "4xx", "5xx"
  count: number;
}

// 延迟分布桶
export interface MeshLatencyBucket {
  le: number;    // 上界 (ms)
  count: number; // 该桶内的请求数
}

// 服务详情响应
export interface MeshServiceDetailResponse extends MeshServiceNode {
  history: MeshServiceHistoryPoint[];
  upstreams: MeshServiceEdge[];
  downstreams: MeshServiceEdge[];
  status_codes: MeshStatusCodeBreakdown[];
  latency_buckets: MeshLatencyBucket[];
}

// API 参数
export interface MeshTopologyParams {
  clusterId?: string;
  timeRange?: string;
}

export interface MeshServiceDetailParams {
  clusterId?: string;
  namespace: string;
  name: string;
  timeRange?: string;
}
