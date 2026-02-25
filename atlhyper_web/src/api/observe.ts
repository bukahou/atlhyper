/**
 * 可观测性查询 API — ClickHouse 按需查询
 *
 * 通过 Master Command 机制将查询转发给 Agent 执行 ClickHouse 查询
 * 响应格式: { message: string, data: T }
 *
 * 类型对齐:
 *   - Traces: model_v3/apm/trace.go   → types/model/apm.ts
 *   - Logs:   model_v3/log/log.go     → types/model/log.ts
 *   - Metrics: model_v3/metrics/      → types/node-metrics.ts
 */

import { get, post } from "./request";
import type { TraceSummary, TraceDetail, APMService, Topology, OperationStats, APMServiceSeriesResponse, HTTPStats, DBOperationStats } from "@/types/model/apm";
import type { LogEntry, LogFacets, LogHistogramBucket } from "@/types/model/log";
import type { NodeMetrics, Summary, Point } from "@/types/node-metrics";

// ============================================================================
// 通用响应包装
// ============================================================================

interface ObserveResponse<T> {
  message: string;
  data: T;
}

// ============================================================================
// SLO 类型（对齐 model_v3/slo/slo.go）
// ============================================================================

export interface StatusCodeCount {
  code: string;
  count: number;
}

export interface IngressSLO {
  serviceKey: string;
  displayName: string;
  rps: number;
  successRate: number;
  errorRate: number;
  p50Ms: number;
  p90Ms: number;
  p99Ms: number;
  avgMs: number;
  statusCodes: StatusCodeCount[];
  totalRequests: number;
  totalErrors: number;
}

export interface ServiceSLO {
  namespace: string;
  name: string;
  rps: number;
  successRate: number;
  p50Ms: number;
  p90Ms: number;
  p99Ms: number;
  mtlsRate: number;
  statusCodes: StatusCodeCount[];
}

export interface ServiceEdge {
  srcNamespace: string;
  srcName: string;
  dstNamespace: string;
  dstName: string;
  rps: number;
  successRate: number;
  avgMs: number;
}

export interface SLODataPoint {
  timestamp: string;
  rps: number;
  successRate: number;
  p50Ms: number;
  p99Ms: number;
}

export interface SLOTimeSeries {
  namespace?: string;
  name: string;
  points: SLODataPoint[];
}

export interface SLOSummary {
  totalServices: number;
  healthyServices: number;
  warningServices: number;
  criticalServices: number;
  avgSuccessRate: number;
  totalRps: number;
  avgP99Ms: number;
}

// ============================================================================
// Log 查询结果（对齐 model_v3，不含 histogram）
// ============================================================================

export interface LogQueryResponse {
  logs: LogEntry[];
  total: number;
  facets: LogFacets;
  histogram?: LogHistogramBucket[];
}

// ============================================================================
// Metrics API
// ============================================================================

/** 获取指标汇总 */
export function getMetricsSummary(clusterId: string) {
  return get<ObserveResponse<Summary>>("/api/v2/observe/metrics/summary", {
    cluster_id: clusterId,
  });
}

/** 获取所有节点指标列表 */
export function getMetricsNodes(clusterId: string) {
  return get<ObserveResponse<NodeMetrics[]>>("/api/v2/observe/metrics/nodes", {
    cluster_id: clusterId,
  });
}

/** 获取单节点指标 */
export function getMetricsNode(clusterId: string, nodeName: string) {
  return get<ObserveResponse<NodeMetrics>>(
    `/api/v2/observe/metrics/nodes/${encodeURIComponent(nodeName)}`,
    { cluster_id: clusterId }
  );
}

/** 获取节点时序数据 */
export function getMetricsNodeSeries(clusterId: string, nodeName: string, metric: string, minutes?: number) {
  return get<ObserveResponse<Point[]>>(
    `/api/v2/observe/metrics/nodes/${encodeURIComponent(nodeName)}/series`,
    { cluster_id: clusterId, metric, ...(minutes ? { minutes } : {}) }
  );
}

// ============================================================================
// Logs API
// ============================================================================

