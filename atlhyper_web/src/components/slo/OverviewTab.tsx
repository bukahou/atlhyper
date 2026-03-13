"use client";

import { useMemo } from "react";
import { ErrorBudgetBar, formatNumber } from "./common";
import { HistoryChart } from "./HistoryChart";
import { ErrorBudgetBurnChart } from "./ErrorBudgetBurnChart";
import type { SLOMetrics } from "@/types/slo";

interface OverviewTabTranslations {
  availability: string;
  p95Latency: string;
  p99Latency: string;
  errorRate: string;
  totalRequests: string;
  errorBudget: string;
  target: string;
  throughput: string;
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  estimatedExhaust: string;
  noData: string;
}

/** Main Overview Tab */
export function OverviewTab({ summary, errorBudgetRemaining, targets, history, t }: {
  summary: SLOMetrics | null;
  errorBudgetRemaining: number;
  targets?: { availability: number; p95Latency: number };
  history?: { timestamp: string; p95Latency: number; p99Latency: number; errorRate: number; availability: number; rps: number; errorBudget: number }[];
  t: OverviewTabTranslations;
}) {
  const availability = summary?.availability ?? 0;
  const p95Latency = summary?.p95Latency ?? 0;
  const p99Latency = summary?.p99Latency ?? 0;
  const errorRate = summary?.errorRate ?? 0;
  const rps = summary?.requestsPerSec ?? 0;
  const totalRequests = summary?.totalRequests ?? 0;

  const budgetHistory = useMemo(() => {
    if (!history || history.length === 0) return [];
    return history.map(h => ({
      timestamp: h.timestamp,
      errorBudget: h.errorBudget,
    }));
  }, [history]);

  return (
    <div className="space-y-4">
      {/* Golden Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.availability}</div>
          <div className="text-lg font-bold text-default">{availability.toFixed(3)}%</div>
          {targets && <div className="text-xs text-muted mt-1">{t.target}: {targets.availability}%</div>}
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.p95Latency} / {t.p99Latency}</div>
          <div className="text-lg font-bold text-default">{p95Latency}ms / {p99Latency}ms</div>
          {targets && <div className="text-xs text-muted mt-1">{t.target} P95: {targets.p95Latency}ms</div>}
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.errorRate}</div>
          <div className="text-lg font-bold text-default">{errorRate.toFixed(3)}%</div>
          <div className="text-xs text-muted mt-1">
            {Math.round(totalRequests * errorRate / 100)} / {formatNumber(totalRequests)}
            {targets && <span> · {t.target}: {(100 - targets.availability).toFixed(1)}%</span>}
          </div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.totalRequests}</div>
          <div className="text-lg font-bold text-default">{formatNumber(totalRequests)}</div>
          <div className="text-xs text-muted mt-1">{formatNumber(rps)} req/s</div>
        </div>
        <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
          <div className="text-xs text-muted mb-1">{t.errorBudget}</div>
          <div className={`text-lg font-bold ${
            errorBudgetRemaining > 50 ? "text-emerald-500" :
            errorBudgetRemaining > 20 ? "text-amber-500" : "text-red-500"
          }`}>{errorBudgetRemaining.toFixed(1)}%</div>
          <div className="mt-1"><ErrorBudgetBar percent={errorBudgetRemaining} /></div>
        </div>
      </div>

      {/* Charts -- always visible */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <HistoryChart
          history={history || []}
          targets={targets}
          t={{
            p95Latency: t.p95Latency,
            errorRate: t.errorRate,
            target: t.target,
            sloTrend: t.sloTrend,
            noData: t.noData,
          }}
        />
        <ErrorBudgetBurnChart
          history={budgetHistory}
          errorBudgetRemaining={errorBudgetRemaining}
          t={{
            errorBudgetBurn: t.errorBudgetBurn,
            current: t.current,
            estimatedExhaust: t.estimatedExhaust,
            noData: t.noData,
          }}
        />
      </div>
    </div>
  );
}
