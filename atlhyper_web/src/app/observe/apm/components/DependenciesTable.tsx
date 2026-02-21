"use client";

import type { Dependency } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { ImpactBar } from "./ImpactBar";

interface DependenciesTableProps {
  t: ApmTranslations;
  dependencies: Dependency[];
  onSelectDependency?: (name: string) => void;
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
                  <div className="flex items-center gap-1.5">
                    <span className={`inline-block w-2 h-2 rounded-full ${
                      dep.type === "database" ? "bg-amber-500" :
                      dep.type === "external" ? "bg-purple-500" : "bg-blue-500"
                    }`} />
                    <span className="text-primary hover:underline">{dep.name}</span>
                    <span className="text-[10px] text-muted px-1 py-0.5 rounded bg-[var(--hover-bg)]">{dep.type}</span>
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
          </tbody>
        </table>
      </div>
    </div>
  );
}
