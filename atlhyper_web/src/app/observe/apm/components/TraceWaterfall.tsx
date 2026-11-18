"use client";

import { useState, useMemo, useCallback } from "react";
import { Copy, Check } from "lucide-react";
import type { TraceDetail, TraceSummary, Span } from "@/types/model/apm";
import Link from "next/link";
import { ScrollText } from "lucide-react";
import { logsLinkForTrace } from "@/lib/signal-link";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { getLatencyDistribution } from "@/datasource/apm";
import type { LatencyRange } from "../apm-view";
import { LatencyDistribution } from "./LatencyDistribution";
import { SERVICE_COLORS, buildSpanTree, flattenTree, focusSpanIds } from "./waterfall-utils";
import { SpanRow } from "./SpanRow";
import { SpanDrawer } from "./SpanDrawer";
import { SpanLogs } from "./SpanLogs";
import { ServiceFocusLegend, TraceSampleNavigator } from "./TraceWaterfallControls";
import { TraceMetadata } from "./TraceMetadata";

interface TraceWaterfallProps {
  t: ApmTranslations;
  trace: TraceDetail;
  allTraces: TraceSummary[];
  currentTraceIndex: number;
  /** 进入路径的服务：作为初始 focus（高亮该服务的入口 span 及后代，其余淡化） */
  initialFocusService?: string;
  onNavigateTrace: (index: number) => void;
  /**
   * 延迟分布 brush 选区：样本导航只在选中区间内的 trace 上移动（ES 同款下钻）。
   * 由上层持有 —— 本组件按 traceId 重新挂载，选区要是存在这里，翻到下一条样本就丢了
   */
  latencyFilter: LatencyRange | null;
  onLatencyFilterChange: (range: LatencyRange | null) => void;
}

