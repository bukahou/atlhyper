"use client";

import { useRef, useState } from "react";
import { BarChart3, Layers, X } from "lucide-react";
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

  return (
    <div className="space-y-4">
      <LatencyHistogram
        buckets={data.buckets}
        p50={data.p50_latency_ms}
        p95={data.p95_latency_ms}
        p99={data.p99_latency_ms}
        badgeLabel={`Ingress · ${timeRangeLabel(timeRange)}`}
        t={t}
      />
      {data.methods && data.methods.length > 0 && (
        <MethodChart
          methods={data.methods}
          totalRequests={data.total_requests}
          badgeLabel={`Ingress · ${timeRangeLabel(timeRange)}`}
          t={t}
        />
      )}
      {data.status_codes && data.status_codes.length > 0 && (
        <StatusCodeChart
          statusCodes={data.status_codes}
          totalRequests={data.total_requests}
          badgeLabel={`Ingress · ${timeRangeLabel(timeRange)}`}
          t={t}
        />
      )}
    </div>
  );
}

// ==================== Histogram ====================

// Compute P-line position aligned to bar index coordinate system
// Interpolates between bar indices based on le values
function pLinePosition(value: number, barLes: number[]): number | null {
  if (barLes.length === 0) return null;
  if (barLes.length === 1) return 50;
  if (value <= barLes[0]) return 0;
  if (value >= barLes[barLes.length - 1]) return 100;
  // Find which two bars the value falls between
  for (let i = 0; i < barLes.length - 1; i++) {
    if (value >= barLes[i] && value <= barLes[i + 1]) {
      const frac = (value - barLes[i]) / (barLes[i + 1] - barLes[i]);
      // Each bar center is at (i + 0.5) / barLes.length * 100%
      const posA = (i + 0.5) / barLes.length * 100;
      const posB = (i + 1.5) / barLes.length * 100;
      return posA + frac * (posB - posA);
    }
  }
  return null;
}

