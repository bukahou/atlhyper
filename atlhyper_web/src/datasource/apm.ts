/**
 * APM 数据源 — 直接调用 Master observe API 查询 ClickHouse（没有演示数据分支）
 *
 * 约定同 datasource/cluster.ts：
 * - 读函数返回解包后的数据
 * - 读函数最后一个参数是可选的 AbortSignal，原样交给 api 层，用于取消请求
 * - 请求失败时错误原样抛出，由查询层与界面显示出错状态；不在这里吞成空数据
 *   （吞掉之后「接口挂了」和「确实没有数据」在界面上分不出来）
 *
 * 这里不判断权限：链路列表 / 详情 / 相关性需要 traces.read，由查询工厂的 enabled 把关。
 */

import { mockGetLatencyDistribution } from "@/mock/apm";
import type { APMService, TraceSummary, TraceDetail, Topology, OperationStats, Dependency, APMTimePoint, HTTPStats, DBOperationStats, CorrelationResult, CorrelationMode } from "@/types/model/apm";
import * as observeApi from "@/api/observe-apm";
import type { TraceTimeRangeQuery } from "@/api/observe-apm";
import type { TimeRangeParams } from "@/lib/time-range";

/** 链路列表的筛选条件 */
export interface TraceFilter {
  service?: string;
  operation?: string;
  minDurationMs?: number;
  limit?: number;
  /** HTTP 状态码（按状态码钻取时用） */
  statusCode?: string;
  /** HTTP 方法（按状态码钻取时用） */
  method?: string;
}

/** 相关性分析的对象：哪个服务（可选到某个操作）、分析失败还是延迟 */
export interface CorrelationSubject {
  serviceName: string;
  correlationMode: CorrelationMode;
  operation?: string;
}

/** 「最近 15 分钟」：快照覆盖的默认窗口，也是没有相对范围时趋势图退回的窗口 */
const DEFAULT_RECENT_WINDOW = "15m";

function toTimeRangeQuery(range: TimeRangeParams): TraceTimeRangeQuery {
  return {
    ...(range.since ? { time_range: range.since } : {}),
    ...(range.startTime ? { start_time: range.startTime } : {}),
    ...(range.endTime ? { end_time: range.endTime } : {}),
  };
}

/**
 * HTTP / 数据库统计的时间参数。
 *
 * 15 分钟是这两个端点的默认窗口，不传 time_range 时后端直接读快照，比带参数查 ClickHouse 快。
 */
function toStatsTimeRangeQuery(range: TimeRangeParams): TraceTimeRangeQuery {
  return toTimeRangeQuery(range.since === DEFAULT_RECENT_WINDOW ? { ...range, since: undefined } : range);
}

/** 时间范围字符串（"30m" / "6h" / "2d"）→ 分钟数；认不出来时按 60 分钟 */
export function timeRangeToMinutes(timeRange: string): number {
  const match = timeRange.match(/^(\d+)(m|h|d)$/);
  if (!match) return 60;
  const amount = parseInt(match[1], 10);
  switch (match[2]) {
    case "m": return amount;
    case "h": return amount * 60;
    default: return amount * 24 * 60;
  }
}

export async function getAPMServices(clusterId: string, range: TimeRangeParams, signal?: AbortSignal): Promise<APMService[]> {
  return (await observeApi.getTracesServices(clusterId, toTimeRangeQuery(range), signal)).data.data || [];
}

export async function getTopology(clusterId: string, range: TimeRangeParams, signal?: AbortSignal): Promise<Topology> {
  const topology = (await observeApi.getTracesTopology(clusterId, toTimeRangeQuery(range), signal)).data.data;
  return { nodes: topology?.nodes || [], edges: topology?.edges || [] };
}

export async function getOperations(clusterId: string, range: TimeRangeParams, signal?: AbortSignal): Promise<OperationStats[]> {
  return (await observeApi.getTracesOperations(clusterId, toTimeRangeQuery(range), signal)).data.data || [];
}

