/**
 * Logs 信号域 API
 */

import { get, post } from "./request";
import type { ObserveResponse } from "./observe-common";
import type { LogEntry, LogFacets, LogHistogramResult } from "@/types/model/log";

/** 日志查询结果（不含 histogram） */
export interface LogQueryResponse {
  logs: LogEntry[];
  total: number;
  facets: LogFacets;
}

/** 查询日志 (POST) */
export function queryLogs(params: {
  cluster_id: string;
  query?: string;
  service?: string;
  level?: string;
  scope?: string;
  trace_id?: string;
  span_id?: string;
  limit?: number;
  offset?: number;
  since?: string;
  start_time?: string;
  end_time?: string;
}) {
  return post<ObserveResponse<LogQueryResponse>>("/api/v2/observe/logs/query", params);
}

/** 查询日志直方图 (GET, ClickHouse 聚合) */
export function getLogsHistogram(params: {
  cluster_id: string;
  since?: string;
  service?: string;
  level?: string;
  scope?: string;
  query?: string;
  start_time?: string;
  end_time?: string;
}) {
  return get<ObserveResponse<LogHistogramResult>>("/api/v2/observe/logs/histogram", params);
}
