"use client";

import { useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { QueryView, TimeRangePicker } from "@/components/common";
import { OTelGuard } from "@/components/observe/OTelGuard";
import { SignalFreshnessBadge } from "@/components/observe/SignalFreshnessBadge";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { useI18n } from "@/i18n/context";
import { toSpanMs } from "@/lib/time-range";
import { metricsKeys } from "@/queries/keys";
import { hardwareHealthQuery, metricsSnapshotQuery } from "@/queries/observe-metrics";
import { MetricsSnapshotView } from "./components/MetricsSnapshotView";

// ==================== 主页面 ====================
export default function MetricsPage() {
  return (
    <OTelGuard>
      <MetricsPageContent />
    </OTelGuard>
  );
}

function MetricsPageContent() {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const { clusterId } = useCurrentCluster();

  const [expandedNode, setExpandedNode] = useState<string | null>(null);

  // P3 统一：全页唯一时间控制点，作用于所有趋势图（集群趋势 + 节点展开趋势）。
  // 快照类内容（矩阵/速览）天然是当前值，不受时间范围影响 —— 这是快照的定义，无需提示。
  const { selection: timeSelection, setSelection: setTimeSelection } = useObserveTimeRange("trendOnly");
  const freshness = useSignalFreshness("metrics");
  const historyHours = Math.max(1, Math.round(toSpanMs(timeSelection) / 3_600_000));

  // 快照与硬件健康是两个查询，并行取；硬件接口失败只是矩阵缺判定，不拖垮整页
  const snapshotQuery = useQuery(metricsSnapshotQuery(clusterId));
  const hardwareQuery = useQuery(hardwareHealthQuery(clusterId));
  const { refetch: refreshMetrics, isRefetching } = useManualRefetch(metricsKeys.all);

  const handleNodeToggle = useCallback((nodeName: string) => {
    setExpandedNode((current) => (current === nodeName ? null : nodeName));
  }, []);

  const lastUpdateText =
    snapshotQuery.dataUpdatedAt > 0 ? new Date(snapshotQuery.dataUpdatedAt).toLocaleTimeString() : "—";

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-6">
        {/* 标题栏 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg sm:text-xl font-bold text-default">{t.nav.metrics}</h1>
            <p className="text-xs sm:text-sm text-muted mt-1">
              {nm.pageDescription}
            </p>
          </div>
          <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
            <SignalFreshnessBadge item={freshness} />
            <TimeRangePicker value={timeSelection} onChange={setTimeSelection} t={nm} />
            <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
              {nm.lastUpdate}: {lastUpdateText}
            </span>
            <button
              onClick={refreshMetrics}
              disabled={isRefetching}
              title={t.common.refresh}
              className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefetching ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        <QueryView query={snapshotQuery} requireCluster>
          {(snapshot) => (
            <MetricsSnapshotView
              snapshot={snapshot}
              hardware={hardwareQuery.data ?? null}
              clusterId={clusterId}
              expandedNode={expandedNode}
              onNodeToggle={handleNodeToggle}
              historyHours={historyHours}
              trendSpanMs={toSpanMs(timeSelection)}
            />
          )}
        </QueryView>
      </div>
    </Layout>
  );
}
