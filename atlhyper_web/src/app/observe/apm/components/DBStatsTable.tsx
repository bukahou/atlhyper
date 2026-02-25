"use client";

import type { DBOperationStats } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";

interface DBStatsTableProps {
  t: ApmTranslations;
  stats: DBOperationStats[];
}

export function DBStatsTable({ t, stats }: DBStatsTableProps) {
  if (stats.length === 0) {
    return (
      <div>
        <h3 className="text-sm font-medium text-default mb-3">{t.database}</h3>
        <div className="py-8 text-center text-sm text-muted">{t.noDBCalls}</div>
      </div>
    );
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-default mb-3">{t.database}</h3>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-muted text-xs border-b border-[var(--border-color)]">
              <th className="pb-2 pr-4">{t.dbSystem}</th>
              <th className="pb-2 pr-4">{t.dbName}</th>
              <th className="pb-2 pr-4">{t.dbOperation}</th>
              <th className="pb-2 pr-4">{t.dbTable}</th>
              <th className="pb-2 pr-4 text-right">{t.dbCallCount}</th>
              <th className="pb-2 pr-4 text-right">{t.latencyAvg}</th>
              <th className="pb-2 pr-4 text-right">P99</th>
              <th className="pb-2 text-right">{t.errorRate}</th>
            </tr>
          </thead>
          <tbody>
            {stats.map((s, i) => (
              <tr key={i} className="border-b border-[var(--border-color)] last:border-0">
                <td className="py-2 pr-4">
                  <span className="inline-flex items-center gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-purple-500" />
                    <span className="text-default">{s.dbSystem}</span>
                  </span>
                </td>
                <td className="py-2 pr-4 text-default">{s.dbName || "—"}</td>
                <td className="py-2 pr-4 font-mono text-xs text-default">{s.operation || "—"}</td>
                <td className="py-2 pr-4 font-mono text-xs text-default">{s.table || "—"}</td>
                <td className="py-2 pr-4 text-right text-default">{s.callCount.toLocaleString()}</td>
                <td className="py-2 pr-4 text-right text-default">{formatDurationMs(s.avgMs)}</td>
                <td className="py-2 pr-4 text-right text-default">{formatDurationMs(s.p99Ms)}</td>
                <td className="py-2 text-right">
                  <span className={s.errorRate > 0.05 ? "text-red-500" : s.errorRate > 0.01 ? "text-orange-500" : "text-emerald-500"}>
                    {(s.errorRate * 100).toFixed(2)}%
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
