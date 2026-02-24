"use client";

import { BarChart3, Layers } from "lucide-react";
import type { LatencyDistributionResponse } from "@/types/slo";

export interface LatencyTabTranslations {
  latencyDistribution: string;
  methodBreakdown: string;
  statusCodeBreakdown: string;
  requests: string;
  clearSelection: string;
  noData: string;
}

function timeRangeLabel(tr: string): string {
  switch (tr) {
    case "1d": return "24h";
    case "7d": return "7d";
    case "30d": return "30d";
    default: return tr;
  }
}

export function LatencyTab({ data, timeRange, t }: {
  data: LatencyDistributionResponse | null;
  timeRange: string;
  t: LatencyTabTranslations;
}) {
  if (!data || !data.buckets || data.buckets.length === 0) {
    return (
      <div className="text-center py-8 text-muted text-sm">
        {t.noData}
      </div>
    );
  }

  const badgeLabel = `Ingress · ${timeRangeLabel(timeRange)}`;
  const hasMethods = data.methods && data.methods.length > 0;
  const hasStatusCodes = data.statusCodes && data.statusCodes.length > 0;

  return (
    <div className="flex flex-col lg:flex-row gap-4">
      {/* Left: Histogram (60%) — flex 使卡片与右侧等高 */}
      <div className="lg:w-[60%] flex-shrink-0 flex">
        <LatencyHistogram
          buckets={data.buckets}
          p50={data.p50LatencyMs}
          p95={data.p95LatencyMs}
          p99={data.p99LatencyMs}
          badgeLabel={badgeLabel}
          t={t}
        />
      </div>
      {/* Right: Method + StatusCode (40%) */}
      {(hasMethods || hasStatusCodes) && (
        <div className="flex-1 min-w-0 space-y-4">
          {hasMethods && (
            <MethodChart
              methods={data.methods}
              totalRequests={data.totalRequests}
              badgeLabel={badgeLabel}
              t={t}
            />
          )}
          {hasStatusCodes && (
            <StatusCodeChart
              statusCodes={data.statusCodes}
              totalRequests={data.totalRequests}
              badgeLabel={badgeLabel}
              t={t}
            />
          )}
        </div>
      )}
    </div>
  );
}

// ==================== Histogram (Kibana-style fixed axis) ====================

// 1-2-5 log-scale tick series (evenly spaced on log axis)
const STANDARD_TICKS = [1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000];

// Log-scale mapping: ms value → 0–100% position within [lo, hi]
function logPos(ms: number, lo: number, hi: number): number {
  const v = Math.log10(Math.max(ms, 0.1));
  const a = Math.log10(Math.max(lo, 0.1));
  const b = Math.log10(Math.max(hi, 1));
  if (b <= a) return 50;
  return Math.min(100, Math.max(0, ((v - a) / (b - a)) * 100));
}

// Determine visible axis range, snapped to standard ticks with padding
// p99Hint: 当有 P99 值时，用 P99 * 3 作为上界参考，避免轴太宽
function axisRange(les: number[], p99Hint?: number): [number, number] {
  if (les.length === 0) return [1, 1000];
  const minLe = Math.min(...les);
  const maxLe = Math.max(...les);
  // 用 P99 收紧上界: max(p99*3, maxLe 的较小值) 避免尾部稀疏数据把轴拉太宽
  const effectiveMax = p99Hint ? Math.min(maxLe, Math.max(p99Hint * 3, maxLe * 0.3)) : maxLe;
  let lo = STANDARD_TICKS[0];
  for (const t of STANDARD_TICKS) { if (t <= minLe * 0.6) lo = t; else break; }
  let hi = STANDARD_TICKS[STANDARD_TICKS.length - 1];
  for (let i = STANDARD_TICKS.length - 1; i >= 0; i--) { if (STANDARD_TICKS[i] >= effectiveMax * 1.4) hi = STANDARD_TICKS[i]; else break; }
  return [Math.min(lo, minLe * 0.5), Math.max(hi, effectiveMax * 1.5)];
}

// Select visible tick labels within axis range
function visibleTicks(lo: number, hi: number): number[] {
  return STANDARD_TICKS.filter(t => t >= lo && t <= hi);
}

function tickLabel(ms: number): string { return ms >= 1000 ? `${ms / 1000}s` : `${ms}ms`; }

