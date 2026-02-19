/**
 * APM (Application Performance Monitoring) API
 *
 * Trace data flows: Agent queries Jaeger -> Master forwards via Command system
 * Current implementation uses mock data (Agent/Master pipeline not yet built)
 */

import { get, post } from "./request";

// ============================================================
// Types
// ============================================================

export interface SpanTag {
  key: string;
  value: string;
}

export interface Span {
  spanId: string;
  parentSpanId: string;
  operationName: string;
  serviceName: string;
  startTime: number; // microseconds
  duration: number; // microseconds
  status: string; // "ok" | "error"
  tags: SpanTag[];
}

export interface TraceDetail {
  traceId: string;
  spans: Span[];
}

export interface TraceSummary {
  traceId: string;
  rootService: string;
  rootOperation: string;
  startTime: number; // microseconds
  duration: number; // microseconds
  spanCount: number;
  serviceCount: number;
  hasError: boolean;
}

export interface TraceService {
  name: string;
  operations: string[];
}

export interface TraceQueryParams {
  cluster_id: string;
  service?: string;
  operation?: string;
  minDuration?: number;
  maxDuration?: number;
  limit?: number;
}

export interface ServiceStats {
  name: string;
  environment: string;
  latencyAvg: number; // μs
  throughput: number; // traces/min
  errorRate: number; // 0-1
  latencyPoints: number[]; // per-trace latency values for sparkline
  errorRatePoints: number[]; // per-trace error flags (0 or 1) for sparkline
}

export interface Dependency {
  name: string;
  latencyAvg: number;
  throughput: number;
  errorRate: number;
  impact: number; // 0-1
}

export interface LatencyBucket {
  rangeStart: number; // μs
  rangeEnd: number;
  count: number;
}

export interface SpanTypeBreakdown {
  type: string; // "HTTP" | "DB" | "other"
  percentage: number;
}

// ============================================================
// API responses
// ============================================================

interface TraceServicesResponse {
  message: string;
  data: TraceService[];
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

// ============================================================
// API methods (will connect to real backend later)
// ============================================================

/**
 * GET /api/v2/apm/services?cluster_id=xxx
 */
export function getTraceServices(params: { cluster_id: string }) {
  return get<TraceServicesResponse>("/api/v2/apm/services", params);
}

/**
 * POST /api/v2/apm/traces/query
 */
export function queryTraces(params: TraceQueryParams) {
  return post<TraceListResponse>("/api/v2/apm/traces/query", params);
}

/**
 * GET /api/v2/apm/traces/{traceId}?cluster_id=xxx
 */
export function getTraceDetail(params: { cluster_id: string; traceId: string }) {
  return get<TraceDetailResponse>(
    `/api/v2/apm/traces/${encodeURIComponent(params.traceId)}`,
    { cluster_id: params.cluster_id }
  );
}
