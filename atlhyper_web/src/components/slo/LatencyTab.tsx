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

export function LatencyTab({ data, t }: {
  data: LatencyDistributionResponse | null;
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
        t={t}
      />
      {data.methods && data.methods.length > 0 && (
        <MethodChart
          methods={data.methods}
          totalRequests={data.total_requests}
          t={t}
        />
      )}
      {data.status_codes && data.status_codes.length > 0 && (
        <StatusCodeChart
          statusCodes={data.status_codes}
          totalRequests={data.total_requests}
          t={t}
        />
      )}
    </div>
  );
}

// ==================== Histogram ====================

function LatencyHistogram({ buckets, p50, p95, p99, t }: {
  buckets: { le: number; count: number }[];
  p50: number;
  p95: number;
  p99: number;
  t: LatencyTabTranslations;
}) {
  const chartRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<{ start: number; end: number } | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState<number | null>(null);

  const maxCount = Math.max(...buckets.map(b => b.count), 1);
  const activeBuckets = buckets.filter(b => b.count > 0);
  const minLe = activeBuckets.length > 0 ? activeBuckets[0].le : 0;
  const maxLe = activeBuckets.length > 0 ? activeBuckets[activeBuckets.length - 1].le : 1000;
  const leRange = maxLe - minLe || 1;

  const handleMouseDown = (e: React.MouseEvent) => {
    if (!chartRef.current) return;
    const rect = chartRef.current.getBoundingClientRect();
    const x = (e.clientX - rect.left) / rect.width;
    const le = minLe + x * leRange;
    setDragStart(le);
    setIsDragging(true);
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging || dragStart === null || !chartRef.current) return;
    const rect = chartRef.current.getBoundingClientRect();
    const x = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    const le = minLe + x * leRange;
    setSelection({
      start: Math.min(dragStart, le),
      end: Math.max(dragStart, le),
    });
  };

  const handleMouseUp = () => {
    setIsDragging(false);
    setDragStart(null);
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">{t.latencyDistribution}</span>
        </div>
        <div className="flex items-center gap-3">
          {selection && (
            <button
              onClick={() => setSelection(null)}
              className="text-[10px] text-muted hover:text-default flex items-center gap-1"
            >
              <X className="w-3 h-3" />
              {t.clearSelection}
            </button>
          )}
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
          <div
            ref={chartRef}
            className="relative h-36 cursor-crosshair select-none flex-1"
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            onMouseLeave={handleMouseUp}
          >
            {/* Grid */}
            <div className="absolute inset-0 pointer-events-none">
              <div className="absolute top-0 left-0 right-0 border-t border-[var(--border-color)]" />
              <div className="absolute top-1/2 left-0 right-0 border-t border-dashed border-[var(--border-color)]" />
              <div className="absolute bottom-0 left-0 right-0 border-t border-[var(--border-color)]" />
            </div>

            {/* Selection highlight */}
            {selection && (
              <div
                className="absolute top-0 bottom-0 border-l-2 border-r-2 border-blue-500 bg-blue-500/5 pointer-events-none"
                style={{
                  left: `${((selection.start - minLe) / leRange) * 100}%`,
                  width: `${((selection.end - selection.start) / leRange) * 100}%`,
                }}
              />
            )}

            {/* P50/P95/P99 lines */}
            {[
              { value: p50, color: "blue", label: "P50" },
              { value: p95, color: "amber", label: "P95" },
              { value: p99, color: "red", label: "P99" },
            ].map(({ value, color, label }) =>
              value >= minLe && value <= maxLe ? (
                <div
                  key={label}
                  className={`absolute top-0 bottom-0 w-px bg-${color}-500/70 z-20 pointer-events-none`}
                  style={{ left: `${((value - minLe) / leRange) * 100}%` }}
                >
                  <div className={`absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-${color}-500 text-white text-[8px] font-medium whitespace-nowrap rounded`}>
                    {label}
                  </div>
                </div>
              ) : null
            )}

            {/* Bars */}
            <div className="flex items-end h-full gap-[2px] relative z-10">
              {buckets.map((bucket, idx) => {
                const heightPercent = (bucket.count / maxCount) * 100;
                const isInSelection = selection &&
                  bucket.le >= selection.start &&
                  bucket.le <= selection.end;
                const threshold = leRange * 0.05;
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
                        isInSelection
                          ? "bg-blue-500"
                          : isNearP99
                            ? "bg-red-400/80 group-hover:bg-red-500"
                            : isNearP95
                              ? "bg-amber-400/90 group-hover:bg-amber-500"
                              : isNearP50
                                ? "bg-blue-400/80 group-hover:bg-blue-500"
                                : bucket.count > 0
                                  ? "bg-teal-400/80 group-hover:bg-teal-500 dark:bg-teal-500/70 dark:group-hover:bg-teal-400"
                                  : "bg-transparent"
                      }`}
                      style={{
                        height: bucket.count > 0 ? `${Math.max(heightPercent, 3)}%` : "0",
                      }}
                    />
                    {bucket.count > 0 && (
                      <div className="absolute bottom-full mb-2 hidden group-hover:block z-30 pointer-events-none">
                        <div className="bg-slate-900 text-white text-[10px] px-2.5 py-1.5 rounded-md shadow-xl whitespace-nowrap border border-slate-700">
                          <div className="font-medium">&le; {bucket.le}ms</div>
                          <div className="text-slate-300">{bucket.count.toLocaleString()} {t.requests}</div>
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>

            {/* X axis */}
            <div className="absolute -bottom-5 left-0 right-0 flex justify-between text-[9px] text-muted">
              <span>{minLe}ms</span>
              <span>{Math.round(minLe + leRange * 0.25)}ms</span>
              <span>{Math.round(minLe + leRange * 0.5)}ms</span>
              <span>{Math.round(minLe + leRange * 0.75)}ms</span>
              <span>{maxLe}ms</span>
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

function MethodChart({ methods, totalRequests, t }: {
  methods: { method: string; count: number }[];
  totalRequests: number;
  t: LatencyTabTranslations;
}) {
  const maxCount = Math.max(...methods.map(m => m.count), 1);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center gap-3">
        <Layers className="w-4 h-4 text-primary" />
        <span className="text-sm font-medium text-default">{t.methodBreakdown}</span>
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

function StatusCodeChart({ statusCodes, totalRequests, t }: {
  statusCodes: { code: string; count: number }[];
  totalRequests: number;
  t: LatencyTabTranslations;
}) {
  const maxCount = Math.max(...statusCodes.map(s => s.count), 1);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center gap-3">
        <BarChart3 className="w-4 h-4 text-primary" />
        <span className="text-sm font-medium text-default">{t.statusCodeBreakdown}</span>
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
