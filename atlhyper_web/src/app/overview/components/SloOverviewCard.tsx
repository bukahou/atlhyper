"use client";

import { memo } from "react";
import { ArrowRight } from "lucide-react";
import Link from "next/link";
import type { DomainSLOListResponseV2 } from "@/types/slo";
import type { useI18n } from "@/i18n/context";

interface SloOverviewCardProps {
  data: DomainSLOListResponseV2 | null;
  t: ReturnType<typeof useI18n>["t"];
}

const statusOrder: Record<string, number> = { critical: 0, warning: 1, healthy: 2, unknown: 3 };

const getStatusDotColor = (status: string) => {
  switch (status) {
    case "healthy": return "bg-green-500";
    case "warning": return "bg-yellow-500";
    case "critical": return "bg-red-500";
    default: return "bg-gray-400";
  }
};

const getValueColor = (value: number, goodThreshold: number, warnThreshold: number) => {
  if (value >= goodThreshold) return "text-green-500";
  if (value >= warnThreshold) return "text-yellow-500";
  return "text-red-500";
};

export const SloOverviewCard = memo(function SloOverviewCard({ data, t }: SloOverviewCardProps) {
  if (!data || data.domains.length === 0) {
    return (
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-4 h-[290px] flex flex-col">
        <h3 className="text-base font-semibold text-default mb-2 flex-shrink-0">{t.overview.sloOverview}</h3>
        <div className="flex-1 flex items-center justify-center text-muted text-sm">
          {t.slo.noData}
        </div>
      </div>
    );
  }

  const { summary, domains } = data;
  const sortedDomains = [...domains].sort((a, b) =>
    (statusOrder[a.status] ?? 3) - (statusOrder[b.status] ?? 3)
  );

  const total = summary.total_domains;
  const healthyPct = total > 0 ? (summary.healthy_count / total) * 100 : 0;
  const warningPct = total > 0 ? (summary.warning_count / total) * 100 : 0;
  const criticalPct = total > 0 ? (summary.critical_count / total) * 100 : 0;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4 h-[290px] flex flex-col">
      <h3 className="text-base font-semibold text-default mb-2 flex-shrink-0">{t.overview.sloOverview}</h3>

      <div className="flex-1 flex flex-col gap-2 overflow-hidden">
        {/* Summary: 3 columns */}
        <div className="grid grid-cols-3 gap-2 flex-shrink-0">
          <div className="bg-[var(--background)] rounded-lg p-2">
            <div className="text-xs text-muted mb-1">{t.slo.monitoredDomains}</div>
            <div className="text-base font-bold text-default">{total}</div>
            <div className="h-1.5 bg-[var(--card-bg)] rounded-full overflow-hidden flex mt-1">
              {healthyPct > 0 && <div className="h-full bg-green-500" style={{ width: `${healthyPct}%` }} />}
              {warningPct > 0 && <div className="h-full bg-yellow-500" style={{ width: `${warningPct}%` }} />}
              {criticalPct > 0 && <div className="h-full bg-red-500" style={{ width: `${criticalPct}%` }} />}
            </div>
          </div>

          <div className="bg-[var(--background)] rounded-lg p-2">
            <div className="text-xs text-muted mb-1">{t.slo.avgAvailability}</div>
            <div className={`text-base font-bold ${getValueColor(summary.avg_availability, 99, 95)}`}>
              {summary.avg_availability.toFixed(2)}%
            </div>
          </div>

          <div className="bg-[var(--background)] rounded-lg p-2">
            <div className="text-xs text-muted mb-1">{t.slo.errorBudget}</div>
            <div className={`text-base font-bold ${getValueColor(summary.avg_error_budget, 50, 20)}`}>
              {summary.avg_error_budget.toFixed(1)}%
            </div>
          </div>
        </div>

        {/* Domain list */}
        <div className="flex-1 overflow-y-auto space-y-1.5 min-h-0">
          {sortedDomains.map((domain) => {
            const m = domain.summary;
            const statusLabel = t.slo[domain.status as "healthy" | "warning" | "critical"] ?? t.slo.unknown;
            return (
              <div key={domain.domain} className="bg-[var(--background)] rounded-lg px-2.5 py-2">
                {/* Row 1: domain + status */}
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className={`w-2 h-2 rounded-full flex-shrink-0 ${getStatusDotColor(domain.status)}`} />
                    <span className="text-sm font-medium text-default truncate">{domain.domain}</span>
                  </div>
                  <span className={`text-xs flex-shrink-0 ${getStatusDotColor(domain.status).replace("bg-", "text-")}`}>
                    {statusLabel}
                  </span>
                </div>
                {/* Row 2: metrics with labels */}
                <div className="flex items-center gap-3 text-xs pl-4">
                  <span>
                    <span className="text-muted">{t.slo.availability} </span>
                    <span className={`font-medium ${getValueColor(m?.availability ?? 0, 99, 95)}`}>
                      {m?.availability != null ? `${m.availability.toFixed(2)}%` : "-"}
                    </span>
                  </span>
                  <span>
                    <span className="text-muted">P95 </span>
                    <span className="font-medium text-default">{m?.p95_latency ?? "-"}{t.slo.ms}</span>
                  </span>
                  <span>
                    <span className="text-muted">{t.slo.errorRate} </span>
                    <span className="font-medium text-default">{m?.error_rate != null ? `${m.error_rate.toFixed(2)}%` : "-"}</span>
                  </span>
                  <span>
                    <span className="text-muted">{t.slo.rps} </span>
                    <span className="font-medium text-default">{m?.requests_per_sec != null ? m.requests_per_sec.toFixed(1) : "-"}</span>
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Link to SLO detail page */}
      <Link
        href="/workbench/slo"
        className="flex items-center justify-center gap-1 text-xs text-blue-500 hover:text-blue-400 mt-2 flex-shrink-0"
      >
        {t.overview.viewSloDetail} <ArrowRight className="w-3 h-3" />
      </Link>
    </div>
  );
});
