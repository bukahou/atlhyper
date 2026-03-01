/**
 * Log Mock — 查询 API
 */

import type { LogQueryResult, LogFacets } from "@/types/model/log";
import { mockLogEntries } from "./data";

/** 时间范围 → 毫秒 */
const TIME_RANGE_MS: Record<string, number> = {
  "15min": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "15d": 15 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
};

export interface MockLogQueryParams {
  search?: string;
  services?: string[];
  severities?: string[];
  scopes?: string[];
  timeRange?: string;
  limit?: number;
  offset?: number;
}

/** 计算分面统计（基于传入的日志子集） */
function computeFacets(entries: typeof mockLogEntries): LogFacets {
  const svcMap = new Map<string, number>();
  const sevMap = new Map<string, number>();
  const scopeMap = new Map<string, number>();

  for (const log of entries) {
    svcMap.set(log.serviceName, (svcMap.get(log.serviceName) ?? 0) + 1);
    sevMap.set(log.severity, (sevMap.get(log.severity) ?? 0) + 1);
    scopeMap.set(log.scopeName, (scopeMap.get(log.scopeName) ?? 0) + 1);
  }

  const toFacets = (m: Map<string, number>) =>
    Array.from(m.entries())
      .map(([value, count]) => ({ value, count }))
      .sort((a, b) => b.count - a.count);

  return {
    services: toFacets(svcMap),
    severities: toFacets(sevMap),
    scopes: toFacets(scopeMap),
  };
}

export function mockQueryLogs(params?: MockLogQueryParams): LogQueryResult {
  let logs = mockLogEntries;

  // 时间范围过滤（最先执行，决定可见数据集）
  if (params?.timeRange && TIME_RANGE_MS[params.timeRange]) {
    // 以 mock 数据中最新的时间戳为基准（模拟 "now"）
    const newest = Math.max(...logs.map((l) => new Date(l.timestamp).getTime()));
    const cutoff = newest - TIME_RANGE_MS[params.timeRange];
    logs = logs.filter((l) => new Date(l.timestamp).getTime() >= cutoff);
  }

  // Facets 基于时间范围内的数据（在其他过滤之前）
  const facets = computeFacets(logs);

  // Body 全文搜索
  if (params?.search) {
    const q = params.search.toLowerCase();
    logs = logs.filter((l) => l.body.toLowerCase().includes(q));
  }

  // 服务过滤（多选 OR）
  if (params?.services && params.services.length > 0) {
    const set = new Set(params.services);
    logs = logs.filter((l) => set.has(l.serviceName));
  }

  // 级别过滤（多选 OR）
  if (params?.severities && params.severities.length > 0) {
    const set = new Set(params.severities);
    logs = logs.filter((l) => set.has(l.severity));
  }

  // ScopeName 过滤（多选 OR）
  if (params?.scopes && params.scopes.length > 0) {
    const set = new Set(params.scopes);
    logs = logs.filter((l) => set.has(l.scopeName));
  }

  const total = logs.length;

  const offset = params?.offset ?? 0;
  const limit = params?.limit ?? 100;
  logs = logs.slice(offset, offset + limit);

  return {
    logs,
    total,
    facets,
  };
}
