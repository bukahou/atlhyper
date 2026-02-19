"use client";

import { useState, useMemo, useCallback } from "react";
import {
  ChevronRight,
  ChevronDown,
  ChevronLeft,
  ChevronsLeft,
  ChevronsRight,
  Copy,
  Check,
  X,
} from "lucide-react";
import type { TraceDetail, TraceSummary, Span, LatencyBucket } from "@/api/apm";
import type { ApmTranslations } from "@/types/i18n";
import { mockGetLatencyDistribution } from "@/api/apm-mock";
import { LatencyDistribution } from "./LatencyDistribution";

interface TraceWaterfallProps {
  t: ApmTranslations;
  trace: TraceDetail;
  allTraces: TraceSummary[]; // all traces for this service (for sampling navigation)
  currentTraceIndex: number;
  onNavigateTrace: (index: number) => void;
}

function formatDuration(us: number): string {
  if (us < 1000) return `${us}μs`;
  if (us < 1_000_000) return `${(us / 1000).toFixed(1)}ms`;
  return `${(us / 1_000_000).toFixed(2)}s`;
}

function formatTimeAgo(us: number): string {
  const now = Date.now() * 1000; // current time in μs
  const diffMs = (now - us) / 1000;
  const diffMin = diffMs / 60000;
  const diffHour = diffMin / 60;
  const diffDay = diffHour / 24;

  if (diffDay >= 1) return `${Math.floor(diffDay)}d ago`;
  if (diffHour >= 1) return `${Math.floor(diffHour)}h ago`;
  if (diffMin >= 1) return `${Math.floor(diffMin)}m ago`;
  return "just now";
}

// Softer palette — Tailwind -400 shades for a cleaner, less garish look
const SERVICE_COLORS = [
  "#60a5fa", // blue-400
  "#34d399", // emerald-400
  "#fbbf24", // amber-400
  "#a78bfa", // violet-400
  "#f87171", // red-400
  "#22d3ee", // cyan-400
  "#fb923c", // orange-400
  "#818cf8", // indigo-400
];

interface SpanNode {
  span: Span;
  children: SpanNode[];
  depth: number;
}

function buildSpanTree(spans: Span[]): SpanNode[] {
  const spanMap = new Map<string, SpanNode>();
  const roots: SpanNode[] = [];

  for (const span of spans) {
    spanMap.set(span.spanId, { span, children: [], depth: 0 });
  }

  for (const span of spans) {
    const node = spanMap.get(span.spanId)!;
    if (span.parentSpanId && spanMap.has(span.parentSpanId)) {
      spanMap.get(span.parentSpanId)!.children.push(node);
    } else {
      roots.push(node);
    }
  }

  function setDepth(node: SpanNode, depth: number) {
    node.depth = depth;
    node.children.forEach((c) => setDepth(c, depth + 1));
  }
  roots.forEach((r) => setDepth(r, 0));
  return roots;
}

function flattenTree(nodes: SpanNode[], collapsed: Set<string>): SpanNode[] {
  const result: SpanNode[] = [];
  function walk(node: SpanNode) {
    result.push(node);
    if (!collapsed.has(node.span.spanId)) {
      node.children.forEach(walk);
    }
  }
  nodes.forEach(walk);
  return result;
}

function countDescendants(node: SpanNode): number {
  let count = node.children.length;
  for (const child of node.children) count += countDescendants(child);
  return count;
}

function getSpanLabel(span: Span): { op: string; dur: string } {
  return { op: span.operationName, dur: formatDuration(span.duration) };
}

