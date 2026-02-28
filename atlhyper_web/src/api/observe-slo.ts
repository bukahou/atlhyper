/**
 * SLO 信号域 API
 */

import { get } from "./request";
import type { ObserveResponse } from "./observe-common";

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