/** 查询日志 (POST) */
export function queryLogs(params: {
  cluster_id: string;
  query?: string;
  service?: string;
  level?: string;
  scope?: string;
  limit?: number;
  offset?: number;
  since?: string;
  start_time?: string;
  end_time?: string;
}) {
  return post<ObserveResponse<LogQueryResponse>>("/api/v2/observe/logs/query", params);
}

// ============================================================================
// Traces API
// ============================================================================

/** 查询 Trace 列表 */
export function getTracesList(clusterId: string, params?: {
  service?: string;
  operation?: string;
  min_duration?: string;
  max_duration?: string;
  limit?: number;
  offset?: number;
  start_time?: string;
  end_time?: string;
  time_range?: string;
}) {
  return get<ObserveResponse<TraceSummary[]>>("/api/v2/observe/traces", {
    cluster_id: clusterId,
    ...params,
  });
}

/** 获取服务列表 */
export function getTracesServices(clusterId: string, timeRange?: string) {
  return get<ObserveResponse<APMService[]>>("/api/v2/observe/traces/services", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}

/** 获取服务拓扑 */
export function getTracesTopology(clusterId: string, timeRange?: string) {
  return get<ObserveResponse<Topology>>("/api/v2/observe/traces/topology", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}

/** 获取操作级聚合统计 */
export function getTracesOperations(clusterId: string, timeRange?: string) {
  return get<ObserveResponse<OperationStats[]>>("/api/v2/observe/traces/operations", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}

/** 获取服务时序趋势 (Concentrator 预聚合) */
export function getAPMServiceSeries(clusterId: string, serviceName: string, minutes?: number) {
  return get<ObserveResponse<APMServiceSeriesResponse>>(
    `/api/v2/observe/traces/services/${encodeURIComponent(serviceName)}/series`,
    { cluster_id: clusterId, ...(minutes ? { minutes: String(minutes) } : {}) },
  );
}

/** 获取 HTTP 状态码分布 */
export function getTracesHTTPStats(clusterId: string, params: {
  service: string;
  time_range?: string;
}) {
  return get<ObserveResponse<HTTPStats[]>>("/api/v2/observe/traces/stats", {
    cluster_id: clusterId,
    sub_action: "http_stats",
    ...params,
  });
}

/** 获取数据库操作统计 */
export function getTracesDBStats(clusterId: string, params: {
  service: string;
  time_range?: string;
}) {
  return get<ObserveResponse<DBOperationStats[]>>("/api/v2/observe/traces/stats", {
    cluster_id: clusterId,
    sub_action: "db_stats",
    ...params,
  });
}

/** 获取 Trace 详情 */
export function getTraceDetail(clusterId: string, traceId: string) {
  return get<ObserveResponse<TraceDetail>>(
    `/api/v2/observe/traces/${encodeURIComponent(traceId)}`,
    { cluster_id: clusterId }
  );
}

// ============================================================================
// SLO API
// ============================================================================

/** 获取 SLO 汇总 */
export function getSLOSummary(clusterId: string, timeRange?: string) {
  return get<ObserveResponse<SLOSummary>>("/api/v2/observe/slo/summary", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}

/** 获取 Ingress SLO 列表 */
export function getSLOIngress(clusterId: string, timeRange?: string) {
  return get<ObserveResponse<IngressSLO[]>>("/api/v2/observe/slo/ingress", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}

/** 获取服务 SLO 列表 */
export function getSLOServices(clusterId: string, timeRange?: string) {
  return get<ObserveResponse<ServiceSLO[]>>("/api/v2/observe/slo/services", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}

/** 获取服务间调用关系 */
export function getSLOEdges(clusterId: string, timeRange?: string) {
  return get<ObserveResponse<ServiceEdge[]>>("/api/v2/observe/slo/edges", {
    cluster_id: clusterId,
    ...(timeRange ? { time_range: timeRange } : {}),
  });
}

/** 获取 SLO 时序数据 */
export function getSLOTimeSeries(clusterId: string, params?: {
  service?: string;
  time_range?: string;
  interval?: string;
}) {
  return get<ObserveResponse<SLOTimeSeries>>("/api/v2/observe/slo/timeseries", {
    cluster_id: clusterId,
    ...params,
  });
}