export function TraceWaterfall({
  t,
  trace,
  allTraces,
  currentTraceIndex,
  onNavigateTrace,
}: TraceWaterfallProps) {
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);
  const [collapsedSpans, setCollapsedSpans] = useState<Set<string>>(new Set());
  const [copiedId, setCopiedId] = useState(false);

  const serviceColorMap = useMemo(() => {
    const services = [...new Set(trace.spans.map((s) => s.serviceName))];
    const map = new Map<string, string>();
    services.forEach((svc, i) => {
      map.set(svc, SERVICE_COLORS[i % SERVICE_COLORS.length]);
    });
    return map;
  }, [trace.spans]);

  const tree = useMemo(() => buildSpanTree(trace.spans), [trace.spans]);
  const flatSpans = useMemo(
    () => flattenTree(tree, collapsedSpans),
    [tree, collapsedSpans]
  );

  const traceStart = useMemo(
    () => Math.min(...trace.spans.map((s) => s.startTime)),
    [trace.spans]
  );
  const traceEnd = useMemo(
    () => Math.max(...trace.spans.map((s) => s.startTime + s.duration)),
    [trace.spans]
  );
  const traceDuration = traceEnd - traceStart;

  // Latency distribution for all traces
  const latencyBuckets = useMemo(
    () => mockGetLatencyDistribution(allTraces),
    [allTraces]
  );

  // Find which bucket the current trace falls into
  const highlightBucket = useMemo(() => {
    if (allTraces.length === 0 || currentTraceIndex < 0) return undefined;
    const currentDuration = allTraces[currentTraceIndex]?.duration ?? 0;
    for (let i = latencyBuckets.length - 1; i >= 0; i--) {
      if (currentDuration >= latencyBuckets[i].rangeStart) return i;
    }
    return 0;
  }, [allTraces, currentTraceIndex, latencyBuckets]);

  const toggleCollapse = useCallback((spanId: string) => {
    setCollapsedSpans((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) next.delete(spanId);
      else next.add(spanId);
      return next;
    });
  }, []);

  const copyTraceId = () => {
    navigator.clipboard.writeText(trace.traceId);
    setCopiedId(true);
    setTimeout(() => setCopiedId(false), 2000);
  };

  // Current trace summary
  const currentTraceSummary = allTraces[currentTraceIndex];

  // Timeline tick marks
  const tickCount = 6;
  const ticks = Array.from({ length: tickCount }, (_, i) => (i / (tickCount - 1)) * traceDuration);

  return (
    <div className="space-y-4">
      {/* Latency Distribution */}
      <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
        <LatencyDistribution
          title={t.latencyDistribution}
          totalTraces={allTraces.length}
          buckets={latencyBuckets}
          highlightBucket={highlightBucket}
        />
      </div>

      {/* Trace sample navigation */}
      <div className="flex items-center justify-between border border-[var(--border-color)] rounded-xl px-4 py-3 bg-card">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium text-default">{t.traceSample}</span>
          <div className="flex items-center gap-1">
            <button
              onClick={() => onNavigateTrace(0)}
              disabled={currentTraceIndex <= 0}
              className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors"
            >
              <ChevronsLeft className="w-4 h-4 text-muted" />
            </button>
            <button
              onClick={() => onNavigateTrace(currentTraceIndex - 1)}
              disabled={currentTraceIndex <= 0}
              className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors"
            >
              <ChevronLeft className="w-4 h-4 text-muted" />
            </button>
            <span className="text-sm text-default px-2 min-w-[60px] text-center">
              {currentTraceIndex + 1} / {allTraces.length}
            </span>
            <button
              onClick={() => onNavigateTrace(currentTraceIndex + 1)}
              disabled={currentTraceIndex >= allTraces.length - 1}
              className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors"
            >
              <ChevronRight className="w-4 h-4 text-muted" />
            </button>
            <button
              onClick={() => onNavigateTrace(allTraces.length - 1)}
              disabled={currentTraceIndex >= allTraces.length - 1}
              className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 transition-colors"
            >
              <ChevronsRight className="w-4 h-4 text-muted" />
            </button>
          </div>
        </div>
        {currentTraceSummary && (
          <div className="text-xs text-muted">
            {formatTimeAgo(currentTraceSummary.startTime)} | {formatDuration(currentTraceSummary.duration)}
          </div>
        )}
      </div>

      {/* Trace info + tabs */}
      <div className="border border-[var(--border-color)] rounded-xl bg-card overflow-hidden">
        {/* Trace ID header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted">Trace ID:</span>
            <code className="text-xs text-default font-mono bg-[var(--hover-bg)] px-2 py-0.5 rounded">
              {trace.traceId}
            </code>
            <button
              onClick={copyTraceId}
              className="p-1 rounded hover:bg-[var(--hover-bg)] transition-colors"
            >
              {copiedId ? (
                <Check className="w-3.5 h-3.5 text-emerald-500" />
              ) : (
                <Copy className="w-3.5 h-3.5 text-muted" />
              )}
            </button>
          </div>
          <span className="text-xs text-muted">
            {trace.spans.length} {t.spans} | {formatDuration(traceDuration)}
          </span>
        </div>

        {/* Tabs — only timeline active */}
        <div className="flex border-b border-[var(--border-color)] px-4">
          {[t.timeline, t.metadata, t.logs].map((label, i) => (
            <button
              key={label}
              disabled={i > 0}
              className={`px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors ${
                i === 0
                  ? "text-primary border-primary"
                  : "text-muted/50 border-transparent cursor-not-allowed"
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {/* Service legend */}
        <div className="flex flex-wrap gap-3 px-4 py-2 border-b border-[var(--border-color)]">
          {[...serviceColorMap.entries()].map(([svc, color]) => (
            <div key={svc} className="flex items-center gap-1.5 text-xs">
              <span
                className="w-2.5 h-2.5 rounded-full"
                style={{ backgroundColor: color }}
              />
              <span className="text-muted">{svc}</span>
            </div>
          ))}
        </div>

        {/* Timeline header */}
        <div className="px-4 py-2 border-b border-[var(--border-color)]">
          <div className="flex">
            <div className="w-[80px] flex-shrink-0" />
            <div className="flex-1 flex justify-between text-[10px] text-muted">
              {ticks.map((tick, i) => (
                <span key={i}>{formatDuration(tick)}</span>
              ))}
            </div>
          </div>
        </div>

        {/* Waterfall */}
        <div className="overflow-auto">
            {flatSpans.map((node) => {
              const { span, depth } = node;
              const color = serviceColorMap.get(span.serviceName) ?? "#94a3b8";
              const offset =
                traceDuration > 0
                  ? ((span.startTime - traceStart) / traceDuration) * 100
                  : 0;
              const width =
                traceDuration > 0
                  ? (span.duration / traceDuration) * 100
                  : 100;
              const isSelected = selectedSpan?.spanId === span.spanId;
              const childCount = countDescendants(node);
              const hasChildren = node.children.length > 0;
              const isCollapsed = collapsedSpans.has(span.spanId);

              const { op, dur } = getSpanLabel(span);
              const barIsWide = width > 15;
              const barH = 24;

              return (
                <div
                  key={span.spanId}
                  onClick={() => setSelectedSpan(span)}
                  className={`flex items-center cursor-pointer border-b border-[var(--border-color)]/20 transition-colors ${
                    isSelected
                      ? "bg-primary/5"
                      : "hover:bg-[var(--hover-bg)]"
                  }`}
                  style={{ height: 34 }}
                >
                  {/* Left column: collapse toggle */}
                  <div
                    className="w-[80px] flex-shrink-0 flex items-center gap-1 text-xs px-2"
                    style={{ paddingLeft: `${depth * 12 + 8}px` }}
                  >
                    {hasChildren ? (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleCollapse(span.spanId);
                        }}
                        className="flex items-center gap-0.5 p-0.5 rounded hover:bg-[var(--hover-bg)]"
                      >
                        {isCollapsed ? (
                          <ChevronRight className="w-3 h-3 text-muted" />
                        ) : (
                          <ChevronDown className="w-3 h-3 text-muted" />
                        )}
                        <span className="text-[10px] text-muted">{childCount}</span>
                      </button>
                    ) : (
                      <span className="w-[18px]" />
                    )}
                  </div>

                  {/* Timeline bar */}
                  <div className="flex-1 relative" style={{ height: barH }}>
                    {/* Bar: translucent fill + solid left accent border */}
                    <div
                      className="absolute top-0"
                      style={{
                        left: `${offset}%`,
                        width: `${Math.max(width, 0.3)}%`,
                        height: barH,
                        borderRadius: 4,
                        borderLeft: `3px solid ${color}`,
                        background: `${color}30`,
                      }}
                    >
                      {/* Label inside bar */}
                      {barIsWide && (
                        <div className="absolute inset-0 flex items-center gap-1 px-2 overflow-hidden">
                          <span className="text-[11px] font-medium truncate" style={{ color }}>
                            {op}
                          </span>
                          <span className="text-[10px] flex-shrink-0" style={{ color: `${color}99` }}>
                            {dur}
                          </span>
                        </div>
                      )}
                    </div>
                    {/* Label outside bar (when bar is narrow) */}
                    {!barIsWide && (
                      <div
                        className="absolute flex items-center gap-1.5 whitespace-nowrap"
                        style={{
                          left: `${offset + Math.max(width, 0.3) + 0.5}%`,
                          top: 0,
                          height: barH,
                        }}
                      >
                        <span className="text-[11px] text-default truncate">
                          {op}
                        </span>
                        <span className="text-[10px] text-muted flex-shrink-0">
                          {dur}
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
        </div>
      </div>

      {/* Span detail drawer (overlay) */}
      <SpanDrawer
        t={t}
        span={selectedSpan}
        trace={trace}
        serviceColorMap={serviceColorMap}
        traceStart={traceStart}
        onClose={() => setSelectedSpan(null)}
      />
    </div>
  );
}

// ============================================================
// Tag grouping helpers
// ============================================================

function groupTags(tags: { key: string; value: string }[]) {
  const http: typeof tags = [];
  const db: typeof tags = [];
  const server: typeof tags = [];
  const other: typeof tags = [];

  for (const tag of tags) {
    if (tag.key.startsWith("http.") || tag.key.startsWith("url.")) http.push(tag);
    else if (tag.key.startsWith("db.")) db.push(tag);
    else if (tag.key.startsWith("server.")) server.push(tag);
    else other.push(tag);
  }
  return { http, db, server, other };
}

// ============================================================
// SpanDrawer — right-side overlay drawer
// ============================================================

function SpanDrawer({
  t,
  span,
  trace,
  serviceColorMap,
  traceStart,
  onClose,
}: {
  t: ApmTranslations;
  span: Span | null;
  trace: TraceDetail;
  serviceColorMap: Map<string, string>;
  traceStart: number;
  onClose: () => void;
}) {
  if (!span) return null;

  const color = serviceColorMap.get(span.serviceName) ?? "#94a3b8";

  // Compute self-time: span.duration minus direct children duration
  const childDuration = trace.spans
    .filter((s) => s.parentSpanId === span.spanId)
    .reduce((sum, s) => sum + s.duration, 0);
  const selfTime = span.duration - childDuration;
  const startOffset = span.startTime - traceStart;

  const { http, db, server, other } = groupTags(span.tags);

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/30 z-40 transition-opacity"
        onClick={onClose}
      />
      {/* Drawer */}
      <div className="fixed right-0 top-0 h-full w-[520px] max-w-[90vw] z-50 bg-card border-l border-[var(--border-color)] shadow-2xl overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 z-10 bg-card border-b border-[var(--border-color)] px-5 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <span
              className="w-3 h-3 rounded-full flex-shrink-0"
              style={{ backgroundColor: color }}
            />
            <h3 className="text-sm font-semibold text-default truncate">
              {t.spanDetail}
            </h3>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-[var(--hover-bg)] transition-colors flex-shrink-0"
          >
            <X className="w-4 h-4 text-muted" />
          </button>
        </div>

        <div className="p-5 space-y-5">
          {/* Overview grid */}
          <section>
            <SectionHeader title={t.overview} />
            <div className="grid grid-cols-2 gap-2.5">
              <MetricCard label={t.serviceName}>
                <div className="flex items-center gap-1.5">
                  <span
                    className="w-2 h-2 rounded-full flex-shrink-0"
                    style={{ backgroundColor: color }}
                  />
                  <span className="text-sm font-medium text-default truncate">
                    {span.serviceName}
                  </span>
                </div>
              </MetricCard>
              <MetricCard label={t.status}>
                <span
                  className={`text-sm font-medium ${
                    span.status === "error" ? "text-red-400" : "text-emerald-400"
                  }`}
                >
                  {span.status}
                </span>
              </MetricCard>
              <MetricCard label={t.duration} fullWidth>
                <span className="text-sm font-medium text-default">
                  {formatDuration(span.duration)}
                </span>
              </MetricCard>
              <MetricCard label={t.selfTime}>
                <span className="text-sm font-medium text-default">
                  {formatDuration(selfTime)}
                </span>
                <span className="text-[10px] text-muted ml-1">
                  ({span.duration > 0 ? Math.round((selfTime / span.duration) * 100) : 0}%)
                </span>
              </MetricCard>
              <MetricCard label={t.childTime}>
                <span className="text-sm font-medium text-default">
                  {formatDuration(childDuration)}
                </span>
              </MetricCard>
              <MetricCard label={t.startOffset} fullWidth>
                <span className="text-sm font-mono text-default">
                  +{formatDuration(startOffset)}
                </span>
              </MetricCard>
            </div>
          </section>

          {/* Operation */}
          <section>
            <SectionHeader title={t.operationName} />
            <div className="px-3 py-2.5 rounded-lg bg-[var(--hover-bg)] text-sm font-mono text-default break-all">
              {span.operationName}
            </div>
          </section>

          {/* IDs */}
          <section>
            <SectionHeader title={t.spanIds} />
            <div className="space-y-1.5">
              <IdRow label="Span ID" value={span.spanId} />
              {span.parentSpanId && (
                <IdRow label={t.parentSpan} value={span.parentSpanId} />
              )}
              <IdRow label="Trace ID" value={trace.traceId} />
            </div>
          </section>

          {/* Tag groups */}
          {http.length > 0 && (
            <section>
              <SectionHeader title={t.httpAttributes} />
              <TagTable tags={http} />
            </section>
          )}

          {db.length > 0 && (
            <section>
              <SectionHeader title={t.dbAttributes} />
              <TagTable tags={db} />
            </section>
          )}

          {server.length > 0 && (
            <section>
              <SectionHeader title={t.serverAttributes} />
              <TagTable tags={server} />
            </section>
          )}

          {other.length > 0 && (
            <section>
              <SectionHeader title={t.otherAttributes} />
              <TagTable tags={other} />
            </section>
          )}

          {span.tags.length === 0 && (
            <section>
              <SectionHeader title={t.tags} />
              <p className="text-xs text-muted">{t.noTags}</p>
            </section>
          )}
        </div>
      </div>
    </>
  );
}

// ============================================================
// Drawer sub-components
// ============================================================

function SectionHeader({ title }: { title: string }) {
  return (
    <h4 className="text-[11px] font-semibold text-muted uppercase tracking-wider mb-2">
      {title}
    </h4>
  );
}

function MetricCard({
  label,
  children,
  fullWidth,
}: {
  label: string;
  children: React.ReactNode;
  fullWidth?: boolean;
}) {
  return (
    <div
      className={`px-3 py-2 rounded-lg bg-[var(--hover-bg)] ${fullWidth ? "col-span-2" : ""}`}
    >
      <div className="text-[10px] text-muted mb-0.5">{label}</div>
      <div className="flex items-center">{children}</div>
    </div>
  );
}

function IdRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[var(--hover-bg)] text-xs">
      <span className="text-muted flex-shrink-0 min-w-[80px]">{label}</span>
      <code className="text-default font-mono text-[11px] truncate">{value}</code>
    </div>
  );
}

function TagTable({ tags }: { tags: { key: string; value: string }[] }) {
  return (
    <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
      {tags.map((tag, i) => (
        <div
          key={tag.key}
          className={`flex gap-3 px-3 py-2 text-xs ${
            i < tags.length - 1 ? "border-b border-[var(--border-color)]" : ""
          }`}
        >
          <span className="text-muted flex-shrink-0 min-w-[120px]">
            {tag.key}
          </span>
          <span className="text-default font-mono break-all">
            {tag.value}
          </span>
        </div>
      ))}
    </div>
  );
}
