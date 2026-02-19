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

const SERVICE_COLORS = [
  "#3b82f6", // blue
  "#10b981", // emerald
  "#f59e0b", // amber
  "#8b5cf6", // purple
  "#ef4444", // red
  "#06b6d4", // cyan
  "#f97316", // orange
  "#6366f1", // indigo
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

function getSpanLabel(span: Span): { icon: string; text: string } {
  const op = span.operationName;
  const dur = formatDuration(span.duration);

  // HTTP span
  if (/^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b/.test(op)) {
    const statusTag = span.tags.find((t) => t.key === "http.status_code");
    const status = statusTag ? `${statusTag.value.startsWith("2") ? "2xx" : statusTag.value}` : "";
    return {
      icon: "HTTP",
      text: `${status ? status + " " : ""}${op} ${dur}`,
    };
  }

  // DB span
  if (/^(SELECT|INSERT|UPDATE|DELETE FROM)\b/i.test(op)) {
    return { icon: "DB", text: `${op} ${dur}` };
  }

  // Default
  return { icon: "", text: `${op} ${dur}` };
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

        {/* Waterfall + detail split */}
        <div className="flex">
          {/* Waterfall */}
          <div className="flex-1 min-w-0 overflow-auto">
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

              const { icon, text } = getSpanLabel(span);
              // Determine if label fits inside bar
              const barIsWide = width > 15; // if bar occupies >15% of width, label inside

              return (
                <div
                  key={span.spanId}
                  onClick={() => setSelectedSpan(span)}
                  className={`flex items-center cursor-pointer border-b border-[var(--border-color)]/20 transition-colors ${
                    isSelected
                      ? "bg-primary/5"
                      : "hover:bg-[var(--hover-bg)]"
                  }`}
                  style={{ height: 36 }}
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
                  <div className="flex-1 relative" style={{ height: 28 }}>
                    <div
                      className="absolute top-0 rounded-sm overflow-hidden"
                      style={{
                        left: `${offset}%`,
                        width: `${Math.max(width, 0.3)}%`,
                        height: 28,
                        backgroundColor: color,
                      }}
                    >
                      {/* Label inside bar */}
                      {barIsWide && (
                        <div className="absolute inset-0 flex items-center px-2 overflow-hidden">
                          <span className="text-[11px] text-white font-medium truncate whitespace-nowrap">
                            {icon && (
                              <span className="opacity-80 mr-1">{icon}</span>
                            )}
                            {text}
                          </span>
                        </div>
                      )}
                    </div>
                    {/* Label outside bar (when bar is narrow) */}
                    {!barIsWide && (
                      <div
                        className="absolute flex items-center text-[11px] text-default whitespace-nowrap"
                        style={{
                          left: `${offset + Math.max(width, 0.3) + 0.5}%`,
                          top: 0,
                          height: 28,
                        }}
                      >
                        {icon && (
                          <span className="text-muted mr-1">{icon}</span>
                        )}
                        <span className="truncate">{text}</span>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Span detail panel */}
          {selectedSpan && (
            <div className="w-[320px] flex-shrink-0 border-l border-[var(--border-color)] overflow-auto bg-[var(--background)]">
              <div className="p-4 space-y-4">
                <h3 className="text-sm font-semibold text-default">
                  {t.spanDetail}
                </h3>

                <div className="space-y-2">
                  <InfoRow label={t.serviceName} value={selectedSpan.serviceName} />
                  <InfoRow label={t.operationName} value={selectedSpan.operationName} />
                  <InfoRow label={t.duration} value={formatDuration(selectedSpan.duration)} />
                  <InfoRow
                    label={t.status}
                    value={
                      <span className={selectedSpan.status === "error" ? "text-red-500" : "text-emerald-500"}>
                        {selectedSpan.status}
                      </span>
                    }
                  />
                  <InfoRow
                    label="Span ID"
                    value={<span className="font-mono text-[11px]">{selectedSpan.spanId}</span>}
                  />
                  {selectedSpan.parentSpanId && (
                    <InfoRow
                      label={t.parentSpan}
                      value={<span className="font-mono text-[11px]">{selectedSpan.parentSpanId}</span>}
                    />
                  )}
                </div>

                {/* Tags */}
                <div>
                  <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
                    {t.tags}
                  </h4>
                  {selectedSpan.tags.length === 0 ? (
                    <p className="text-xs text-muted">{t.noTags}</p>
                  ) : (
                    <div className="space-y-1">
                      {selectedSpan.tags.map((tag) => (
                        <div
                          key={tag.key}
                          className="flex items-start gap-2 text-xs py-1 px-2 rounded-lg bg-[var(--hover-bg)]"
                        >
                          <span className="text-muted flex-shrink-0 min-w-[90px]">
                            {tag.key}
                          </span>
                          <span className="text-default font-mono break-all">
                            {tag.value}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2 text-xs">
      <span className="text-muted min-w-[80px] flex-shrink-0">{label}</span>
      <span className="text-default">{value}</span>
    </div>
  );
}