/** 链路列表（内容级数据，需要 traces.read） */
export async function queryTraces(
  clusterId: string,
  filter: TraceFilter,
  range: TimeRangeParams,
  signal?: AbortSignal,
): Promise<{ traces: TraceSummary[]; total: number }> {
  const response = await observeApi.getTracesList(
    clusterId,
    {
      service: filter.service,
      operation: filter.operation,
      min_duration: filter.minDurationMs ? String(filter.minDurationMs) : undefined,
      limit: filter.limit,
      status_code: filter.statusCode,
      method: filter.method,
      ...toTimeRangeQuery(range),
    },
    signal,
  );
  // 快照直读返回 { traces, total }，经指令通道读取返回数组
  const payload = response.data.data;
  if (Array.isArray(payload)) return { traces: payload, total: payload.length };
  const traces = payload?.traces || [];
  return { traces, total: payload?.total ?? traces.length };
}

/** 链路详情（内容级数据，需要 traces.read）。后端没有返回内容时为 null */
export async function getTraceDetail(clusterId: string, traceId: string, signal?: AbortSignal): Promise<TraceDetail | null> {
  return (await observeApi.getTraceDetail(clusterId, traceId, signal)).data.data || null;
}

/** 服务的时序趋势。端点只支持「最近 N 分钟」，所以只用相对范围；绝对范围时退回默认窗口 */
export async function getServiceTimeSeries(
  clusterId: string,
  serviceName: string,
  range: TimeRangeParams,
  signal?: AbortSignal,
): Promise<APMTimePoint[]> {
  const minutes = timeRangeToMinutes(range.since || DEFAULT_RECENT_WINDOW);
  return (await observeApi.getAPMServiceSeries(clusterId, serviceName, minutes, signal)).data.data?.points || [];
}

export async function getHTTPStats(
  clusterId: string,
  serviceName: string,
  range: TimeRangeParams,
  signal?: AbortSignal,
): Promise<HTTPStats[]> {
  const response = await observeApi.getTracesHTTPStats(
    clusterId,
    { service: serviceName, ...toStatsTimeRangeQuery(range) },
    signal,
  );
  return response.data.data || [];
}

export async function getDBStats(
  clusterId: string,
  serviceName: string,
  range: TimeRangeParams,
  signal?: AbortSignal,
): Promise<DBOperationStats[]> {
  const response = await observeApi.getTracesDBStats(
    clusterId,
    { service: serviceName, ...toStatsTimeRangeQuery(range) },
    signal,
  );
  return response.data.data || [];
}

/** 相关性分析（读的是链路属性，需要 traces.read）。没有结果时为 null */
export async function getCorrelations(
  clusterId: string,
  subject: CorrelationSubject,
  range: TimeRangeParams,
  signal?: AbortSignal,
): Promise<CorrelationResult | null> {
  const response = await observeApi.getTracesCorrelations(
    clusterId,
    {
      service: subject.serviceName,
      operation: subject.operation,
      mode: subject.correlationMode,
      ...toTimeRangeQuery(range),
    },
    signal,
  );
  return response.data.data || null;
}

// ============================================================
// 客户端计算函数 — 从已加载的真实数据派生
// ============================================================

/** 延迟分布直方图 — 纯计算，入参为已加载的 traces */
export function getLatencyDistribution(traces: TraceSummary[]) {
  return mockGetLatencyDistribution(traces);
}

/** 从 topology 数据派生某服务的依赖列表 */
export function getDependenciesFromTopology(serviceName: string, topology: Topology | null): Dependency[] {
  if (!topology) return [];

  const nodeMap = new Map(topology.nodes.map((n) => [n.id, n]));

  // 找出该服务作为 source 的所有 edges
  const deps: Dependency[] = [];
  for (const edge of topology.edges) {
    // source 格式: "namespace/serviceName"，匹配 serviceName
    const sourceService = edge.source.split("/").pop() ?? "";
    if (sourceService !== serviceName) continue;

    const targetNode = nodeMap.get(edge.target);
    const name = targetNode?.name ?? edge.target;
    const type = targetNode?.type ?? "service";

    deps.push({
      name,
      type,
      callCount: edge.callCount,
      avgMs: edge.avgMs,
      errorRate: edge.errorRate,
      impact: 0, // 后面统一计算
    });
  }

  // 计算 impact
  const maxTotal = Math.max(...deps.map((d) => d.callCount * d.avgMs), 1);
  for (const dep of deps) {
    dep.impact = (dep.callCount * dep.avgMs) / maxTotal;
  }

  return deps;
}
