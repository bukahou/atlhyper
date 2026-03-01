"use client";

import type { MeshTabTranslations } from "./MeshTypes";

// Histogram utilities (Kibana-style fixed axis, log scale)
// 1-2-5 log-scale tick series
const MESH_TICKS = [1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000];

function meshLogPos(ms: number, lo: number, hi: number): number {
  const v = Math.log10(Math.max(ms, 0.1));
  const a = Math.log10(Math.max(lo, 0.1));
  const b = Math.log10(Math.max(hi, 1));
  if (b <= a) return 50;
  return Math.min(100, Math.max(0, ((v - a) / (b - a)) * 100));
}

function meshAxisRange(les: number[]): [number, number] {
  if (les.length === 0) return [1, 1000];
  const minLe = Math.min(...les), maxLe = Math.max(...les);
  let lo = MESH_TICKS[0];
  for (const t of MESH_TICKS) { if (t <= minLe * 0.6) lo = t; else break; }
  let hi = MESH_TICKS[MESH_TICKS.length - 1];
  for (let i = MESH_TICKS.length - 1; i >= 0; i--) { if (MESH_TICKS[i] >= maxLe * 1.4) hi = MESH_TICKS[i]; else break; }
  return [Math.min(lo, minLe * 0.5), Math.max(hi, maxLe * 1.5)];
}

function meshVisibleTicks(lo: number, hi: number): number[] {
  return MESH_TICKS.filter(t => t >= lo && t <= hi);
}

function meshTickLabel(ms: number): string { return ms >= 1000 ? `${ms / 1000}s` : `${ms}ms`; }

// Mini Latency Histogram for service detail (Kibana-style)
export function MiniLatencyHistogram({ buckets, allBuckets, p50, p95, p99, t }: {
  buckets: { le: number; count: number }[];
  allBuckets: { le: number; count: number }[];
  p50: number;
  p95: number;
  p99: number;
  t: MeshTabTranslations;
}) {
  if (buckets.length === 0) return null;
  const maxCount = Math.max(...buckets.map(b => b.count), 1);
  const [lo, hi] = meshAxisRange(buckets.map(b => b.le));
  const ticks = meshVisibleTicks(lo, hi);

  const bars = buckets.map((b) => {
    const idx = allBuckets.indexOf(b);
    const prev = idx > 0 ? allBuckets[idx - 1].le : lo;
    const left = meshLogPos(prev, lo, hi);
    const right = meshLogPos(b.le, lo, hi);
    const color = b.le > p99
      ? "bg-red-400/80 hover:bg-red-500"
      : b.le > p95 ? "bg-amber-400/90 hover:bg-amber-500"
      : b.le > p50 ? "bg-teal-400/80 hover:bg-teal-500 dark:bg-teal-500/70 dark:hover:bg-teal-400"
      : "bg-blue-400/80 hover:bg-blue-500 dark:bg-blue-500/70 dark:hover:bg-blue-400";
    const prevLe = idx > 0 ? allBuckets[idx - 1].le : 0;
    const rawWidth = right - left;
    const cappedWidth = Math.min(Math.max(rawWidth, 0.5), 5);
    const center = (left + right) / 2;
    const barLeft = center - cappedWidth / 2;
    return { ...b, prevLe, left: barLeft, width: cappedWidth, color };
  });

  return (
    <div className="flex gap-2">
      <div className="flex flex-col justify-between h-28 text-[9px] text-muted text-right w-8 flex-shrink-0">
        <span>{maxCount.toLocaleString()}</span>
        <span>{Math.round(maxCount / 2).toLocaleString()}</span>
        <span>0</span>
      </div>
      <div className="relative h-28 flex-1">
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-0 left-0 right-0 border-t border-[var(--border-color)]" />
          <div className="absolute top-1/2 left-0 right-0 border-t border-dashed border-[var(--border-color)]" />
          <div className="absolute bottom-0 left-0 right-0 border-t border-[var(--border-color)]" />
        </div>
        {/* Vertical tick grid */}
        {ticks.map(tick => (
          <div key={tick} className="absolute top-0 bottom-0 w-px bg-[var(--border-color)] opacity-30 pointer-events-none"
            style={{ left: `${meshLogPos(tick, lo, hi)}%` }} />
        ))}
        {/* P50/P95/P99 lines */}
        {[
          { value: p50, c: "blue", label: "P50" },
          { value: p95, c: "amber", label: "P95" },
          { value: p99, c: "red", label: "P99" },
        ].map(({ value, c, label }) => (
          <div key={label} className={`absolute top-0 bottom-0 w-px bg-${c}-500/70 z-20 pointer-events-none`}
            style={{ left: `${meshLogPos(value, lo, hi)}%` }}>
            <div className={`absolute -top-1 left-1/2 -translate-x-1/2 px-1 py-0.5 bg-${c}-500 text-white text-[8px] font-medium whitespace-nowrap rounded`}>
              {label}
            </div>
          </div>
        ))}
        {/* Bars — log scale positioning */}
        {bars.map((bar, idx) => {
          const hPct = (bar.count / maxCount) * 100;
          return (
            <div key={idx} className="absolute bottom-0 group z-10"
              style={{ left: `${bar.left}%`, width: `${bar.width}%`, height: "100%" }}>
              <div className="h-full flex items-end px-[0.5px]">
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
        {/* X axis — fixed tick labels */}
        <div className="absolute -bottom-4 left-0 right-0 text-[9px] text-muted">
          {ticks.map(tick => (
            <span key={tick} className="absolute -translate-x-1/2 whitespace-nowrap"
              style={{ left: `${meshLogPos(tick, lo, hi)}%` }}>
              {meshTickLabel(tick)}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}
