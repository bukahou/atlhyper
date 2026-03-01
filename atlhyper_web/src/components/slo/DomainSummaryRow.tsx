"use client";

import { Globe, ChevronDown, ChevronRight } from "lucide-react";
import { StatusBadge, ErrorBudgetBar, TrendIcon, formatNumber, formatLatency } from "./common";

interface DomainSummaryRowProps {
  domain: string;
  status: string;
  tls: boolean;
  serviceCount: number;
  availability: number;
  p95Latency: number;
  errorRate: number;
  rps: number;
  errorBudgetRemaining: number;
  targets: { availability: number; p95Latency: number };
  trend: "up" | "down" | "stable";
  expanded: boolean;
  onToggle: () => void;
  t: {
    services: string;
    availability: string;
    p95Latency: string;
    errorRate: string;
    errorBudget: string;
    throughput: string;
    healthy: string;
    warning: string;
    critical: string;
    unknown: string;
  };
}

export function DomainSummaryRow({
  domain, status, tls, serviceCount,
  availability, p95Latency, errorRate, rps, errorBudgetRemaining,
  targets, trend, expanded, onToggle, t,
}: DomainSummaryRowProps) {
  const statusLabels = { healthy: t.healthy, warning: t.warning, critical: t.critical, unknown: t.unknown };

  return (
    <button onClick={onToggle} className="w-full px-3 sm:px-4 py-3 flex flex-col lg:flex-row lg:items-center gap-2 lg:gap-4 hover:bg-[var(--hover-bg)] transition-colors">
      {/* Domain Info */}
      <div className="flex items-center gap-2 sm:gap-3 flex-1 min-w-0">
        <div className={`p-1.5 sm:p-2 rounded-lg flex-shrink-0 ${
          status === "healthy" ? "bg-emerald-500/10" :
          status === "warning" ? "bg-amber-500/10" : "bg-red-500/10"
        }`}>
          <Globe className={`w-4 h-4 ${
            status === "healthy" ? "text-emerald-500" :
            status === "warning" ? "text-amber-500" : "text-red-500"
          }`} />
        </div>
        <div className="text-left min-w-0 flex-1">
          <div className="flex items-center gap-1.5 sm:gap-2 flex-wrap">
            {tls && <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">HTTPS</span>}
            <span className="font-medium text-default text-sm sm:text-base truncate max-w-[150px] sm:max-w-none">{domain}</span>
            <StatusBadge status={status} labels={statusLabels} />
            <span className="text-xs text-muted hidden sm:inline">({serviceCount} {t.services})</span>
          </div>
        </div>
        <div className="flex items-center gap-2 lg:hidden">
          <TrendIcon trend={trend} />
          {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
        </div>
      </div>

      {/* Mobile Metrics */}
      <div className="flex items-center gap-3 lg:hidden ml-8 sm:ml-10">
        <div className="text-center">
          <div className={`text-sm font-semibold ${availability >= targets.availability ? "text-emerald-500" : "text-red-500"}`}>{availability.toFixed(1)}%</div>
          <div className="text-[10px] text-muted">{t.availability}</div>
        </div>
        <div className="text-center">
          <div className={`text-sm font-semibold ${errorRate <= 1 ? "text-emerald-500" : "text-red-500"}`}>{errorRate.toFixed(2)}%</div>
          <div className="text-[10px] text-muted">{t.errorRate}</div>
        </div>
        <div className="text-center">
          <div className="text-sm font-semibold text-default">{formatNumber(rps)}/s</div>
          <div className="text-[10px] text-muted">{t.throughput}</div>
        </div>
      </div>

      {/* Desktop Metrics */}
      <div className="hidden lg:flex items-center gap-5">
        <div className="w-32">
          <div className="text-[10px] text-muted mb-0.5">{t.availability}</div>
          <div className="flex items-center gap-1">
            <span className={`text-sm font-semibold ${availability >= targets.availability ? "text-emerald-500" : "text-red-500"}`}>{availability.toFixed(2)}%</span>
            <span className="text-xs text-muted">/ {targets.availability}%</span>
          </div>
        </div>
        <div className="w-32">
          <div className="text-[10px] text-muted mb-0.5">{t.p95Latency}</div>
          <div className="flex items-center gap-1">
            <span className={`text-sm font-semibold ${p95Latency <= targets.p95Latency ? "text-emerald-500" : "text-amber-500"}`}>{formatLatency(p95Latency)}</span>
            <span className="text-xs text-muted">/ {formatLatency(targets.p95Latency)}</span>
          </div>
        </div>
        <div className="w-28">
          <div className="text-[10px] text-muted mb-0.5">{t.errorRate}</div>
          <span className={`text-sm font-semibold ${errorRate <= 1 ? "text-emerald-500" : "text-red-500"}`}>{errorRate.toFixed(2)}%</span>
        </div>
        <div className="w-32">
          <div className="text-[10px] text-muted mb-0.5">{t.errorBudget}</div>
          <ErrorBudgetBar percent={errorBudgetRemaining} />
        </div>
        <div className="w-24">
          <div className="text-[10px] text-muted mb-0.5">{t.throughput}</div>
          <span className="text-sm font-semibold text-default">{formatNumber(rps)}/s</span>
        </div>
      </div>

      {/* Desktop expand */}
      <div className="hidden lg:flex items-center gap-2">
        <TrendIcon trend={trend} />
        {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
      </div>
    </button>
  );
}
