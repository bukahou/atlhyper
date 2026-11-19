"use client";

import { memo } from "react";
import { CheckCircle, XCircle, AlertCircle } from "lucide-react";
import type { TransformedOverview } from "@/types/overview";
import type { useI18n } from "@/i18n/context";
import { clusterHealthLevel } from "@/lib/cluster-health";
import { hardwareStatusBar, hardwareStatusChip } from "@/lib/hardware-status";

interface HealthCardProps {
  data: TransformedOverview["healthCard"];
  t: ReturnType<typeof useI18n>["t"];
}

export const HealthCard = memo(function HealthCard({ data, t }: HealthCardProps) {
  // 结论与原因都由后端判定，这里只换算颜色、出文案
  const StatusIcon = data.status === "Healthy" ? CheckCircle : data.status === "Degraded" ? AlertCircle : XCircle;
  const reasonText = data.reason ? t.overview.healthReasons[data.reason] ?? data.reason : "";

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-full">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-default">{t.overview.clusterHealth}</h3>
        <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${hardwareStatusChip(clusterHealthLevel(data.status))}`}>
          <StatusIcon className="w-3 h-3" />
          {t.overview.healthStatus[data.status] ?? data.status}
        </span>
      </div>
      {reasonText && (
        <p className="text-sm text-muted mb-4 truncate" title={reasonText}>{reasonText}</p>
      )}
      <div className="space-y-3">
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span className="text-muted">{t.overview.nodeReady}</span>
            <span className="text-default transition-all duration-300">{data.nodeReadyPct.toFixed(1)}%</span>
          </div>
          <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${hardwareStatusBar(clusterHealthLevel(data.nodeReadyStatus))}`}
              style={{ width: `${Math.min(100, data.nodeReadyPct)}%` }}
            />
          </div>
        </div>
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span className="text-muted">{t.overview.podHealthy}</span>
            <span className="text-default transition-all duration-300">{data.podHealthyPct.toFixed(1)}%</span>
          </div>
          <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${hardwareStatusBar(clusterHealthLevel(data.podReadyStatus))}`}
              style={{ width: `${Math.min(100, data.podHealthyPct)}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  );
});
