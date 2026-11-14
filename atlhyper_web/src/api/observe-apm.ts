/**
 * APM (Traces) 信号域 API
 *
 * 读函数的最后一个参数是可选的 AbortSignal，用于取消请求。
 */

import { get } from "./request";
import type { ObserveResponse } from "./observe-common";
import type { TraceSummary, TraceDetail, APMService, Topology, OperationStats, APMServiceSeriesResponse, HTTPStats, DBOperationStats, CorrelationResult } from "@/types/model/apm";

/** 时间范围的查询参数：相对范围传 time_range（如 "1h"），绝对范围传起止时间（ISO） */
export interface TraceTimeRangeQuery {
  time_range?: string;
  start_time?: string;
  end_time?: string;
}

/** 链路列表的两种返回形态：快照直读是 { traces, total }，经指令通道读取是数组 */
export type TraceListPayload = TraceSummary[] | { traces?: TraceSummary[]; total?: number };

/** 查询 Trace 列表 */
export function getTracesList(
  clusterId: string,
  params?: TraceTimeRangeQuery & {
    service?: string;
    operation?: string;
    min_duration?: string;
    max_duration?: string;
    limit?: number;
    offset?: number;
    status_code?: string;
    method?: string;
  },
  signal?: AbortSignal,
) {
  return get<ObserveResponse<TraceListPayload>>("/api/v2/observe/traces", { cluster_id: clusterId, ...params }, signal);
}

/** 获取服务列表 */
export function getTracesServices(clusterId: string, range?: TraceTimeRangeQuery, signal?: AbortSignal) {
  return get<ObserveResponse<APMService[]>>("/api/v2/observe/traces/services", { cluster_id: clusterId, ...range }, signal);
}

/** 获取服务拓扑 */
export function getTracesTopology(clusterId: string, range?: TraceTimeRangeQuery, signal?: AbortSignal) {
  return get<ObserveResponse<Topology>>("/api/v2/observe/traces/topology", { cluster_id: clusterId, ...range }, signal);
}

/** 获取操作级聚合统计 */
export function getTracesOperations(clusterId: string, range?: TraceTimeRangeQuery, signal?: AbortSignal) {
  return get<ObserveResponse<OperationStats[]>>("/api/v2/observe/traces/operations", { cluster_id: clusterId, ...range }, signal);
}

/** 获取服务时序趋势 (Concentrator 预聚合) */
export function getAPMServiceSeries(clusterId: string, serviceName: string, minutes?: number, signal?: AbortSignal) {
  return get<ObserveResponse<APMServiceSeriesResponse>>(
    `/api/v2/observe/traces/services/${encodeURIComponent(serviceName)}/series`,
    { cluster_id: clusterId, ...(minutes ? { minutes: String(minutes) } : {}) },
    signal,
  );
}

/** 相关性分析：慢/错请求中显著超标的属性（对齐 ES APM Correlations） */
export function getTracesCorrelations(
  clusterId: string,
  params: TraceTimeRangeQuery & {
    service: string;
    operation?: string;
    mode: "latency" | "failure";
  },
  signal?: AbortSignal,
) {
  return get<ObserveResponse<CorrelationResult>>(
    "/api/v2/observe/traces/correlations",
    { cluster_id: clusterId, ...params },
    signal,
  );
}

/** 获取 HTTP 状态码分布 */
export function getTracesHTTPStats(
  clusterId: string,
  params: TraceTimeRangeQuery & { service: string },
  signal?: AbortSignal,
) {
  return get<ObserveResponse<HTTPStats[]>>(
    "/api/v2/observe/traces/stats",
    { cluster_id: clusterId, sub_action: "http_stats", ...params },
    signal,
  );
}

/** 获取数据库操作统计 */
export function getTracesDBStats(
  clusterId: string,
  params: TraceTimeRangeQuery & { service: string },
  signal?: AbortSignal,
) {
  return get<ObserveResponse<DBOperationStats[]>>(
    "/api/v2/observe/traces/stats",
    { cluster_id: clusterId, sub_action: "db_stats", ...params },
    signal,
  );
}

/** 获取 Trace 详情 */
export function getTraceDetail(clusterId: string, traceId: string, signal?: AbortSignal) {
  return get<ObserveResponse<TraceDetail>>(
    `/api/v2/observe/traces/${encodeURIComponent(traceId)}`,
    { cluster_id: clusterId },
    signal,
  );
}
