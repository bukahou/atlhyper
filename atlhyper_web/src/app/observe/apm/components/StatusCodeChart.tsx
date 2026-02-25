"use client";

import { useMemo } from "react";
import type { HTTPStats } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";

const STATUS_COLORS: Record<string, { bg: string; text: string; bar: string }> = {
  "2xx": { bg: "bg-emerald-500/15", text: "text-emerald-500", bar: "bg-emerald-500" },
  "3xx": { bg: "bg-blue-500/15", text: "text-blue-500", bar: "bg-blue-500" },
  "4xx": { bg: "bg-orange-500/15", text: "text-orange-500", bar: "bg-orange-500" },
  "5xx": { bg: "bg-red-500/15", text: "text-red-500", bar: "bg-red-500" },
};

function getStatusGroup(code: number): string {
  return `${Math.floor(code / 100)}xx`;
}

function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

interface StatusCodeChartProps {
  t: ApmTranslations;
  stats: HTTPStats[];
}

export function StatusCodeChart({ t, stats }: StatusCodeChartProps) {
  const rows = useMemo(() => {
    // Aggregate by method+statusCode
    const map = new Map<string, { method: string; code: number; count: number }>();
    for (const s of stats) {
      const key = `${s.method} ${s.statusCode}`;
      const existing = map.get(key);
      if (existing) {
        existing.count += s.count;
      } else {
        map.set(key, { method: s.method, code: s.statusCode, count: s.count });
      }
    }
    const items = [...map.values()].sort((a, b) => b.count - a.count);
    const total = items.reduce((s, i) => s + i.count, 0);
    const maxCount = items.length > 0 ? items[0].count : 1;
    return items.map((item) => ({
      ...item,
      group: getStatusGroup(item.code),
      pct: total > 0 ? (item.count / total) * 100 : 0,
      // Use log scale for bar width so small values are still visible
      barWidth: Math.max(2, (Math.log10(item.count + 1) / Math.log10(maxCount + 1)) * 100),
    }));
  }, [stats]);

  if (rows.length === 0) {
    return (
      <div>
        <h3 className="text-sm font-medium text-default mb-3">{t.httpStatusDistribution}</h3>
        <div className="py-8 text-center text-sm text-muted">{t.noData}</div>
      </div>
    );
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-default mb-3">{t.httpStatusDistribution}</h3>
      <div className="space-y-1.5">
        {rows.map((row) => {
          const colors = STATUS_COLORS[row.group] || STATUS_COLORS["2xx"];
          return (
            <div key={`${row.method}-${row.code}`} className="flex items-center gap-3 h-8">
              {/* Status badge */}
              <div className="w-20 shrink-0 flex items-center gap-1.5">
                <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold ${colors.bg} ${colors.text}`}>
                  {row.code}
                </span>
                <span className="text-xs text-muted">{row.method}</span>
              </div>

              {/* Bar */}
              <div className="flex-1 h-5 rounded bg-[var(--border-color)]/30 relative overflow-hidden">
                <div
                  className={`h-full rounded ${colors.bar} transition-all duration-300`}
                  style={{ width: `${row.barWidth}%`, opacity: 0.7 }}
                />
              </div>

              {/* Count + Percentage */}
              <div className="w-28 shrink-0 text-right flex items-center justify-end gap-2">
                <span className="text-sm font-medium text-default tabular-nums">{formatCount(row.count)}</span>
                <span className="text-xs text-muted tabular-nums w-12 text-right">{row.pct.toFixed(1)}%</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
