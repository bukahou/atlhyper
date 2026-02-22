/**
 * APM 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 *
 * mock: 纯前端 Mock 数据（含 Latency/Dependencies/SpanType 等客户端计算）
 * api:  通过 Master observe API 查询 ClickHouse
 */

import { getDataSourceMode } from "@/config/data-source";
import {
  mockGetAPMServices,
  mockQueryTraces,
  mockGetTraceDetail,
  mockGetLatencyDistribution,
  mockGetDependencies,
  mockGetSpanTypeBreakdown,
  mockGetTopology,
} from "@/mock/apm";
import type { MockTraceQueryParams } from "@/mock/apm";
import type { APMService, TraceSummary, TraceDetail, Topology } from "@/types/model/apm";
import * as observeApi from "@/api/observe";

export type { MockTraceQueryParams } from "@/mock/apm";

export async function getAPMServices(clusterId?: string): Promise<APMService[]> {
  if (getDataSourceMode("apm") === "mock" || !clusterId) return mockGetAPMServices();
  const response = await observeApi.getTracesServices(clusterId);
  return response.data.data || [];
}

export async function queryTraces(clusterId?: string, params?: MockTraceQueryParams): Promise<{ traces: TraceSummary[]; total: number }> {
  if (getDataSourceMode("apm") === "mock" || !clusterId) return mockQueryTraces(params);
  const response = await observeApi.getTracesList(clusterId, {
    service: params?.service,
    min_duration: params?.minDurationMs ? String(params.minDurationMs) : undefined,
    limit: params?.limit,
  });
  const traces = response.data.data || [];
  return { traces, total: traces.length };
}

export async function getTraceDetail(traceId: string, clusterId?: string): Promise<TraceDetail | null> {
  if (getDataSourceMode("apm") === "mock" || !clusterId) return mockGetTraceDetail(traceId);
  const response = await observeApi.getTraceDetail(clusterId, traceId);
  return response.data.data || null;
}

export async function getTopology(clusterId?: string, timeRange?: string): Promise<Topology> {
  if (getDataSourceMode("apm") === "mock" || !clusterId) return mockGetTopology();
  const response = await observeApi.getTracesTopology(clusterId, timeRange);
  return response.data.data || { nodes: [], edges: [] };
}

// 以下为客户端计算函数，不依赖 API（从已加载的 traces 数据计算）
export function getLatencyDistribution(traces: TraceSummary[]) {
  return mockGetLatencyDistribution(traces);
}

export function getDependencies(serviceName: string) {
  return mockGetDependencies(serviceName);
}

export function getSpanTypeBreakdown(serviceName: string) {
  return mockGetSpanTypeBreakdown(serviceName);
}
