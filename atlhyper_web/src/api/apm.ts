/**
 * APM (Application Performance Monitoring) API
 *
 * 数据源: ClickHouse otel_traces 表
 * 类型定义: @/types/model/apm
 */

import { get, post } from "./request";
import type {
  APMService,
  TraceSummary,
  TraceDetail,
  Topology,
} from "@/types/model/apm";

// ============================================================
// Query params
// ============================================================

export interface APMServiceParams {
  cluster_id: string;
  namespace?: string;
}

export interface TraceQueryParams {
  cluster_id: string;
  service?: string;
  namespace?: string;
  statusCode?: string;
  minDurationMs?: number;
  maxDurationMs?: number;
  limit?: number;
}

export interface TopologyParams {
  cluster_id: string;
  namespace?: string;
}

// ============================================================
// API responses
// ============================================================

interface APMServicesResponse {
  message: string;
  data: APMService[];
}

interface TraceListResponse {
  message: string;
  data: TraceSummary[];
  total: number;
}

interface TraceDetailResponse {
  message: string;
  data: TraceDetail;
}

interface TopologyResponse {
  message: string;
  data: Topology;
}

// ============================================================
// API methods
// ============================================================

/**
 * GET /api/v2/apm/services
 */
export function getAPMServices(params: APMServiceParams) {
  return get<APMServicesResponse>("/api/v2/apm/services", params);
}

/**
 * POST /api/v2/apm/traces/query
 */
export function queryTraces(params: TraceQueryParams) {
  return post<TraceListResponse>("/api/v2/apm/traces/query", params);
}

/**
 * GET /api/v2/apm/traces/{traceId}
 */
export function getTraceDetail(params: { cluster_id: string; traceId: string }) {
  return get<TraceDetailResponse>(
    `/api/v2/apm/traces/${encodeURIComponent(params.traceId)}`,
    { cluster_id: params.cluster_id }
  );
}

/**
 * GET /api/v2/apm/topology
 */
export function getTopology(params: TopologyParams) {
  return get<TopologyResponse>("/api/v2/apm/topology", params);
}
