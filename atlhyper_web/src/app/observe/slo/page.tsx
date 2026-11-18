"use client";

import { useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, RefreshCw } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { QueryView, TimeRangePicker } from "@/components/common";
import { OTelGuard } from "@/components/observe/OTelGuard";
import { SignalFreshnessBadge } from "@/components/observe/SignalFreshnessBadge";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { useI18n } from "@/i18n/context";
import { sloDomainsQuery } from "@/queries/observe-slo";
import { SLODomainsView } from "./components/SLODomainsView";

export default function SLOPage() {
  return (
    <OTelGuard>
      <SLOPageContent />
    </OTelGuard>
  );
}

function SLOPageContent() {
  const { t } = useI18n();
  const sloT = t.slo;
  // 用侧栏选定的集群：换集群，键跟着换，清单就是新集群的
  const { clusterId } = useCurrentCluster();
  const [expandedDomain, setExpandedDomain] = useState<string | null>(null);
  // 全局时间轴 + 贴合到 SLO 的五个预聚合窗口。
  // SLO 查不了任意范围（窗口是 Agent 预先算好缓存的），贴合后必须告诉用户实际用了哪个窗口。
  const {
    selection: timeSelection,
    setSelection: setTimeSelection,
    sloWindow,
    degraded,
    degradeTo,
  } = useObserveTimeRange("sloWindows");
  // SLO 的数据源是 ingress 指标，跟着 metrics 的新鲜度走
  const freshness = useSignalFreshness("metrics");

  const domainsQuery = useQuery(sloDomainsQuery(clusterId, sloWindow));

  const handleToggleDomain = useCallback((domain: string) => {
    setExpandedDomain((current) => (current === domain ? null : domain));
  }, []);

  return (
    <Layout>
      <div className="-m-6 min-h-[calc(100vh-3.5rem)] bg-[var(--background)]">
        {/* Header */}
        <div className="px-4 sm:px-6 py-4 border-b border-[var(--border-color)] bg-card">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-xl bg-gradient-to-br from-violet-100 to-indigo-100 dark:from-violet-900/30 dark:to-indigo-900/30">
                <Activity className="w-5 h-5 sm:w-6 sm:h-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <h1 className="text-base sm:text-lg font-semibold text-default">{sloT.pageTitle}</h1>
                <p className="text-xs text-muted hidden sm:block">{sloT.pageDescription}</p>
              </div>
            </div>
            <div className="flex items-center gap-2 sm:gap-3 self-end sm:self-auto">
              <div className="flex items-center gap-2">
                <SignalFreshnessBadge item={freshness} />
                <TimeRangePicker value={timeSelection} onChange={setTimeSelection} t={sloT} />
                {degraded && (
                  <span className="text-[10px] text-amber-500 whitespace-nowrap" title={sloT.windowDegradedHint}>
                    {sloT.windowDegraded.replace("{window}", degradeTo ?? "")}
                  </span>
                )}
              </div>
              <button
                onClick={() => void domainsQuery.refetch()}
                disabled={domainsQuery.isFetching}
                title={t.common.refresh}
                className="p-2.5 sm:p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${domainsQuery.isFetching ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>
        </div>

        <div className="p-4 sm:p-6">
          <QueryView query={domainsQuery} requireCluster>
            {(domainList) => (
              <SLODomainsView
                domainList={domainList}
                timeRange={sloWindow}
                clusterId={clusterId}
                expandedDomain={expandedDomain}
                onToggleDomain={handleToggleDomain}
                sloT={sloT}
              />
            )}
          </QueryView>
        </div>
      </div>
    </Layout>
  );
}