function LatencyHistogram({ buckets, p50, p95, p99, badgeLabel, t }: {
  buckets: { le: number; count: number }[];
  p50: number;
  p95: number;
  p99: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  // 按 LE 升序排序（后端 map 遍历顺序不保证）
  const sorted = [...buckets].sort((a, b) => a.le - b.le);
  const active = sorted.filter(b => b.count > 0);
  if (active.length === 0) return null;

  const maxCount = Math.max(...active.map(b => b.count), 1);
  const [lo, hi] = axisRange(active.map(b => b.le), p99);
  const ticks = visibleTicks(lo, hi);

  // Each bar covers [prevLe, le] using full bucket list for correct boundaries
  // 过滤掉超出轴范围的桶，避免在不可见区域渲染
  const visibleBuckets = active.filter(b => b.le <= hi * 1.1);
  const bars = visibleBuckets.map((b, i) => {
    const idx = sorted.indexOf(b);
    const prev = idx > 0 ? sorted[idx - 1].le : lo;
    const left = logPos(prev, lo, hi);
    const right = logPos(b.le, lo, hi);
    const color = b.le > p99
      ? "bg-red-400/80 hover:bg-red-500"
      : b.le > p95
        ? "bg-amber-400/90 hover:bg-amber-500"
        : b.le > p50
          ? "bg-teal-400/80 hover:bg-teal-500 dark:bg-teal-500/70 dark:hover:bg-teal-400"
          : "bg-blue-400/80 hover:bg-blue-500 dark:bg-blue-500/70 dark:hover:bg-blue-400";
    const prevLe = idx > 0 ? sorted[idx - 1].le : 0;
    // 固定窄宽度居中，不用桶的自然宽度（太粗）
    const barW = 1.8;
    const center = (left + right) / 2;
    return { ...b, prevLe, left: center - barW / 2, width: barW, color };
  });

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden w-full h-full flex flex-col">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.latencyDistribution}</span>
          <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{badgeLabel}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 rounded text-[10px] text-blue-700 dark:text-blue-400 font-medium">P50 {p50}ms</span>
          <span className="px-2 py-0.5 bg-amber-100 dark:bg-amber-900/30 rounded text-[10px] text-amber-700 dark:text-amber-400 font-medium">P95 {p95}ms</span>
          <span className="px-2 py-0.5 bg-red-100 dark:bg-red-900/30 rounded text-[10px] text-red-700 dark:text-red-400 font-medium">P99 {p99}ms</span>
        </div>
      </div>

      <div className="px-4 py-3 flex-1 flex flex-col min-h-0">
        <div className="flex gap-2 flex-1 min-h-[9rem]">
          {/* Y axis */}
          <div className="flex flex-col justify-between text-[9px] text-muted text-right w-8 flex-shrink-0">
            <span>{maxCount.toLocaleString()}</span>
            <span>{Math.round(maxCount / 2).toLocaleString()}</span>
            <span>0</span>
          </div>

          {/* Chart column */}
          <div className="flex-1 flex flex-col min-w-0">
            {/* Bars area */}
            <div className="relative flex-1">
              {/* Horizontal grid */}
              <div className="absolute inset-0 pointer-events-none">
                <div className="absolute top-0 left-0 right-0 border-t border-[var(--border-color)]" />
                <div className="absolute top-1/2 left-0 right-0 border-t border-dashed border-[var(--border-color)]" />
                <div className="absolute bottom-0 left-0 right-0 border-t border-[var(--border-color)]" />
              </div>

              {/* Vertical tick grid */}
              {ticks.map(tick => (
                <div key={tick} className="absolute top-0 bottom-0 w-px bg-[var(--border-color)] opacity-30 pointer-events-none"
                  style={{ left: `${logPos(tick, lo, hi)}%` }} />
              ))}

              {/* P50/P95/P99 lines */}
              {[
                { value: p50, c: "blue", label: "P50" },
                { value: p95, c: "amber", label: "P95" },
                { value: p99, c: "red", label: "P99" },
              ].map(({ value, c, label }) => (
                <div key={label} className={`absolute top-0 bottom-0 w-px bg-${c}-500/70 z-20 pointer-events-none`}
                  style={{ left: `${logPos(value, lo, hi)}%` }}>
                  <div className={`absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-${c}-500 text-white text-[8px] font-medium whitespace-nowrap rounded`}>
                    {label}
                  </div>
                </div>
              ))}

              {/* Bars — positioned by log scale */}
              {bars.map((bar, idx) => {
                const hPct = (bar.count / maxCount) * 100;
                return (
                  <div key={idx} className="absolute bottom-0 group z-10"
                    style={{ left: `${bar.left}%`, width: `${bar.width}%`, height: "100%" }}>
                    <div className="h-full flex items-end px-px">
                      <div className={`w-full rounded-t-sm transition-all duration-150 ${bar.color}`}
                        style={{ height: `${Math.max(hPct, 2)}%` }} />
                    </div>
                    <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 hidden group-hover:block z-30 pointer-events-none">
                      <div className="bg-slate-900 text-white text-[10px] px-2.5 py-1.5 rounded-md shadow-xl whitespace-nowrap border border-slate-700">
                        <div className="font-medium">{bar.prevLe > 0 ? `${bar.prevLe}–${bar.le}ms` : `0–${bar.le}ms`}</div>
                        <div className="text-slate-300">{bar.count.toLocaleString()} {t.requests}</div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>

            {/* X axis — normal flow, not absolute */}
            <div className="relative h-5 mt-1 flex-shrink-0 text-[9px] text-muted">
              {ticks.map(tick => (
                <span key={tick} className="absolute -translate-x-1/2 whitespace-nowrap"
                  style={{ left: `${logPos(tick, lo, hi)}%` }}>
                  {tickLabel(tick)}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ==================== Method Chart ====================

const methodColors: Record<string, string> = {
  GET: "bg-blue-500",
  POST: "bg-emerald-500",
  PUT: "bg-amber-500",
  DELETE: "bg-red-500",
  OTHER: "bg-slate-500",
  PATCH: "bg-violet-500",
};

function MethodChart({ methods, totalRequests, badgeLabel, t }: {
  methods: { method: string; count: number }[];
  totalRequests: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  const maxCount = Math.max(...methods.map(m => m.count), 1);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center gap-3">
        <Layers className="w-4 h-4 text-primary" />
        <span className="text-sm font-medium text-default">{t.methodBreakdown}</span>
        <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{badgeLabel}</span>
        <span className="text-[10px] text-muted">
          {totalRequests.toLocaleString()} {t.requests}
        </span>
      </div>
      <div className="p-4 space-y-2.5">
        {methods.map((m) => {
          const percent = totalRequests > 0 ? (m.count / totalRequests) * 100 : 0;
          const barWidth = (m.count / maxCount) * 100;
          return (
            <div key={m.method} className="flex items-center gap-3">
              <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-medium w-14 text-center text-white ${methodColors[m.method] || "bg-slate-500"}`}>
                {m.method}
              </span>
              <div className="flex-1 h-5 bg-[var(--hover-bg)] rounded-sm overflow-hidden relative">
                <div
                  className={`h-full rounded-sm ${methodColors[m.method] || "bg-slate-500"} opacity-80`}
                  style={{ width: `${barWidth}%` }}
                />
              </div>
              <div className="w-28 text-right flex items-center gap-2 justify-end">
                <span className="text-xs font-medium text-default">{m.count.toLocaleString()}</span>
                <span className="text-[10px] text-muted">({percent.toFixed(0)}%)</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ==================== Status Code Chart ====================

const statusColorMap: Record<string, { bar: string; bg: string; text: string }> = {
  "2": { bar: "bg-emerald-500", bg: "bg-emerald-50 dark:bg-emerald-900/20", text: "text-emerald-700 dark:text-emerald-400" },
  "3": { bar: "bg-blue-500", bg: "bg-blue-50 dark:bg-blue-900/20", text: "text-blue-700 dark:text-blue-400" },
  "4": { bar: "bg-amber-500", bg: "bg-amber-50 dark:bg-amber-900/20", text: "text-amber-700 dark:text-amber-400" },
  "5": { bar: "bg-red-500", bg: "bg-red-50 dark:bg-red-900/20", text: "text-red-700 dark:text-red-400" },
};
const defaultStatusColor = statusColorMap["2"];

/** 支持 "200"/"2xx" 等格式，按首字符匹配 */
function getStatusColor(code: string) {
  return statusColorMap[code[0]] || defaultStatusColor;
}

function StatusCodeChart({ statusCodes, totalRequests, badgeLabel, t }: {
  statusCodes: { code: string; count: number }[];
  totalRequests: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  const sorted = [...statusCodes].sort((a, b) => a.code.localeCompare(b.code));
  const maxCount = Math.max(...sorted.map(s => s.count), 1);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center gap-3">
        <BarChart3 className="w-4 h-4 text-primary" />
        <span className="text-sm font-medium text-default">{t.statusCodeBreakdown}</span>
        <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{badgeLabel}</span>
        <span className="text-[10px] text-muted">
          {totalRequests.toLocaleString()} {t.requests}
        </span>
      </div>
      <div className="p-4 space-y-2.5">
        {sorted.map((s) => {
          const percent = totalRequests > 0 ? (s.count / totalRequests) * 100 : 0;
          const barWidth = (s.count / maxCount) * 100;
          const colors = getStatusColor(s.code);
          return (
            <div key={s.code} className="flex items-center gap-3">
              <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded font-semibold w-10 text-center ${colors.text} ${colors.bg}`}>
                {s.code}
              </span>
              <div className="flex-1 h-5 bg-[var(--hover-bg)] rounded-sm overflow-hidden relative">
                <div
                  className={`h-full rounded-sm ${colors.bar} opacity-80`}
                  style={{ width: `${barWidth}%` }}
                />
              </div>
              <div className="w-32 text-right flex items-center gap-2 justify-end">
                <span className="text-xs font-medium text-default">{percent.toFixed(1)}%</span>
                <span className="text-[10px] text-muted">{s.count.toLocaleString()}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
