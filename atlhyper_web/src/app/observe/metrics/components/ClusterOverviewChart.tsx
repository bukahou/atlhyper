"use client";

import { useMemo, memo } from "react";
import { useQueries, type UseQueryResult } from "@tanstack/react-query";
import { TrendingUp, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { nodeMetricsHistoryQuery } from "@/queries/observe-metrics";
import type { NodeMetricsHistory } from "@/datasource/metrics";
import type { NodeMetrics } from "@/types/node-metrics";
import { METRICS } from "./chartConstants";
import { MetricChart } from "./MetricChart";

// ==================== Main Container ====================

interface ClusterOverviewChartProps {
  nodes: NodeMetrics[];
  clusterId: string;
  /** 时间范围（小时），由页面级 TimeRangePicker 下发（P3 统一时间控制） */
  hours: number;
}

/** 把各节点的历史查询合成两样：按查询顺序排列的历史、是否还有节点在首次加载 */
function combineNodeHistories(results: UseQueryResult<NodeMetricsHistory>[]) {
  return {
    histories: results.map((result) => result.data),
    anyPending: results.some((result) => result.isPending),
  };
}

export const ClusterOverviewChart = memo(function ClusterOverviewChart({
  nodes,
  clusterId,
  hours,
}: ClusterOverviewChartProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;

  // Top 5 nodes per metric (each metric sorts independently)
  const topNodesByMetric = useMemo(() => {
    const result: Record<string, string[]> = {};
    for (const m of METRICS) {
      result[m.key] = [...nodes]
        .sort((a, b) => m.sortSnapshot(b) - m.sortSnapshot(a))
        .slice(0, 5)
        .map((n) => n.nodeName);
    }
    return result;
  }, [nodes]);

  // Union of all top-5 lists for history fetching
  const allTopNodeNames = useMemo(() => {
    const set = new Set<string>();
    for (const names of Object.values(topNodesByMetric)) {
      for (const n of names) set.add(n);
    }
    return Array.from(set);
  }, [topNodesByMetric]);

  // 每个 top 节点一个历史查询，各自缓存、各自按分钟刷新。
  // 快照每 10 秒刷新一轮，名单会重算，但没换的节点键不变，不会跟着重新请求
  const { histories, anyPending } = useQueries({
    queries: allTopNodeNames.map((nodeName) => nodeMetricsHistoryQuery(clusterId, nodeName, hours)),
    combine: combineNodeHistories,
  });

  /** Per-node history: { nodeName: { metricKey: Point[] } } */
  const historyMap = useMemo(() => {
    const map: Record<string, NodeMetricsHistory> = {};
    allTopNodeNames.forEach((nodeName, index) => {
      const history = histories[index];
      if (history) map[nodeName] = history;
    });
    return map;
  }, [allTopNodeNames, histories]);

  const metricTitles: Record<string, string> = {
    cpu: nm.cpu.title,
    memory: nm.memory.title,
    disk: nm.disk.title,
    temp: nm.temperature.title,
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-teal-500/10 rounded-lg">
            <TrendingUp className="w-5 h-5 text-teal-500" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-default">{nm.overview.title}</h3>
            <span className="text-[10px] text-muted">{nm.overview.top5}</span>
          </div>
        </div>
      </div>

      {/* 2x2 Grid */}
      <div className="relative">
        {anyPending && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-card/80 rounded-lg">
            <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
          </div>
        )}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
          {METRICS.map((m) => (
            <MetricChart
              key={m.key}
              title={metricTitles[m.key]}
              unit={m.unit}
              metricKey={m.key}
              nodeNames={topNodesByMetric[m.key] || []}
              historyMap={historyMap}
              hours={hours}
            />
          ))}
        </div>
      </div>

    </div>
  );
});
