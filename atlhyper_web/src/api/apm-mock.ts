/**
 * APM Mock Data
 *
 * Real trace data exported from Jaeger (zgmf-x10a cluster, geass services).
 * Used until Agent/Master APM pipeline is built.
 */

import type {
  TraceService,
  TraceSummary,
  TraceDetail,
  TraceQueryParams,
  ServiceStats,
  Dependency,
  LatencyBucket,
  SpanTypeBreakdown,
} from "./apm";

import mockRaw from "./apm-mock-data.json";

// ============================================================
// Typed mock data
// ============================================================

const mockData = mockRaw as {
  services: TraceService[];
  traceList: TraceSummary[];
  traceDetails: Record<string, TraceDetail>;
};

// ============================================================
// Mock API functions (same signatures as real API)
// ============================================================

export async function mockGetTraceServices(): Promise<TraceService[]> {
  return mockData.services;
}

export async function mockQueryTraces(
  params?: TraceQueryParams
): Promise<{ traces: TraceSummary[]; total: number }> {
  let traces = [...mockData.traceList];

  // Filter by service
  if (params?.service) {
    traces = traces.filter((t) => t.rootService === params.service);
  }

  // Filter by operation
  if (params?.operation) {
    traces = traces.filter((t) => t.rootOperation === params.operation);
  }

  // Filter by duration
  if (params?.minDuration) {
    traces = traces.filter((t) => t.duration >= params.minDuration!);
  }
  if (params?.maxDuration) {
    traces = traces.filter((t) => t.duration <= params.maxDuration!);
  }

  // Sort by startTime desc (newest first)
  traces.sort((a, b) => b.startTime - a.startTime);

  // Apply limit
  const limit = params?.limit ?? 20;
  const total = traces.length;
  traces = traces.slice(0, limit);

  return { traces, total };
}

export async function mockGetTraceDetail(
  traceId: string
): Promise<TraceDetail | null> {
  return mockData.traceDetails[traceId] ?? null;
}

// ============================================================
// Kibana-style computed mock data
// ============================================================

/** Compute per-service stats from all trace and span data */
export function mockGetAllServiceStats(): ServiceStats[] {
  const serviceMap = new Map<
    string,
    { durations: number[]; errors: number[] }
  >();

  // Initialize from services list
  for (const svc of mockData.services) {
    serviceMap.set(svc.name, { durations: [], errors: [] });
  }

  // Aggregate per-trace data grouped by rootService
  for (const trace of mockData.traceList) {
    const entry = serviceMap.get(trace.rootService);
    if (entry) {
      entry.durations.push(trace.duration);
      entry.errors.push(trace.hasError ? 1 : 0);
    }
  }

  // Compute time window for throughput calculation
  const allTimes = mockData.traceList.map((t) => t.startTime);
  const timeRangeMin =
    allTimes.length > 1
      ? (Math.max(...allTimes) - Math.min(...allTimes)) / 60_000_000
      : 1; // μs -> minutes

  return Array.from(serviceMap.entries()).map(([name, data]) => {
    const count = data.durations.length;
    const totalDuration = data.durations.reduce((a, b) => a + b, 0);
    const errorCount = data.errors.reduce((a, b) => a + b, 0);

    return {
      name,
      environment: "production",
      latencyAvg: count > 0 ? totalDuration / count : 0,
      throughput: count > 0 ? count / Math.max(timeRangeMin, 1) : 0,
      errorRate: count > 0 ? errorCount / count : 0,
      latencyPoints: data.durations,
      errorRatePoints: data.errors,
    };
  });
}

