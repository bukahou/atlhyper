/**
 * APM Mock — Trace 查询 API
 */

import type { TraceSummary, TraceDetail, OperationStats } from "@/types/model/apm";
import { mockTraceSummaries, mockTraceDetailsMap } from "./data";

export interface MockTraceQueryParams {
  service?: string;
  operation?: string;
  namespace?: string;
  statusCode?: string;
  minDurationMs?: number;
  maxDurationMs?: number;
  limit?: number;
  timeRange?: string;
}

export function mockQueryTraces(
  params?: MockTraceQueryParams
): { traces: TraceSummary[]; total: number } {
  let traces = [...mockTraceSummaries];

  if (params?.service) {
    traces = traces.filter((t) => t.rootService === params.service);
  }
  if (params?.minDurationMs) {
    traces = traces.filter((t) => t.durationMs >= params.minDurationMs!);
  }
  if (params?.maxDurationMs) {
    traces = traces.filter((t) => t.durationMs <= params.maxDurationMs!);
  }

  const total = traces.length;
  const limit = params?.limit ?? 100;
  traces = traces.slice(0, limit);

  return { traces, total };
}

export function mockGetTraceDetail(traceId: string): TraceDetail | null {
  return mockTraceDetailsMap[traceId] ?? null;
}

/** 从 mock traces 生成操作级聚合统计 */
export function mockGetOperations(): OperationStats[] {
  const map = new Map<string, { durations: number[]; errorCount: number; service: string }>();
  for (const tr of mockTraceSummaries) {
    const key = `${tr.rootService}::${tr.rootOperation}`;
    const entry = map.get(key) ?? { durations: [], errorCount: 0, service: tr.rootService };
    entry.durations.push(tr.durationMs);
    if (tr.hasError) entry.errorCount++;
    map.set(key, entry);
  }

  const result: OperationStats[] = [];
  for (const [key, data] of map) {
    const [, opName] = key.split("::");
    const count = data.durations.length;
    const sorted = [...data.durations].sort((a, b) => a - b);
    const avg = sorted.reduce((a, b) => a + b, 0) / count;
    result.push({
      serviceName: data.service,
      operationName: opName,
      spanCount: count,
      errorCount: data.errorCount,
      successRate: count > 0 ? (count - data.errorCount) / count : 1,
      avgDurationMs: Math.round(avg * 100) / 100,
      p50Ms: sorted[Math.floor(count * 0.5)] ?? 0,
      p99Ms: sorted[Math.floor(count * 0.99)] ?? sorted[count - 1] ?? 0,
      rps: Math.round((count / 900) * 1000) / 1000,
    });
  }
  return result.sort((a, b) => b.spanCount - a.spanCount);
}
