"use client";

import { useMemo } from "react";
import { Activity, AlertTriangle, Globe, Zap, Gauge, Server } from "lucide-react";
import type { SLOTranslations } from "@/types/i18n";
import type { DomainSLOListResponseV2 } from "@/types/slo";
import { SummaryCard, formatNumber } from "./common";
import { SLOListTable } from "./SLOListTable";
import { buildDomainDetailTranslations } from "./detail-translations";

interface SLODomainsViewProps {
  domainList: DomainSLOListResponseV2;
  /** SLO 预聚合窗口（1h / 6h / 24h / 3d / 7d） */
  timeRange: string;
  clusterId: string;
  expandedDomain: string | null;
  onToggleDomain: (domain: string) => void;
  sloT: SLOTranslations;
}

/** SLO 页的主体：汇总卡片 + 域名清单（或空状态）。只在清单取到之后渲染 */
export function SLODomainsView({
  domainList,
  timeRange,
  clusterId,
  expandedDomain,
  onToggleDomain,
  sloT,
}: SLODomainsViewProps) {
  const domains = useMemo(() => domainList.domains ?? [], [domainList.domains]);
  const summary = domainList.summary ?? null;

  const summaryData = useMemo(() => {
    const avgP95 = domains.length > 0
      ? domains.reduce((sum, d) => sum + (d.summary?.p95Latency || 0), 0) / domains.length
      : 0;
    if (summary) {
      return {
        totalServices: summary.totalServices || 0,
        totalDomains: summary.totalDomains,
        healthyCount: summary.healthyCount,
        warningCount: summary.warningCount,
        criticalCount: summary.criticalCount,
        totalRPS: summary.totalRps,
        avgAvailability: summary.avgAvailability,
        avgP95,
      };
    }
    const totalDomains = domains.length;
    const healthyCount = domains.filter(d => d.status === "healthy").length;
    const warningCount = domains.filter(d => d.status === "warning").length;
    const criticalCount = domains.filter(d => d.status === "critical").length;
    const totalRPS = domains.reduce((sum, d) => sum + (d.summary?.requestsPerSec || 0), 0);
    const avgAvailability = totalDomains > 0 ? domains.reduce((sum, d) => sum + (d.summary?.availability || 0), 0) / totalDomains : 0;
    const totalServices = domains.reduce((sum, d) => sum + d.services.length, 0);
    return { totalServices, totalDomains, healthyCount, warningCount, criticalCount, totalRPS, avgAvailability, avgP95 };
  }, [domains, summary]);

  const domainDetailT = useMemo(() => buildDomainDetailTranslations(sloT), [sloT]);
  const listTableT = useMemo(
    () => ({
      domain: sloT.monitoredDomains,
      viewTraces: sloT.viewTraces,
      viewErrorLogs: sloT.viewErrorLogs,
      target: sloT.target,
      currentSli: sloT.currentSli,
      errorBudget: sloT.errorBudget,
      burnRate: sloT.burnRate,
      p95Latency: sloT.p95Latency,
      status: sloT.sloStatus,
      healthy: sloT.healthy,
      warning: sloT.warning,
      critical: sloT.critical,
      unknown: sloT.unknown,
      noData: sloT.noData,
      goodBad: sloT.goodBad,
      exhaustIn: sloT.exhaustIn,
      hours: sloT.hours,
    }),
    [sloT],
  );

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Summary Cards — 始终显示 */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 sm:gap-4">
        <SummaryCard icon={Server} label={sloT.totalServices} value={summaryData.totalServices.toString()} subValue={sloT.ingressBackends} color="bg-blue-500/10 text-blue-500" />
        <SummaryCard icon={Globe} label={sloT.monitoredDomains} value={summaryData.totalDomains.toString()} subValue={`${summaryData.healthyCount} ${sloT.healthy}`} color="bg-violet-500/10 text-violet-500" />
        <SummaryCard icon={Activity} label={sloT.avgAvailability} value={`${summaryData.avgAvailability.toFixed(2)}%`} color="bg-emerald-500/10 text-emerald-500" />
        <SummaryCard icon={Gauge} label={sloT.avgP95} value={`${Math.round(summaryData.avgP95)}ms`} color="bg-cyan-500/10 text-cyan-500" />
        <SummaryCard icon={Zap} label={sloT.totalRPS} value={formatNumber(summaryData.totalRPS)} subValue={sloT.reqPerSec} color="bg-amber-500/10 text-amber-500" />
        <SummaryCard icon={AlertTriangle} label={sloT.alertCount} value={(summaryData.warningCount + summaryData.criticalCount).toString()}
          subValue={`${summaryData.criticalCount} ${sloT.severe}`}
          color={summaryData.criticalCount > 0 ? "bg-red-500/10 text-red-500" : "bg-amber-500/10 text-amber-500"} />
      </div>

      {domains.length === 0 ? (
        /* Empty — 无域名数据 */
        <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
          <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
          <p className="text-default font-medium mb-2">{sloT.noData}</p>
          <p className="text-sm text-muted">{sloT.noDataHint}</p>
        </div>
      ) : (
        <>
          {/* SLO 清单表 —— 一行一个域名，一眼看出谁在烧预算 */}
          <div>
            <div className="flex items-baseline justify-between mb-2">
              <h2 className="text-sm font-semibold text-default">
                {sloT.sloListTitle}
                <span className="ml-2 text-xs font-normal text-muted">({summaryData.totalDomains})</span>
              </h2>
              <p className="text-[11px] text-muted hidden lg:block">{sloT.sloListHint}</p>
            </div>
            <SLOListTable
              domains={domains}
              expandedId={expandedDomain}
              onSelect={onToggleDomain}
              timeRange={timeRange}
              clusterId={clusterId}
              detailT={domainDetailT}
              t={listTableT}
            />
          </div>

          {/* Data Source Note */}
          <div className="p-4 rounded-xl bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
            <div className="flex items-start gap-3">
              <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-900/50">
                <Activity className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="text-sm">
                <p className="font-medium text-blue-800 dark:text-blue-200 mb-1">{sloT.dataSourceTitle}</p>
                <p className="text-blue-700 dark:text-blue-300 text-xs leading-relaxed">{sloT.dataSourceDesc}</p>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
