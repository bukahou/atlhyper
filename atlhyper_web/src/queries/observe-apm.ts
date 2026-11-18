/**
 * 链路页（APM）的取数描述
 *
 * 三层下钻各用各的查询，哪一层显示就只有哪一层的组件在用它们：
 * - 服务列表层：服务列表 + 拓扑
 * - 服务详情层：操作统计 + 拓扑 + 该服务的趋势 / 状态码 / 数据库统计 / 链路列表 / 相关性
 * - 链路详情层：样本序列 + 链路详情
 *
 * 需要 traces.read 的查询（链路列表、详情、相关性）把「有没有权限」作为参数传进来，
 * 调用方用 useSessionReady() && useCan()("traces.read") 得到。
 *
 * 时间范围用 TimeRangeParams：相对范围只有描述（"1h"），由后端在收到请求时换算起止时间。
 */
import { queryOptions } from "@tanstack/react-query";
import type { DataSourceMode } from "@/config/data-source";
import {
  getAPMServices,
  getCorrelations,
  getDBStats,
  getHTTPStats,
  getOperations,
  getServiceTimeSeries,
  getTopology,
  getTraceDetail,
  queryTraces,
  type CorrelationSubject,
  type TraceFilter,
} from "@/datasource/apm";
import type { TimeRangeParams } from "@/lib/time-range";
import { clusterDataAvailable } from "./availability";
import { apmKeys } from "./keys";
import { REFRESH } from "./refresh";

/** APM 数据源没有演示数据分支（见 datasource/apm.ts），所以固定是真实接口 */
const APM_MODE: DataSourceMode = "api";

/** 进入某个操作时取多少条链路样本 */
const OPERATION_SAMPLE_LIMIT = 200;

// ============================================================
// 服务列表层 / 服务详情层共用的聚合数据（轮询）
// ============================================================

/** 服务列表（各服务的聚合统计） */
export function apmServicesQuery(clusterId: string, range: TimeRangeParams) {
  return queryOptions({
    queryKey: apmKeys.services(clusterId, APM_MODE, range),
    queryFn: ({ signal }) => getAPMServices(clusterId, range, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, APM_MODE),
  });
}

/** 服务拓扑（服务列表层画图，服务详情层派生依赖列表） */
export function apmTopologyQuery(clusterId: string, range: TimeRangeParams) {
  return queryOptions({
    queryKey: apmKeys.topology(clusterId, APM_MODE, range),
    queryFn: ({ signal }) => getTopology(clusterId, range, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, APM_MODE),
  });
}

/** 操作级聚合统计（全部服务；服务详情层按服务名筛） */
export function apmOperationsQuery(clusterId: string, range: TimeRangeParams) {
  return queryOptions({
    queryKey: apmKeys.operations(clusterId, APM_MODE, range),
    queryFn: ({ signal }) => getOperations(clusterId, range, signal),
    refetchInterval: REFRESH.standard,
    staleTime: REFRESH.standard,
    enabled: clusterDataAvailable(clusterId, APM_MODE),
  });
}

// ============================================================
// 服务详情层：某一个服务的数据
// ============================================================

/** 服务的时序趋势（延迟 / 吞吐 / 错误数） */
export function apmServiceSeriesQuery(clusterId: string, serviceName: string, range: TimeRangeParams) {
  return queryOptions({
    queryKey: apmKeys.serviceSeries(clusterId, APM_MODE, serviceName, range),
    queryFn: ({ signal }) => getServiceTimeSeries(clusterId, serviceName, range, signal),
    enabled: clusterDataAvailable(clusterId, APM_MODE) && serviceName !== "",
  });
}

/** 服务的 HTTP 状态码分布 */
export function apmHttpStatsQuery(clusterId: string, serviceName: string, range: TimeRangeParams) {
  return queryOptions({
    queryKey: apmKeys.httpStats(clusterId, APM_MODE, serviceName, range),
    queryFn: ({ signal }) => getHTTPStats(clusterId, serviceName, range, signal),
    enabled: clusterDataAvailable(clusterId, APM_MODE) && serviceName !== "",
  });
}

/** 服务的数据库操作统计 */
export function apmDbStatsQuery(clusterId: string, serviceName: string, range: TimeRangeParams) {
  return queryOptions({
    queryKey: apmKeys.dbStats(clusterId, APM_MODE, serviceName, range),
    queryFn: ({ signal }) => getDBStats(clusterId, serviceName, range, signal),
    enabled: clusterDataAvailable(clusterId, APM_MODE) && serviceName !== "",
  });
}

/** 链路列表（服务的全部链路、按状态码筛出的链路）。内容级数据，需要 traces.read */
export function apmTracesQuery(clusterId: string, filter: TraceFilter, range: TimeRangeParams, canReadTraces: boolean) {
  return queryOptions({
    queryKey: apmKeys.traces(clusterId, APM_MODE, filter, range),
    queryFn: ({ signal }) => queryTraces(clusterId, filter, range, signal),
    enabled: clusterDataAvailable(clusterId, APM_MODE) && canReadTraces,
  });
}

/** 相关性分析：慢 / 错的请求里什么属性显著超标。读的是链路属性，需要 traces.read */
export function apmCorrelationsQuery(
  clusterId: string,
  subject: CorrelationSubject,
  range: TimeRangeParams,
  canReadTraces: boolean,
) {
  return queryOptions({
    queryKey: apmKeys.correlations(clusterId, APM_MODE, subject, range),
    queryFn: ({ signal }) => getCorrelations(clusterId, subject, range, signal),
    enabled: clusterDataAvailable(clusterId, APM_MODE) && subject.serviceName !== "" && canReadTraces,
  });
}

// ============================================================
// 链路详情层
// ============================================================

/**
 * 某个操作的链路样本序列（链路详情里前后翻样本、画延迟分布用）。
 *
 * 进入时取一次就不再动：样本是按位置翻的，序列中途变了，「下一条」就不是用户以为的那一条。
 * 所以不轮询、切回标签页也不重取；离开这一层立即丢弃缓存，下次进来取新的。
 */
export function apmOperationSamplesQuery(
  clusterId: string,
  target: { service: string; operation: string },
  range: TimeRangeParams,
  canReadTraces: boolean,
) {
  const filter: TraceFilter = { service: target.service, operation: target.operation, limit: OPERATION_SAMPLE_LIMIT };
  return queryOptions({
    queryKey: apmKeys.operationSamples(clusterId, APM_MODE, filter, range),
    queryFn: async ({ signal }) => (await queryTraces(clusterId, filter, range, signal)).traces,
    staleTime: Infinity,
    gcTime: 0,
    enabled: clusterDataAvailable(clusterId, APM_MODE) && target.operation !== "" && canReadTraces,
  });
}

/**
 * 链路详情（全部 span）。内容级数据，需要 traces.read。
 *
 * 链路是已经发生的历史数据，取到后不自动重取 —— 过了保留期的链路重取只会失败。
 * 用户点页头的刷新时仍会重取。
 */
export function apmTraceDetailQuery(clusterId: string, traceId: string, canReadTraces: boolean) {
  return queryOptions({
    queryKey: apmKeys.traceDetail(clusterId, APM_MODE, traceId),
    queryFn: ({ signal }) => getTraceDetail(clusterId, traceId, signal),
    staleTime: Infinity,
    enabled: clusterDataAvailable(clusterId, APM_MODE) && traceId !== "" && canReadTraces,
  });
}