export function TraceWaterfall({
  t,
  trace,
  allTraces,
  currentTraceIndex,
  initialFocusService,
  onNavigateTrace,
  latencyFilter,
  onLatencyFilterChange,
}: TraceWaterfallProps) {
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);
  const [collapsedSpans, setCollapsedSpans] = useState<Set<string>>(new Set());
  const [copiedId, setCopiedId] = useState(false);
  const [activeTab, setActiveTab] = useState(0);
  // focus = 高亮而非裁剪：完整树永远渲染，非 focus 服务淡化。
  // 初始值来自进入路径（从哪个服务点进来就聚焦谁），图例可切换/清除。
  const [focusService, setFocusService] = useState<string | null>(initialFocusService ?? null);

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

  const focusedIds = useMemo(() => {
    if (!focusService) return null;
    const ids = focusSpanIds(trace, focusService);
    return ids.size > 0 ? ids : null; // 无匹配 = 不淡化任何行
  }, [trace, focusService]);

  // 各服务的 self time 汇总 —— 直接回答「在网关内花了多久 / 在 user 内花了多久」。
  // 用 self time 而非 span 总时长：后者层层嵌套会重复计入等待下游的时间，
  // 三个 span 各 373/372/367ms 加起来 1112ms 远超 trace 的 373ms，无法解读。
  const serviceStats = useMemo(() => {
    const m = new Map<string, { spans: number; selfMs: number }>();
    const walk = (n: (typeof tree)[number]) => {
      const cur = m.get(n.span.serviceName) ?? { spans: 0, selfMs: 0 };
      cur.spans += 1;
      cur.selfMs += n.selfDurationMs;
      m.set(n.span.serviceName, cur);
      n.children.forEach(walk);
    };
    tree.forEach(walk);
    return m;
  }, [tree]);

  // 占比分母 = Σ(各服务 self time)，**不是** trace 墙钟时长。
  // 并行调用时各服务 self 之和会超过墙钟（实测：19 span / 墙钟 153ms，
  // media 的 anime+threed+av 等并行调用 self 合计 242ms → 用墙钟做分母
  // 会显示「158%」，四服务相加 257%）。改用 Σself 后各项相加恒为 100%，
  // 语义也更有用：「61% 的执行时间花在 media」是可行动结论。
  const grandSelfMs = useMemo(
    () => [...serviceStats.values()].reduce((sum, v) => sum + v.selfMs, 0),
    [serviceStats],
  );

  // Convert ISO timestamps to ms for relative positioning
  const spanTimesMs = useMemo(() => {
    return trace.spans.map((s) => new Date(s.timestamp).getTime());
  }, [trace.spans]);

  const traceStartMs = useMemo(() => Math.min(...spanTimesMs), [spanTimesMs]);
  const traceEndMs = useMemo(() => {
    return Math.max(...trace.spans.map((s, i) => spanTimesMs[i] + s.durationMs));
  }, [trace.spans, spanTimesMs]);
  const traceDurationMs = traceEndMs - traceStartMs;

  const latencyBuckets = useMemo(
    () => getLatencyDistribution(allTraces),
    [allTraces]
  );

  // 筛选后的样本序列（保留原始 index 供导航回调）
  const sampleEntries = useMemo(() => {
    const all = allTraces.map((tr, i) => ({ tr, i }));
    if (!latencyFilter) return all;
    return all.filter(({ tr }) => tr.durationMs >= latencyFilter.startMs && tr.durationMs < latencyFilter.endMs);
  }, [allTraces, latencyFilter]);
  const samplePos = sampleEntries.findIndex((e) => e.i === currentTraceIndex);

  const handleBrushRange = useCallback((range: LatencyRange | null) => {
    onLatencyFilterChange(range);
    if (range) {
      // 当前样本不在选区内时跳到选区内第一条
      const first = allTraces.findIndex(
        (tr) => tr.durationMs >= range.startMs && tr.durationMs < range.endMs
      );
      const cur = allTraces[currentTraceIndex];
      const inRange = cur && cur.durationMs >= range.startMs && cur.durationMs < range.endMs;
      if (!inRange && first >= 0) onNavigateTrace(first);
    }
  }, [allTraces, currentTraceIndex, onNavigateTrace, onLatencyFilterChange]);

  const highlightBucket = useMemo(() => {
    if (allTraces.length === 0 || currentTraceIndex < 0) return undefined;
    const currentDuration = allTraces[currentTraceIndex]?.durationMs ?? 0;
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

  const tickCount = 6;
  const ticks = Array.from({ length: tickCount }, (_, i) => (i / (tickCount - 1)) * traceDurationMs);

  return (
    <div className="space-y-4">
      {/* Latency Distribution */}
      <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
        <LatencyDistribution
          title={t.latencyDistribution}
          totalTraces={allTraces.length}
          buckets={latencyBuckets}
          highlightBucket={highlightBucket}
          onSelectRange={handleBrushRange}
        />
        {latencyFilter && (
          <div className="flex items-center gap-2 mt-2 text-xs">
            <span className="text-primary">
              {t.latencyFilterActive}: {formatDurationMs(latencyFilter.startMs)} – {latencyFilter.endMs === Number.MAX_VALUE ? "∞" : formatDurationMs(latencyFilter.endMs)}
              <span className="text-muted ml-1.5">({sampleEntries.length} traces)</span>
            </span>
            <button onClick={() => onLatencyFilterChange(null)} className="text-muted hover:text-default underline">
              {t.clearFilter}
            </button>
          </div>
        )}
      </div>

      {/* Trace sample navigation */}
      <TraceSampleNavigator
        t={t}
        sampleEntries={sampleEntries}
        samplePos={samplePos}
        currentTraceSummary={allTraces[currentTraceIndex]}
        onNavigateTrace={onNavigateTrace}
      />

      {/* Trace waterfall */}
      <div className="border border-[var(--border-color)] rounded-xl bg-card overflow-hidden">
        {/* Trace ID header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted">Trace ID:</span>
            <code className="text-xs text-default font-mono bg-[var(--hover-bg)] px-2 py-0.5 rounded">
              {trace.traceId}
            </code>
            <button onClick={copyTraceId} className="p-1 rounded hover:bg-[var(--hover-bg)] transition-colors">
              {copiedId ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5 text-muted" />}
            </button>
            {/* 反向链路：Logs 早就能跳到 APM，反过来一直缺 —— 看瀑布图时想确认
                某一段到底打了什么日志，只能自己去 Logs 页粘 traceId */}
            <Link
              href={logsLinkForTrace(trace.traceId)}
              className="inline-flex items-center gap-1 text-xs text-blue-500 hover:underline"
            >
              <ScrollText className="w-3 h-3" />
              {t.viewLogs}
            </Link>
          </div>
          <span className="text-xs text-muted">
            {trace.spanCount} {t.spans} | {trace.serviceCount} {t.serviceCount} | {formatDurationMs(trace.durationMs)}
          </span>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-[var(--border-color)] px-4">
          {[t.timeline, t.metadata, t.logs].map((label, i) => (
            <button
              key={label}
              onClick={() => setActiveTab(i)}
              className={`px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors ${
                activeTab === i ? "text-primary border-primary" : "text-muted border-transparent hover:text-default"
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {/* Metadata tab content */}
        {activeTab === 1 && (
          <TraceMetadata t={t} trace={trace} />
        )}

        {/* Logs tab content */}
        {activeTab === 2 && (
          <SpanLogs t={t} traceId={trace.traceId} serviceColorMap={serviceColorMap} />
        )}

        {/* Timeline tab content */}
        {activeTab === 0 && <>
        {/* Service legend —— 可点击控件：toggle 服务 focus */}
        <ServiceFocusLegend
          t={t}
          serviceColorMap={serviceColorMap}
          serviceStats={serviceStats}
          grandSelfMs={grandSelfMs}
          focusService={focusService}
          onFocusServiceChange={setFocusService}
        />

        {/* Timeline header —— 与 SpanRow 的轨道共用坐标系。
            左侧留出与最浅层缩进相同的宽度（折叠控件 22px + padding 8px）；
            深层行的缩进由其自身 flex 宽度产生，轨道仍按百分比对齐，刻度不失准 */}
        <div className="py-2 border-b border-[var(--border-color)]">
          <div className="flex">
            <div className="flex-shrink-0" style={{ width: 30 }} />
            <div className="flex-1 flex justify-between text-[10px] text-muted font-mono pr-4">
              {ticks.map((tick, i) => (
                <span key={i}>{formatDurationMs(tick)}</span>
              ))}
            </div>
          </div>
        </div>

        {/* Waterfall rows */}
        <div className="overflow-auto">
          {flatSpans.map(({ node, ancestorHasNext, isLastChild }) => (
            <SpanRow
              key={node.span.spanId}
              node={node}
              ancestorHasNext={ancestorHasNext}
              isLastChild={isLastChild}
              serviceColorMap={serviceColorMap}
              traceStartMs={traceStartMs}
              traceDurationMs={traceDurationMs}
              isSelected={selectedSpan?.spanId === node.span.spanId}
              isCollapsed={collapsedSpans.has(node.span.spanId)}
              focusedIds={focusedIds}
              onSelect={setSelectedSpan}
              onToggleCollapse={toggleCollapse}
            />
          ))}
        </div>
        </>}
      </div>

      {/* Span detail drawer */}
      <SpanDrawer
        t={t}
        span={selectedSpan}
        trace={trace}
        serviceColorMap={serviceColorMap}
        traceStartMs={traceStartMs}
        onClose={() => setSelectedSpan(null)}
      />
    </div>
  );
}