/** Compute downstream dependencies for a service from span parent-child relationships */
export function mockGetServiceDependencies(serviceName: string): Dependency[] {
  const depMap = new Map<
    string,
    { durations: number[]; errorCount: number }
  >();

  // Look through all trace details for cross-service calls from this service
  for (const detail of Object.values(mockData.traceDetails)) {
    const spanMap = new Map(detail.spans.map((s) => [s.spanId, s]));

    for (const span of detail.spans) {
      if (span.serviceName === serviceName && span.parentSpanId) {
        // Find children of this span that belong to a different service
        for (const child of detail.spans) {
          if (
            child.parentSpanId === span.spanId &&
            child.serviceName !== serviceName
          ) {
            const entry = depMap.get(child.serviceName) ?? {
              durations: [],
              errorCount: 0,
            };
            entry.durations.push(child.duration);
            if (child.status === "error") entry.errorCount++;
            depMap.set(child.serviceName, entry);
          }
        }
      }
    }

    // Also check root spans calling other services
    for (const span of detail.spans) {
      if (span.serviceName !== serviceName) continue;
      for (const child of detail.spans) {
        if (
          child.parentSpanId === span.spanId &&
          child.serviceName !== serviceName &&
          !depMap.has(child.serviceName)
        ) {
          const entry = depMap.get(child.serviceName) ?? {
            durations: [],
            errorCount: 0,
          };
          entry.durations.push(child.duration);
          if (child.status === "error") entry.errorCount++;
          depMap.set(child.serviceName, entry);
        }
      }
    }
  }

  // Find max total duration for impact calculation
  const allTotals = Array.from(depMap.values()).map((d) =>
    d.durations.reduce((a, b) => a + b, 0)
  );
  const maxTotal = Math.max(...allTotals, 1);

  return Array.from(depMap.entries()).map(([name, data]) => {
    const count = data.durations.length;
    const totalDuration = data.durations.reduce((a, b) => a + b, 0);
    return {
      name,
      latencyAvg: count > 0 ? totalDuration / count : 0,
      throughput: count,
      errorRate: count > 0 ? data.errorCount / count : 0,
      impact: totalDuration / maxTotal,
    };
  });
}

/** Compute span type time breakdown for a service */
export function mockGetSpanTypeBreakdown(
  serviceName: string
): SpanTypeBreakdown[] {
  let httpTime = 0;
  let dbTime = 0;
  let otherTime = 0;

  for (const detail of Object.values(mockData.traceDetails)) {
    for (const span of detail.spans) {
      if (span.serviceName !== serviceName) continue;

      const op = span.operationName.toUpperCase();
      if (
        op.startsWith("GET") ||
        op.startsWith("POST") ||
        op.startsWith("PUT") ||
        op.startsWith("DELETE") ||
        op.startsWith("PATCH") ||
        op.startsWith("HTTP")
      ) {
        httpTime += span.duration;
      } else if (op.startsWith("SELECT") || op.startsWith("INSERT") || op.startsWith("UPDATE") || op.startsWith("DELETE FROM")) {
        dbTime += span.duration;
      } else {
        otherTime += span.duration;
      }
    }
  }

  const total = httpTime + dbTime + otherTime;
  if (total === 0) return [{ type: "other", percentage: 100 }];

  const result: SpanTypeBreakdown[] = [];
  if (httpTime > 0) result.push({ type: "HTTP", percentage: (httpTime / total) * 100 });
  if (dbTime > 0) result.push({ type: "DB", percentage: (dbTime / total) * 100 });
  if (otherTime > 0) result.push({ type: "other", percentage: (otherTime / total) * 100 });
  return result;
}

/** Compute latency distribution histogram buckets (log scale) */
export function mockGetLatencyDistribution(
  traces: TraceSummary[]
): LatencyBucket[] {
  // Log-scale bucket boundaries in μs
  const boundaries = [
    0, 1000, 2000, 5000, 10000, 20000, 50000, 100000, 200000, 500000,
    1000000, 2000000, 5000000,
  ];

  const buckets: LatencyBucket[] = boundaries.slice(0, -1).map((start, i) => ({
    rangeStart: start,
    rangeEnd: boundaries[i + 1],
    count: 0,
  }));

  // Add overflow bucket
  buckets.push({
    rangeStart: boundaries[boundaries.length - 1],
    rangeEnd: Infinity,
    count: 0,
  });

  for (const trace of traces) {
    for (let i = buckets.length - 1; i >= 0; i--) {
      if (trace.duration >= buckets[i].rangeStart) {
        buckets[i].count++;
        break;
      }
    }
  }

  // Filter out empty buckets at the edges
  let start = 0;
  let end = buckets.length - 1;
  while (start < end && buckets[start].count === 0) start++;
  while (end > start && buckets[end].count === 0) end--;

  return buckets.slice(start, end + 1);
}
