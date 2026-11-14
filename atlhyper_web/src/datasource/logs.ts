/**
 * Logs 数据源代理
 *
 * 根据中心配置自动切换 mock / api。约定同 datasource/cluster.ts：
 * - 读函数返回解包后的数据，mock 分支返回相同形态
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 *
 * 这里不判断权限：日志正文需要 logs.read，由查询工厂的 enabled 把关（没有权限就不发请求）。
 */

import { getDataSourceMode } from "@/config/data-source";
import { mockQueryLogs } from "@/mock/logs";
import * as observeApi from "@/api/observe-logs";
import type { TimeRangeParams } from "@/lib/time-range";
import type { LogQueryResult, LogHistogramResult } from "@/types/model/log";

/** 日志的筛选条件 + 时间范围（直方图与列表共用的部分） */
export interface LogHistogramParams extends TimeRangeParams {
  search?: string;
  services?: string[];
  severities?: string[];
  scopes?: string[];
}

/** 日志列表的查询条件：在筛选条件之上再加分页与跨信号关联 */
export interface LogQueryParams extends LogHistogramParams {
  limit?: number;
  offset?: number;
  /** 跨信号关联：按 TraceId 过滤 */
  traceId?: string;
  /** 跨信号关联：按 SpanId 过滤 */
  spanId?: string;
}

/** 后端的多选筛选目前只支持单值：恰好选了一个才下推，否则不过滤 */
function singleSelected(values?: string[]): string | undefined {
  return values?.length === 1 ? values[0] : undefined;
}

/** 查询日志（列表 + 分面，不含直方图） */
export async function queryLogs(
  clusterId: string,
  params: LogQueryParams,
  signal?: AbortSignal,
): Promise<LogQueryResult> {
  if (getDataSourceMode("logs") === "mock") {
    const { logs, total, facets } = mockQueryLogs(params);
    return { logs, total, facets };
  }

  const response = await observeApi.queryLogs(
    {
      cluster_id: clusterId,
      query: params.search,
      level: singleSelected(params.severities),
      service: singleSelected(params.services),
      scope: singleSelected(params.scopes),
      limit: params.limit,
      offset: params.offset,
      since: params.since,
      start_time: params.startTime,
      end_time: params.endTime,
      trace_id: params.traceId,
      span_id: params.spanId,
    },
    signal,
  );

  const page = response.data.data;
  return {
    logs: page?.logs || [],
    total: page?.total || 0,
    facets: page?.facets || { services: [], severities: [], scopes: [] },
  };
}

/**
 * 查询日志直方图（独立 ClickHouse 聚合，~30 桶）
 *
 * mock 模式: 从 mock 日志数据生成客户端桶
 * api  模式: 直接调用 /histogram 端点获取服务端预聚合数据
 */
export async function queryLogHistogram(
  clusterId: string,
  params: LogHistogramParams,
  signal?: AbortSignal,
): Promise<LogHistogramResult> {
  if (getDataSourceMode("logs") === "mock") return mockLogHistogram(params);

  const response = await observeApi.getLogsHistogram(
    {
      cluster_id: clusterId,
      query: params.search,
      level: singleSelected(params.severities),
      service: singleSelected(params.services),
      scope: singleSelected(params.scopes),
      since: params.since,
      start_time: params.startTime,
      end_time: params.endTime,
    },
    signal,
  );

  const histogram = response.data.data;
  return {
    buckets: histogram?.buckets || [],
    intervalMs: histogram?.intervalMs || 0,
  };
}

/** Mock 直方图：从 mock 日志数据客户端分桶 */
function mockLogHistogram(params: LogHistogramParams): LogHistogramResult {
  const logs = mockQueryLogs(params).logs;
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

  const countByBucket = new Map<string, number>();
  for (const entry of sorted) {
    const ts = new Date(entry.timestamp).getTime();
    const bucketStart = minTime + Math.min(Math.floor((ts - minTime) / intervalMs), bucketCount - 1) * intervalMs;
    const key = `${bucketStart}|${entry.severity}`;
    countByBucket.set(key, (countByBucket.get(key) ?? 0) + 1);
  }

  const buckets = Array.from(countByBucket.entries()).map(([key, count]) => {
    const [ts, severity] = key.split("|");
    return { timestamp: new Date(Number(ts)).toISOString(), severity, count };
  });

  return { buckets, intervalMs };
}
