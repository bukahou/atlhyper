/**
 * Log (日志查询) API
 *
 * 数据源: ClickHouse otel_logs 表
 * 类型定义: @/types/model/log
 */

import { post } from "./request";
import type { LogQueryResult } from "@/types/model/log";

// ============================================================
// Query params
// ============================================================

export interface LogQueryParams {
  cluster_id: string;
  search?: string;
  services?: string[];
  severities?: string[];
  scopes?: string[];
  limit?: number;
  offset?: number;
}

// ============================================================
// API responses
// ============================================================

interface LogQueryResponse {
  message: string;
  data: LogQueryResult;
}

// ============================================================
// API methods
// ============================================================

/**
 * POST /api/v2/logs/query
 */
export function queryLogs(params: LogQueryParams) {
  return post<LogQueryResponse>("/api/v2/logs/query", params);
}