function LatencyHistogram({ buckets, p50, p95, p99, badgeLabel, t }: {
  buckets: { le: number; count: number }[];
  p50: number;
  p95: number;
  p99: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  // Only render non-zero buckets
  const activeBuckets = buckets.filter(b => b.count > 0);
  const maxCount = Math.max(...activeBuckets.map(b => b.count), 1);
  const barLes = activeBuckets.map(b => b.le);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.latencyDistribution}</span>
          <span className="px-1.5 py-0.5 rounded text-[9px] font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{badgeLabel}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 rounded text-[10px] text-blue-700 dark:text-blue-400 font-medium">
            P50 {p50}ms
          </span>
          <span className="px-2 py-0.5 bg-amber-100 dark:bg-amber-900/30 rounded text-[10px] text-amber-700 dark:text-amber-400 font-medium">
            P95 {p95}ms
          </span>
          <span className="px-2 py-0.5 bg-red-100 dark:bg-red-900/30 rounded text-[10px] text-red-700 dark:text-red-400 font-medium">
            P99 {p99}ms
          </span>
        </div>
      </div>

      <div className="px-4 pt-4 pb-6">
        <div className="flex gap-2">
          {/* Y axis */}
          <div className="flex flex-col justify-between h-36 text-[9px] text-muted text-right w-8 flex-shrink-0">
            <span>{maxCount}</span>
            <span>{Math.round(maxCount / 2)}</span>
            <span>0</span>
          </div>

          {/* Chart */}
          <div className="relative h-36 flex-1">
            {/* Grid */}
            <div className="absolute inset-0 pointer-events-none">
              <div className="absolute top-0 left-0 right-0 border-t border-[var(--border-color)]" />
              <div className="absolute top-1/2 left-0 right-0 border-t border-dashed border-[var(--border-color)]" />
              <div className="absolute bottom-0 left-0 right-0 border-t border-[var(--border-color)]" />
            </div>

            {/* P50/P95/P99 lines — aligned to bar positions */}
            {[
              { value: p50, color: "blue", label: "P50" },
              { value: p95, color: "amber", label: "P95" },
              { value: p99, color: "red", label: "P99" },
            ].map(({ value, color, label }) => {
              const pos = pLinePosition(value, barLes);
              return pos !== null ? (
                <div
                  key={label}
                  className={`absolute top-0 bottom-0 w-px bg-${color}-500/70 z-20 pointer-events-none`}
                  style={{ left: `${pos}%` }}
                >
                  <div className={`absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-${color}-500 text-white text-[8px] font-medium whitespace-nowrap rounded`}>
                    {label}
                  </div>
                </div>
              ) : null;
            })}

            {/* Bars — only active (count > 0) */}
            <div className="flex items-end h-full gap-[2px] relative z-10">
              {activeBuckets.map((bucket, idx) => {
                const heightPercent = (bucket.count / maxCount) * 100;
                const threshold = barLes.length > 1 ? (barLes[barLes.length - 1] - barLes[0]) * 0.05 : 50;
                const isNearP50 = Math.abs(bucket.le - p50) < threshold;
                const isNearP95 = Math.abs(bucket.le - p95) < threshold;
                const isNearP99 = Math.abs(bucket.le - p99) < threshold;

                return (
                  <div
                    key={idx}
                    className="flex-1 flex flex-col items-center justify-end group relative"
                    style={{ height: "100%" }}
                  >
                    <div
                      className={`w-full rounded-t-sm transition-all duration-150 ${
                        isNearP99
                          ? "bg-red-400/80 group-hover:bg-red-500"
                          : isNearP95
                            ? "bg-amber-400/90 group-hover:bg-amber-500"
                            : isNearP50
                              ? "bg-blue-400/80 group-hover:bg-blue-500"
                              : "bg-teal-400/80 group-hover:bg-teal-500 dark:bg-teal-500/70 dark:group-hover:bg-teal-400"
                      }`}
                      style={{ height: `${Math.max(heightPercent, 3)}%` }}
                    />
                    <div className="absolute bottom-full mb-2 hidden group-hover:block z-30 pointer-events-none">
                      <div className="bg-slate-900 text-white text-[10px] px-2.5 py-1.5 rounded-md shadow-xl whitespace-nowrap border border-slate-700">
                        <div className="font-medium">&le; {bucket.le}ms</div>
                        <div className="text-slate-300">{bucket.count.toLocaleString()} {t.requests}</div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>

            {/* X axis — show actual bucket le values */}
            <div className="absolute -bottom-5 left-0 right-0 flex justify-between text-[9px] text-muted">
              {activeBuckets.length <= 6
                ? activeBuckets.map((b, i) => (
                    <span key={i} className="text-center" style={{ width: `${100 / activeBuckets.length}%` }}>
                      {b.le}ms
                    </span>
                  ))
                : [0, Math.floor(activeBuckets.length / 2), activeBuckets.length - 1].map(i => {
                    const b = activeBuckets[i];
                    return (
                      <span key={i} style={{ position: "absolute", left: `${((i + 0.5) / activeBuckets.length) * 100}%`, transform: "translateX(-50%)" }}>
                        {b.le}ms
                      </span>
                    );
                  })
              }
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

const statusColors: Record<string, { bar: string; bg: string; text: string }> = {
  "2xx": { bar: "bg-emerald-500", bg: "bg-emerald-50 dark:bg-emerald-900/20", text: "text-emerald-700 dark:text-emerald-400" },
  "3xx": { bar: "bg-blue-500", bg: "bg-blue-50 dark:bg-blue-900/20", text: "text-blue-700 dark:text-blue-400" },
  "4xx": { bar: "bg-amber-500", bg: "bg-amber-50 dark:bg-amber-900/20", text: "text-amber-700 dark:text-amber-400" },
  "5xx": { bar: "bg-red-500", bg: "bg-red-50 dark:bg-red-900/20", text: "text-red-700 dark:text-red-400" },
};

function StatusCodeChart({ statusCodes, totalRequests, badgeLabel, t }: {
  statusCodes: { code: string; count: number }[];
  totalRequests: number;
  badgeLabel: string;
  t: LatencyTabTranslations;
}) {
  const maxCount = Math.max(...statusCodes.map(s => s.count), 1);

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
        {statusCodes.map((s) => {
          const percent = totalRequests > 0 ? (s.count / totalRequests) * 100 : 0;
          const barWidth = (s.count / maxCount) * 100;
          const colors = statusColors[s.code] || statusColors["2xx"];
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
