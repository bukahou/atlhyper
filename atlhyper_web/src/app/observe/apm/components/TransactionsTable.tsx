"use client";

import { useState, useMemo } from "react";
import { Search } from "lucide-react";
import type { TraceSummary } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { MiniSparkline } from "./MiniSparkline";
import { ImpactBar } from "./ImpactBar";

interface TransactionsTableProps {
  t: ApmTranslations;
  traces: TraceSummary[];
  onSelectOperation?: (operation: string) => void;
}

interface OpStats {
  name: string;
  avgMs: number;
  throughput: number;
  errorRate: number;
  impact: number;
  latencyPoints: number[];
}

export function TransactionsTable({ t, traces, onSelectOperation }: TransactionsTableProps) {
  const [search, setSearch] = useState("");

  const opStats = useMemo(() => {
    const map = new Map<string, { durations: number[]; errorCount: number }>();
    for (const tr of traces) {
      const entry = map.get(tr.rootOperation) ?? { durations: [], errorCount: 0 };
      entry.durations.push(tr.durationMs);
      if (tr.hasError) entry.errorCount++;
      map.set(tr.rootOperation, entry);
    }

    const allTotals = Array.from(map.values()).map((d) => d.durations.reduce((a, b) => a + b, 0));
    const maxTotal = Math.max(...allTotals, 1);

    const result: OpStats[] = [];
    for (const [name, data] of map) {
      const count = data.durations.length;
      const total = data.durations.reduce((a, b) => a + b, 0);
      result.push({
        name,
        avgMs: total / count,
        throughput: count,
        errorRate: data.errorCount / count,
        impact: total / maxTotal,
        latencyPoints: data.durations,
      });
    }
    return result.sort((a, b) => b.impact - a.impact);
  }, [traces]);

  const filtered = useMemo(() => {
    if (!search) return opStats;
    const q = search.toLowerCase();
    return opStats.filter((o) => o.name.toLowerCase().includes(q));
  }, [opStats, search]);

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h4 className="text-xs font-semibold text-muted uppercase tracking-wider">
          {t.transactions}
        </h4>
      </div>
      <div className="relative mb-2">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t.searchTransactions}
          className="w-full pl-8 pr-3 py-1.5 text-xs rounded-md border border-[var(--border-color)] bg-card text-default placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary/30"
        />
      </div>
      <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-[var(--border-color)] bg-[var(--hover-bg)]">
              <th className="text-left px-3 py-2 font-medium text-muted">{t.operationName}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.latencyAvg}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.throughput}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.errorRate}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.impact}</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((op) => (
              <tr
                key={op.name}
                onClick={() => onSelectOperation?.(op.name)}
                className="border-b border-[var(--border-color)] last:border-b-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                <td className="px-3 py-2">
                  <span className="text-primary hover:underline">{op.name}</span>
                </td>
                <td className="px-3 py-2">
                  <div className="flex items-center gap-1.5">
                    <span className="text-default">{formatDurationMs(op.avgMs)}</span>
                    <MiniSparkline data={op.latencyPoints} type="line" color="#6366f1" width={48} height={16} />
                  </div>
                </td>
                <td className="px-3 py-2">
                  <span className="text-default">{op.throughput} {t.tpm}</span>
                </td>
                <td className="px-3 py-2">
                  <span className={op.errorRate > 0 ? "text-orange-500" : "text-default"}>
                    {(op.errorRate * 100).toFixed(1)}%
                  </span>
                </td>
                <td className="px-3 py-2">
                  <ImpactBar value={op.impact} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
