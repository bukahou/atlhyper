import type { TraceDetail, Span } from "@/types/model/apm";

/**
 * filterTraceForService - Focus a trace on a specific service.
 *
 * Keeps only the entry spans of the given service and all their descendants.
 * Upper-layer callers (e.g. gateway) are filtered out so the entry span
 * becomes the new root.
 */
export function filterTraceForService(
  trace: TraceDetail,
  focusService: string,
): TraceDetail {
  const { spans } = trace;
  if (spans.length === 0) return trace;

  // Build lookup tables
  const spanMap = new Map<string, Span>();
  const childrenMap = new Map<string, string[]>();
  for (const span of spans) {
    spanMap.set(span.spanId, span);
    if (span.parentSpanId) {
      const list = childrenMap.get(span.parentSpanId) ?? [];
      list.push(span.spanId);
      childrenMap.set(span.parentSpanId, list);
    }
  }

  // Find entry spans: serviceName matches but parent's serviceName does not (or no parent)
  const entryIds: string[] = [];
  for (const span of spans) {
    if (span.serviceName !== focusService) continue;
    const parent = span.parentSpanId
      ? spanMap.get(span.parentSpanId)
      : undefined;
    if (!parent || parent.serviceName !== focusService) {
      entryIds.push(span.spanId);
    }
  }

  // Fallback: show full trace when no match
  if (entryIds.length === 0) return trace;

  // Collect entry spans and all descendants
  const included = new Set<string>();
  const collect = (id: string) => {
    included.add(id);
    for (const childId of childrenMap.get(id) ?? []) collect(childId);
  };
  entryIds.forEach(collect);

  // Filter + clear parentSpanId on entry spans (make them roots)
  const entrySet = new Set(entryIds);
  const filtered = spans
    .filter((s) => included.has(s.spanId))
    .map((s) =>
      entrySet.has(s.spanId) ? { ...s, parentSpanId: "" } : s,
    );

  return {
    traceId: trace.traceId,
    spans: filtered,
    spanCount: filtered.length,
    serviceCount: new Set(filtered.map((s) => s.serviceName)).size,
    durationMs: trace.durationMs,
  };
}
