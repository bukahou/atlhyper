/**
 * Logs 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import { mockQueryLogs } from "@/mock/logs";
import type { MockLogQueryParams } from "@/mock/logs";
import * as observeApi from "@/api/observe";
import type { LogQueryResult, LogHistogramResult } from "@/types/model/log";

export type { MockLogQueryParams } from "@/mock/logs";

/** timeRange 字符串 → since duration (保留给 mock 模式兼容) */
const SINCE_MAP: Record<string, string> = {
  "15min": "15m",
  "1h": "1h",
  "24h": "24h",
  "7d": "168h",
  "15d": "360h",
  "30d": "720h",
};

/** 查询参数（API 模式需要 clusterId） */
export interface LogQueryParams extends MockLogQueryParams {
  clusterId?: string;
  /** Go duration 字符串（直传 since 参数） */
  since?: string;
  /** Brush 选区开始时间 (epoch ms) */
  startTime?: number;
  /** Brush 选区结束时间 (epoch ms) */
  endTime?: number;
  /** 跨信号关联：按 TraceId 过滤 */
  traceId?: string;
  /** 跨信号关联：按 SpanId 过滤 */
  spanId?: string;
}

/** 直方图查询参数 */
export interface LogHistogramParams {
  clusterId?: string;
  search?: string;
  services?: string[];
  severities?: string[];
  scopes?: string[];
  timeRange?: string;
  /** Go duration 字符串（直传 since） */
  since?: string;
  /** 绝对开始时间 ISO 字符串 */
  startTime?: string;
  /** 绝对结束时间 ISO 字符串 */
  endTime?: string;
}

/**
 * 查询日志（列表 + 分面，不含直方图）
 */
export async function queryLogs(params?: LogQueryParams): Promise<LogQueryResult> {
  if (getDataSourceMode("logs") === "mock" || !params?.clusterId) {
    const mock = mockQueryLogs(params);
    return { logs: mock.logs, total: mock.total, facets: mock.facets };
  }

  const response = await observeApi.queryLogs({
    cluster_id: params.clusterId,
    query: params.search,
    level: params.severities?.length === 1 ? params.severities[0] : undefined,
    service: params.services?.length === 1 ? params.services[0] : undefined,
    scope: params.scopes?.length === 1 ? params.scopes[0] : undefined,
    limit: params.limit,
    offset: params.offset,
    since: params.since || (params.timeRange ? SINCE_MAP[params.timeRange] : undefined),
    start_time: params.startTime ? new Date(params.startTime).toISOString() : undefined,
    end_time: params.endTime ? new Date(params.endTime).toISOString() : undefined,
    trace_id: params.traceId,
    span_id: params.spanId,
  });

  const data = response.data.data;
  return {
    logs: data.logs || [],
    total: data.total || 0,
    facets: data.facets || { services: [], severities: [], scopes: [] },
  };
}

/**
 * 查询日志直方图（独立 ClickHouse 聚合，~30 桶）
 *
 * mock 模式: 从 mock 日志数据生成客户端桶
 * api  模式: 直接调用 /histogram 端点获取服务端预聚合数据
 */
export async function queryLogHistogram(params?: LogHistogramParams): Promise<LogHistogramResult> {
  if (getDataSourceMode("logs") === "mock" || !params?.clusterId) {
    return mockLogHistogram(params);
  }

  const response = await observeApi.getLogsHistogram({
    cluster_id: params.clusterId,
    query: params.search,
    level: params.severities?.length === 1 ? params.severities[0] : undefined,
    service: params.services?.length === 1 ? params.services[0] : undefined,
    scope: params.scopes?.length === 1 ? params.scopes[0] : undefined,
    since: params.since || (params.timeRange ? SINCE_MAP[params.timeRange] : undefined),
    start_time: params.startTime,
    end_time: params.endTime,
  });

  const data = response.data.data;
  return {
    buckets: data?.buckets || [],
    intervalMs: data?.intervalMs || 0,
  };
}

/** Mock 直方图：从 mock 日志数据客户端分桶 */
function mockLogHistogram(params?: LogHistogramParams): LogHistogramResult {
  const mock = mockQueryLogs(params);
  const logs = mock.logs;
  if (logs.length === 0) {
    return { buckets: [], intervalMs: 0 };
  }

  const sorted = [...logs].sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  );
  const minTime = new Date(sorted[0].timestamp).getTime();
  const maxTime = new Date(sorted[sorted.length - 1].timestamp).getTime();
  const bucketCount = Math.min(30, Math.max(5, sorted.length));
  const intervalMs = Math.max(Math.ceil((maxTime - minTime) / bucketCount), 1000);

  const map = new Map<string, number>();
  for (const entry of sorted) {
    const ts = new Date(entry.timestamp).getTime();
    const bucketStart = minTime + Math.min(Math.floor((ts - minTime) / intervalMs), bucketCount - 1) * intervalMs;
    const key = `${bucketStart}|${entry.severity}`;
    map.set(key, (map.get(key) ?? 0) + 1);
  }

  const buckets = Array.from(map.entries()).map(([key, count]) => {
    const [ts, severity] = key.split("|");
    return { timestamp: new Date(Number(ts)).toISOString(), severity, count };
  });

  return { buckets, intervalMs };
}
