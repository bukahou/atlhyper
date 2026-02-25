"use client";

import { useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { Dependency } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { ImpactBar } from "./ImpactBar";

const PAGE_SIZE = 5;

interface DependenciesTableProps {
  t: ApmTranslations;
  dependencies: Dependency[];
  onSelectDependency?: (name: string) => void;
}

export function DependenciesTable({ t, dependencies, onSelectDependency }: DependenciesTableProps) {
  const [page, setPage] = useState(0);

  const totalPages = Math.max(1, Math.ceil(dependencies.length / PAGE_SIZE));
  const safeP = Math.min(page, totalPages - 1);
  const paged = dependencies.slice(safeP * PAGE_SIZE, (safeP + 1) * PAGE_SIZE);
  const emptyRows = PAGE_SIZE - paged.length;

  return (
    <div className="flex flex-col">
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
        {t.dependencies}
      </h4>
      <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
        <table className="w-full text-xs table-fixed">
          <thead>
            <tr className="border-b border-[var(--border-color)] bg-[var(--hover-bg)]">
              <th className="text-left px-3 py-2 font-medium text-muted">{t.serviceName}</th>
              <th className="text-left px-3 py-2 font-medium text-muted w-20">{t.latencyAvg}</th>
              <th className="text-left px-3 py-2 font-medium text-muted w-20">{t.throughput}</th>
              <th className="text-left px-3 py-2 font-medium text-muted w-16">{t.errorRate}</th>
              <th className="text-left px-3 py-2 font-medium text-muted w-24">{t.impact}</th>
            </tr>
          </thead>
          <tbody>
            {paged.map((dep) => (
              <tr
                key={dep.name}
                onClick={() => onSelectDependency?.(dep.name)}
                className="border-b border-[var(--border-color)] last:border-b-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                <td className="px-3 py-2 truncate">
                  <div className="flex items-center gap-1.5">
                    <span className={`inline-block w-2 h-2 rounded-full shrink-0 ${
                      dep.type === "database" ? "bg-amber-500" :
                      dep.type === "external" ? "bg-purple-500" : "bg-blue-500"
                    }`} />
                    <span className="text-primary hover:underline truncate">{dep.name}</span>
                    <span className="text-[10px] text-muted px-1 py-0.5 rounded bg-[var(--hover-bg)] shrink-0">{dep.type}</span>
                  </div>
                </td>
                <td className="px-3 py-2 text-default">{formatDurationMs(dep.avgMs)}</td>
                <td className="px-3 py-2 text-default">{dep.callCount}</td>
                <td className="px-3 py-2">
                  <span className={dep.errorRate > 0 ? "text-orange-500" : "text-default"}>
                    {(dep.errorRate * 100).toFixed(1)}%
                  </span>
                </td>
                <td className="px-3 py-2">
                  <ImpactBar value={dep.impact} />
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
        <span>{dependencies.length} {t.dependencies}</span>
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
