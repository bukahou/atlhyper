/**
 * Log Mock — 查询 API
 */

import type { LogQueryResult, LogFacets } from "@/types/model/log";
import { mockLogEntries } from "./data";

export interface MockLogQueryParams {
  search?: string;
  services?: string[];
  severities?: string[];
  scopes?: string[];
  limit?: number;
  offset?: number;
}

/** 计算分面统计（始终基于全量数据） */
function computeFacets(): LogFacets {
  const svcMap = new Map<string, number>();
  const sevMap = new Map<string, number>();
  const scopeMap = new Map<string, number>();

  for (const log of mockLogEntries) {
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

const cachedFacets = computeFacets();

export function mockQueryLogs(params?: MockLogQueryParams): LogQueryResult {
  let logs = mockLogEntries;

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
    facets: cachedFacets,
  };
}
