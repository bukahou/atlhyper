/**
 * APM 数据源代理
 *
 * 根据中心配置自动切换 mock / api
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
import type { TraceSummary } from "@/types/model/apm";

export type { MockTraceQueryParams } from "@/mock/apm";

export function getAPMServices() {
  if (getDataSourceMode("apm") === "mock") return mockGetAPMServices();
  // TODO: 真实 API 就绪后替换
  return mockGetAPMServices();
}

export function queryTraces(params?: MockTraceQueryParams) {
  if (getDataSourceMode("apm") === "mock") return mockQueryTraces(params);
  return mockQueryTraces(params);
}

export function getTraceDetail(traceId: string) {
  if (getDataSourceMode("apm") === "mock") return mockGetTraceDetail(traceId);
  return mockGetTraceDetail(traceId);
}

export function getLatencyDistribution(traces: TraceSummary[]) {
  if (getDataSourceMode("apm") === "mock") return mockGetLatencyDistribution(traces);
  return mockGetLatencyDistribution(traces);
}

export function getDependencies(serviceName: string) {
  if (getDataSourceMode("apm") === "mock") return mockGetDependencies(serviceName);
  return mockGetDependencies(serviceName);
}

export function getSpanTypeBreakdown(serviceName: string) {
  if (getDataSourceMode("apm") === "mock") return mockGetSpanTypeBreakdown(serviceName);
  return mockGetSpanTypeBreakdown(serviceName);
}

export function getTopology() {
  if (getDataSourceMode("apm") === "mock") return mockGetTopology();
  return mockGetTopology();
}
