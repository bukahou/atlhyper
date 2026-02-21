"use client";

import type { Dependency } from "@/api/apm";
import type { ApmTranslations } from "@/types/i18n";
import { ImpactBar } from "./ImpactBar";

interface DependenciesTableProps {
  t: ApmTranslations;
  dependencies: Dependency[];
  onSelectDependency?: (name: string) => void;
}

function formatDuration(us: number): string {
  if (us < 1000) return `${us.toFixed(0)}μs`;
  if (us < 1_000_000) return `${(us / 1000).toFixed(1)}ms`;
  return `${(us / 1_000_000).toFixed(2)}s`;
}

export function DependenciesTable({ t, dependencies, onSelectDependency }: DependenciesTableProps) {
  if (dependencies.length === 0) {
    return (
      <div>
        <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
          {t.dependencies}
        </h4>
        <div className="text-center py-6 text-muted text-xs border border-[var(--border-color)] rounded-lg">
          {t.noTraces}
        </div>
      </div>
    );
  }

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
        {t.dependencies}
      </h4>
      <div className="border border-[var(--border-color)] rounded-lg overflow-hidden">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-[var(--border-color)] bg-[var(--hover-bg)]">
              <th className="text-left px-3 py-2 font-medium text-muted">{t.serviceName}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.latencyAvg}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.throughput}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.errorRate}</th>
              <th className="text-left px-3 py-2 font-medium text-muted">{t.impact}</th>
            </tr>
          </thead>
          <tbody>
            {dependencies.map((dep) => (
              <tr
                key={dep.name}
                onClick={() => onSelectDependency?.(dep.name)}
                className="border-b border-[var(--border-color)] last:border-b-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                <td className="px-3 py-2">
                  <span className="text-primary hover:underline">{dep.name}</span>
                </td>
                <td className="px-3 py-2 text-default">{formatDuration(dep.latencyAvg)}</td>
                <td className="px-3 py-2 text-default">{dep.throughput}</td>
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
          </tbody>
        </table>
      </div>
    </div>
  );
}
