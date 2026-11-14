"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, BarChart3, Settings2 } from "lucide-react";
import { QueryView } from "@/components/common";
import { HistoryChart } from "./HistoryChart";
import { ErrorBudgetBurnChart } from "./ErrorBudgetBurnChart";
import { BurnRateTable } from "./BurnRateTable";
import { GoodBadCount } from "./GoodBadCount";
import { LatencyTab } from "./LatencyTab";
import { SLOTargetModal } from "./SLOTargetModal";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { sloDomainHistoryQuery, sloLatencyDistributionQuery } from "@/queries/observe-slo";
import type { DomainSLOV2 } from "@/types/slo";
import type { DomainDetailTranslations } from "./detail-translations";

interface DomainTabProps {
  domain: DomainSLOV2;
  timeRange: string;
  clusterId: string;
  t: DomainDetailTranslations;
}

/**
 * 域名 SLO 详情 —— 清单表行内展开的内容
 *
 * 只放清单表里没有的东西。域名、目标、当前 SLI、预算余量、燃烧率概要、P95、状态
 * 都在清单表那一行上了，这里再列一遍是三份重复（改版前正是如此）。
 *
 * 两个 tab 按「排查动作」而非「数据类型」划分：
 *   预算 —— 还能烧多久、烧得多快、错了多少个（SLO 的本职）
 *   延迟 —— 慢在哪里（分位数、方法、状态码）
 *
 * 每个 tab 是一个子组件、各带自己的查询：只有切到那个 tab 才会请求它的数据。
 */
export function DomainDetail({ domain, timeRange, clusterId, t }: DomainTabProps) {
  const [activeTab, setActiveTab] = useState<"budget" | "latency">("budget");
  const [showTargetModal, setShowTargetModal] = useState(false);
  const requireAuth = useRequireAuth();

  return (
    <div className="border-t border-[var(--border-color)] bg-[var(--background)]">
      {/* tab 切换 + 配置入口 */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-[var(--border-color)]">
        <div className="flex gap-1">
          {[
            { id: "budget" as const, label: t.tabBudget, icon: Activity },
            { id: "latency" as const, label: t.tabLatency, icon: BarChart3 },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg transition-colors ${
                activeTab === tab.id
                  ? "bg-primary/10 text-primary"
                  : "text-muted hover:text-default hover:bg-[var(--hover-bg)]"
              }`}
            >
              <tab.icon className="w-3.5 h-3.5" />
              {tab.label}
            </button>
          ))}
        </div>
        <button
          onClick={() => requireAuth(() => setShowTargetModal(true), "slo.manage")}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg text-muted hover:text-default hover:bg-[var(--hover-bg)] transition-colors"
        >
          <Settings2 className="w-3.5 h-3.5" />
          <span className="hidden sm:inline">{t.configTarget}</span>
        </button>
      </div>

      {activeTab === "budget" && <DomainBudgetTab domain={domain} timeRange={timeRange} clusterId={clusterId} t={t} />}
      {activeTab === "latency" && <DomainLatencyTab domain={domain} timeRange={timeRange} clusterId={clusterId} t={t} />}

      <SLOTargetModal
        isOpen={showTargetModal}
        onClose={() => setShowTargetModal(false)}
        domain={domain.domain}
        clusterId={clusterId}
        currentWindowDays={domain.target?.windowDays}
        t={t}
      />
    </div>
  );
}

/** 预算 tab：数字来自清单那一行，两张趋势图要另取历史点 */
function DomainBudgetTab({ domain, timeRange, clusterId, t }: DomainTabProps) {
  const historyQuery = useQuery(sloDomainHistoryQuery(clusterId, domain.domain, timeRange));
  const targets = {
    availability: domain.target?.availability ?? 99,
    p95Latency: domain.target?.p95Latency ?? 300,
  };

  return (
    <div className="p-4 space-y-4">
      {/* 数字先行：允许错几个、实际错了几个 —— 比任何图都直接 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <GoodBadCount summary={domain.summary} budget={domain.budget} t={t} />
        <BurnRateTable burnRates={domain.budget?.burnRates ?? []} t={t} />
      </div>

      <QueryView query={historyQuery}>
        {(history) => (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <HistoryChart
              history={history}
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
              history={history.map((point) => ({ timestamp: point.timestamp, errorBudget: point.errorBudget }))}
              errorBudgetRemaining={domain.budget?.remainingPct ?? domain.errorBudgetRemaining}
              t={{
                errorBudgetBurn: t.errorBudgetBurn,
                current: t.current,
                estimatedExhaust: t.estimatedExhaust,
                noData: t.noData,
              }}
            />
          </div>
        )}
      </QueryView>
    </div>
  );
}

/** 延迟 tab：延迟分布（桶、方法、状态码），没有数据时由 LatencyTab 显示空状态 */
function DomainLatencyTab({ domain, timeRange, clusterId, t }: DomainTabProps) {
  const latencyQuery = useQuery(sloLatencyDistributionQuery(clusterId, domain.domain, timeRange));

  return (
    <div className="p-4">
      <QueryView query={latencyQuery}>
        {(distribution) => (
          <LatencyTab
            data={distribution}
            timeRange={timeRange}
            t={{
              latencyDistribution: t.latencyDistribution,
              methodBreakdown: t.methodBreakdown,
              statusCodeBreakdown: t.statusCodeBreakdown,
              clearSelection: t.clearSelection,
              requests: t.requests,
              noData: t.noData,
            }}
          />
        )}
      </QueryView>
    </div>
  );
}
