/**
 * APM Mock — Trace 查询 API
 */

import type { TraceSummary, TraceDetail } from "@/types/model/apm";
import { mockTraceSummaries, mockTraceDetailsMap } from "./data";

export interface MockTraceQueryParams {
  service?: string;
  namespace?: string;
  statusCode?: string;
  minDurationMs?: number;
  maxDurationMs?: number;
  limit?: number;
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
