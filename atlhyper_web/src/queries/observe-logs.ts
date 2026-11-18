/**
 * 日志页的取数描述
 *
 * 列表与直方图是两个独立的查询：直方图只有计数（所有人可见），列表是日志正文（需要 logs.read）。
 * 两者互不依赖 —— 列表的键里只有它自己的查询条件，直方图返回什么都不会让列表重新请求。
 */
import { keepPreviousData, queryOptions } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import { queryLogHistogram, queryLogs, type LogHistogramParams, type LogQueryParams } from "@/datasource/logs";
import type { TimeRangeParams } from "@/lib/time-range";
import { clusterDataAvailable } from "./availability";
import { logsKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 直方图上 brush 选中的一段：从第一个桶的起点到最后一个桶的起点，外加选中那一刻的桶宽 */
export interface BrushedBuckets {
  firstBucketStartMs: number;
  lastBucketStartMs: number;
  /** 选中那一刻的桶宽。和选区存在一起，列表请求就不必再去读直方图的返回 */
  bucketIntervalMs: number;
}

/**
 * 列表实际查询的时间范围：有 brush 选区时用选区，否则就是页面的时间范围。
 *
 * 选区的结束时间要含最后一个桶的整段（桶的起点 + 桶宽），否则最后一个桶里的日志会被漏掉。
 * 选区是固定的时间段，换算出来的起止时间不随渲染变化，可以放进键。
 */
export function resolveLogListRange(range: TimeRangeParams, brush: BrushedBuckets | null): TimeRangeParams {
  if (brush === null) return range;
  return {
    ...range,
    startTime: new Date(brush.firstBucketStartMs).toISOString(),
    endTime: new Date(brush.lastBucketStartMs + brush.bucketIntervalMs).toISOString(),
  };
}

/**
 * 日志列表（正文 + 分面）。
 *
 * canReadLogs 由调用方用 useSessionReady() && useCan()("logs.read") 得到：
 * 没有权限、或权限还没加载完，都不发请求。权限一就绪查询立即启用，不用等下一轮轮询。
 *
 * 翻页、换筛选条件时先沿用上一份结果，表格不闪空。
 */
export function logListQuery(clusterId: string, params: LogQueryParams, canReadLogs: boolean) {
  const mode = getDataSourceMode("logs");
  return queryOptions({
    queryKey: logsKeys.list(clusterId, mode, params),
    queryFn: ({ signal }) => queryLogs(clusterId, params, signal),
    refetchInterval: REFRESH.moderate,
    staleTime: REFRESH.moderate,
    placeholderData: keepPreviousData,
    enabled: clusterDataAvailable(clusterId, mode) && canReadLogs,
  });
}

/** 日志直方图（按级别分桶的计数）。只有计数，不需要权限 */
export function logHistogramQuery(clusterId: string, params: LogHistogramParams) {
  const mode = getDataSourceMode("logs");
  return queryOptions({
    queryKey: logsKeys.histogram(clusterId, mode, params),
    queryFn: ({ signal }) => queryLogHistogram(clusterId, params, signal),
    refetchInterval: REFRESH.moderate,
    staleTime: REFRESH.moderate,
    // 换筛选条件时先留着上一份图，不闪成「暂无数据」
    placeholderData: keepPreviousData,
    enabled: clusterDataAvailable(clusterId, mode),
  });
}

/**
 * 某条链路关联的全部日志（链路详情的「日志」标签页）。
 *
 * 链路是历史数据，日志只在打开时取一次，不轮询。
 */
export function traceLogsQuery(clusterId: string, traceId: string, limit: number, canReadLogs: boolean) {
  const mode = getDataSourceMode("logs");
  return queryOptions({
    queryKey: logsKeys.traceLogs(clusterId, mode, traceId, limit),
    queryFn: async ({ signal }) => (await queryLogs(clusterId, { traceId, limit }, signal)).logs,
    enabled: clusterDataAvailable(clusterId, mode) && traceId !== "" && canReadLogs,
  });
}
