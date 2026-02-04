"use client";

import { memo } from "react";
import { Package, Layers, Box, Clock } from "lucide-react";
import type { TransformedOverview } from "@/types/overview";
import type { useI18n } from "@/i18n/context";

interface WorkloadSummaryCardProps {
  workloads: TransformedOverview["workloads"];
  podStatus: TransformedOverview["podStatus"];
  peakStats: TransformedOverview["peakStats"];
  t: ReturnType<typeof useI18n>["t"];
}

export const WorkloadSummaryCard = memo(function WorkloadSummaryCard({
  workloads,
  podStatus,
  peakStats,
  t,
}: WorkloadSummaryCardProps) {
  const workloadItems = [
    { name: t.overview.deploymentsLabel, icon: Package, total: workloads.deployments.total, ready: workloads.deployments.ready, color: "#6366F1" },
    { name: t.overview.daemonSetsLabel, icon: Layers, total: workloads.daemonsets.total, ready: workloads.daemonsets.ready, color: "#8B5CF6" },
    { name: t.overview.statefulSetsLabel, icon: Box, total: workloads.statefulsets.total, ready: workloads.statefulsets.ready, color: "#EC4899" },
  ];

  const getStatusColor = (ready: number, total: number) => {
    if (total === 0) return "text-muted";
    if (ready === total) return "text-green-500";
    if (ready >= total * 0.5) return "text-yellow-500";
    return "text-red-500";
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4 h-[290px] flex flex-col">
      <h3 className="text-base font-semibold text-default mb-2 flex-shrink-0">{t.overview.workloadSummary}</h3>

      <div className="flex-1 space-y-2 overflow-hidden">
        {/* 工作负载统计 - 3列 */}
        <div className="grid grid-cols-3 gap-2">
          {workloadItems.map((item) => (
            <div key={item.name} className="bg-[var(--background)] rounded-lg p-2">
              <div className="flex items-center gap-1.5 mb-1">
                <item.icon className="w-3.5 h-3.5" style={{ color: item.color }} />
                <span className="text-xs text-muted truncate">{item.name}</span>
              </div>
              <div className={`text-base font-bold ${getStatusColor(item.ready, item.total)}`}>
                {item.ready}/{item.total}
              </div>
            </div>
          ))}
        </div>

        {/* Jobs 单独一行，显示更多信息 */}
        <div className="bg-[var(--background)] rounded-lg p-2 flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5 text-amber-500" />
            <span className="text-xs text-muted">{t.overview.jobsLabel}</span>
            <span className="text-sm font-bold text-default ml-1">{workloads.jobs.total}</span>
          </div>
          <div className="flex gap-3 text-xs">
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-500" />
              <span className="text-muted">{t.overview.run}</span> <strong>{workloads.jobs.running}</strong>
            </span>
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
              <span className="text-muted">{t.overview.done}</span> <strong>{workloads.jobs.succeeded}</strong>
            </span>
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
              <span className="text-muted">{t.overview.fail}</span> <strong>{workloads.jobs.failed}</strong>
            </span>
          </div>
        </div>

        {/* Pod 状态分布 */}
        <div className="bg-[var(--background)] rounded-lg p-2.5">
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-xs text-muted">{t.overview.podStatus}</span>
            <span className="text-xs text-muted">{t.common.total}: {podStatus.total}</span>
          </div>
          <div className="h-2.5 bg-[var(--card-bg)] rounded-full overflow-hidden flex">
            {podStatus.runningPercent > 0 && (
              <div
                className="h-full bg-green-500"
                style={{ width: `${podStatus.runningPercent}%` }}
                title={`Running: ${podStatus.running} (${podStatus.runningPercent.toFixed(1)}%)`}
              />
            )}
            {podStatus.pendingPercent > 0 && (
              <div
                className="h-full bg-yellow-500"
                style={{ width: `${podStatus.pendingPercent}%` }}
                title={`Pending: ${podStatus.pending} (${podStatus.pendingPercent.toFixed(1)}%)`}
              />
            )}
            {podStatus.succeededPercent > 0 && (
              <div
                className="h-full bg-blue-500"
                style={{ width: `${podStatus.succeededPercent}%` }}
                title={`Succeeded: ${podStatus.succeeded} (${podStatus.succeededPercent.toFixed(1)}%)`}
              />
            )}
            {podStatus.failedPercent > 0 && (
              <div
                className="h-full bg-red-500"
                style={{ width: `${podStatus.failedPercent}%` }}
                title={`Failed: ${podStatus.failed} (${podStatus.failedPercent.toFixed(1)}%)`}
              />
            )}
          </div>
          <div className="flex gap-3 mt-1.5 text-xs">
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-green-500" />
              {podStatus.running}
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-yellow-500" />
              {podStatus.pending}
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-blue-500" />
              {podStatus.succeeded}
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-red-500" />
              {podStatus.failed}
            </span>
          </div>
        </div>

        {/* 峰值统计 */}
        {peakStats.hasData && (
          <div className="flex gap-3 text-xs text-muted px-1">
            <span>{t.overview.cpuPeak}: <strong className="text-orange-500">{peakStats.peakCpu.toFixed(1)}%</strong></span>
            <span>{t.overview.memPeak}: <strong className="text-green-500">{peakStats.peakMem.toFixed(1)}%</strong></span>
          </div>
        )}
      </div>
    </div>
  );
});
