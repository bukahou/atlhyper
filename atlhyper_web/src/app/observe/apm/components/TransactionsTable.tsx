"use client";

import { useState, useMemo, useCallback } from "react";
import { Search, ChevronLeft, ChevronRight, ArrowUp, ArrowDown } from "lucide-react";
import type { OperationStats } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { ImpactBar } from "./ImpactBar";

const PAGE_SIZE = 10;

type SortKey = "name" | "latency" | "throughput" | "errorRate" | "impact";
type SortDir = "asc" | "desc";

interface TransactionsTableProps {
  t: ApmTranslations;
  operations: OperationStats[];
  onSelectOperation?: (operation: string) => void;
}

type OpWithImpact = OperationStats & { impact: number };

function getSortValue(op: OpWithImpact, key: SortKey): number | string {
  switch (key) {
    case "name": return op.operationName.toLowerCase();
    case "latency": return op.avgDurationMs;
    case "throughput": return op.rps;
    case "errorRate": return 1 - op.successRate;
    case "impact": return op.impact;
  }
}

export function TransactionsTable({ t, operations, onSelectOperation }: TransactionsTableProps) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);
  const [sortKey, setSortKey] = useState<SortKey>("impact");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const handleSort = useCallback((key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "desc" ? "asc" : "desc"));
    } else {
      setSortKey(key);
      setSortDir(key === "name" ? "asc" : "desc");
    }
    setPage(0);
  }, [sortKey]);

  const opsWithImpact = useMemo(() => {
    const totals = operations.map((op) => op.avgDurationMs * op.spanCount);
    const maxTotal = Math.max(...totals, 1);
    return operations.map((op, i) => ({
      ...op,
      impact: totals[i] / maxTotal,
    }));
  }, [operations]);

  const filtered = useMemo(() => {
    let result = opsWithImpact;
    if (search) {
      const q = search.toLowerCase();
      result = result.filter((o) => o.operationName.toLowerCase().includes(q));
    }
    // Sort
    const sorted = [...result].sort((a, b) => {
      const va = getSortValue(a, sortKey);
      const vb = getSortValue(b, sortKey);
      if (typeof va === "string" && typeof vb === "string") {
        return sortDir === "asc" ? va.localeCompare(vb) : vb.localeCompare(va);
      }
      const na = va as number, nb = vb as number;
      return sortDir === "asc" ? na - nb : nb - na;
    });
    return sorted;
  }, [opsWithImpact, search, sortKey, sortDir]);

  const handleSearch = (value: string) => {
    setSearch(value);
    setPage(0);
  };

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const safeP = Math.min(page, totalPages - 1);
  const paged = filtered.slice(safeP * PAGE_SIZE, (safeP + 1) * PAGE_SIZE);
  const emptyRows = PAGE_SIZE - paged.length;

  const columns: { key: SortKey; label: string; width: string }[] = [
    { key: "name", label: t.operationName, width: "" },
    { key: "latency", label: t.latencyAvg, width: "w-20" },
    { key: "throughput", label: t.throughput, width: "w-24" },
    { key: "errorRate", label: t.errorRate, width: "w-16" },
    { key: "impact", label: t.impact, width: "w-24" },
  ];

  return (
    <div className="flex flex-col">
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
          onChange={(e) => handleSearch(e.target.value)}
          placeholder={t.searchTransactions}
          className="w-full pl-8 pr-3 py-1.5 text-xs rounded-md border border-[var(--border-color)] bg-card text-default placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary/30"
        />
      </div>
      <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
        <table className="w-full text-xs table-fixed">
          <thead>
            <tr className="border-b border-[var(--border-color)] bg-[var(--hover-bg)]">
              {columns.map((col) => (
                <th
                  key={col.key}
                  onClick={() => handleSort(col.key)}
                  className={`text-left px-3 py-2 font-medium text-muted select-none cursor-pointer hover:text-default transition-colors ${col.width}`}
                >
                  <span className="inline-flex items-center gap-1">
                    {col.label}
                    {sortKey === col.key && (
                      sortDir === "desc"
                        ? <ArrowDown className="w-3 h-3 text-primary" />
                        : <ArrowUp className="w-3 h-3 text-primary" />
                    )}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {paged.map((op) => (
              <tr
                key={op.operationName}
                onClick={() => onSelectOperation?.(op.operationName)}
                className="border-b border-[var(--border-color)] last:border-b-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                <td className="px-3 py-2 truncate">
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
            {emptyRows > 0 && Array.from({ length: emptyRows }).map((_, i) => (
              <tr key={`empty-${i}`} className="border-b border-[var(--border-color)] last:border-b-0" aria-hidden>
                <td className="px-3 py-2">&nbsp;</td>
                <td className="px-3 py-2" />
                <td className="px-3 py-2" />
                <td className="px-3 py-2" />
                <td className="px-3 py-2" />
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex items-center justify-between mt-2 text-xs text-muted h-6">
        <span>{filtered.length} {t.transactions}</span>
        {totalPages > 1 && (
          <div className="flex items-center gap-1">
            <button
              onClick={() => setPage(Math.max(0, safeP - 1))}
              disabled={safeP === 0}
              className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft className="w-3.5 h-3.5" />
            </button>
            <span className="px-1.5 tabular-nums">{safeP + 1} / {totalPages}</span>
            <button
              onClick={() => setPage(Math.min(totalPages - 1, safeP + 1))}
              disabled={safeP >= totalPages - 1}
              className="p-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronRight className="w-3.5 h-3.5" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
