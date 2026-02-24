/**
 * Logs 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import { mockQueryLogs } from "@/mock/logs";
import type { MockLogQueryParams } from "@/mock/logs";
import * as observeApi from "@/api/observe";
import type { LogQueryResult } from "@/types/model/log";

export type { MockLogQueryParams } from "@/mock/logs";

/** 查询参数（API 模式需要 clusterId） */
export interface LogQueryParams extends MockLogQueryParams {
  clusterId?: string;
  /** Brush 选区开始时间 (epoch ms) */
  startTime?: number;
  /** Brush 选区结束时间 (epoch ms) */
  endTime?: number;
}

/**
 * 查询日志
 *
 * mock 模式: 返回完整 LogQueryResult（含 histogram）
 * api  模式: 调用 Master observe API，histogram 从返回的日志生成
 */
export async function queryLogs(params?: LogQueryParams): Promise<LogQueryResult> {
  if (getDataSourceMode("logs") === "mock" || !params?.clusterId) {
    return mockQueryLogs(params);
  }

  // 映射 timeRange → since duration
  const sinceMap: Record<string, string> = {
    "15min": "15m",
    "1h": "1h",
    "24h": "24h",
    "7d": "168h",
    "15d": "360h",
    "30d": "720h",
  };

  const response = await observeApi.queryLogs({
    cluster_id: params.clusterId,
    query: params.search,
    level: params.severities?.length === 1 ? params.severities[0] : undefined,
    service: params.services?.length === 1 ? params.services[0] : undefined,
    scope: params.scopes?.length === 1 ? params.scopes[0] : undefined,
    limit: params.limit,
    offset: params.offset,
    since: params.timeRange ? sinceMap[params.timeRange] : undefined,
    start_time: params.startTime ? new Date(params.startTime).toISOString() : undefined,
    end_time: params.endTime ? new Date(params.endTime).toISOString() : undefined,
  });

  const data = response.data.data;

  // histogram: 优先用后端返回的，fallback 客户端从分页日志生成
  const histogram = (data.histogram && data.histogram.length > 0)
    ? data.histogram
    : (data.logs || []).map((l) => ({ timestamp: l.timestamp, severity: l.severity }));

  return {
    logs: data.logs || [],
    total: data.total || 0,
    facets: data.facets || { services: [], severities: [], scopes: [] },
    histogram,
  };
}
