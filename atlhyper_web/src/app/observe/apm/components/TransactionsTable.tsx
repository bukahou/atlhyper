"use client";

import { useState, useMemo } from "react";
import { Search } from "lucide-react";
import type { OperationStats } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { ImpactBar } from "./ImpactBar";

interface TransactionsTableProps {
  t: ApmTranslations;
  operations: OperationStats[];
  onSelectOperation?: (operation: string) => void;
}

export function TransactionsTable({ t, operations, onSelectOperation }: TransactionsTableProps) {
  const [search, setSearch] = useState("");

  // 计算 impact（基于总耗时占比）
  const opsWithImpact = useMemo(() => {
    const totals = operations.map((op) => op.avgDurationMs * op.spanCount);
    const maxTotal = Math.max(...totals, 1);
    return operations.map((op, i) => ({
      ...op,
      impact: totals[i] / maxTotal,
    }));
  }, [operations]);

  const filtered = useMemo(() => {
    if (!search) return opsWithImpact;
    const q = search.toLowerCase();
    return opsWithImpact.filter((o) => o.operationName.toLowerCase().includes(q));
  }, [opsWithImpact, search]);

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
                key={op.operationName}
                onClick={() => onSelectOperation?.(op.operationName)}
                className="border-b border-[var(--border-color)] last:border-b-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                <td className="px-3 py-2">
                  <span className="text-primary hover:underline">{op.operationName}</span>
                </td>
                <td className="px-3 py-2">
                  <span className="text-default">{formatDurationMs(op.avgDurationMs)}</span>
                </td>
                <td className="px-3 py-2">
                  <span className="text-default">{op.rps.toFixed(1)} {t.tpm}</span>
                </td>
                <td className="px-3 py-2">
                  <span className={(1 - op.successRate) > 0 ? "text-orange-500" : "text-default"}>
                    {((1 - op.successRate) * 100).toFixed(1)}%
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
