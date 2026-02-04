"use client";

import { memo } from "react";
import { CheckCircle, XCircle, AlertCircle } from "lucide-react";
import type { TransformedOverview } from "@/types/overview";
import type { useI18n } from "@/i18n/context";

interface HealthCardProps {
  data: TransformedOverview["healthCard"];
  t: ReturnType<typeof useI18n>["t"];
}

export const HealthCard = memo(function HealthCard({ data, t }: HealthCardProps) {
  const getStatusColor = (status: string) => {
    const s = status.toLowerCase();
    if (s === "healthy") return "text-green-500";
    if (s === "degraded") return "text-yellow-500";
    return "text-red-500";
  };

  const getStatusBg = (status: string) => {
    const s = status.toLowerCase();
    if (s === "healthy") return "bg-green-100 dark:bg-green-900/30";
    if (s === "degraded") return "bg-yellow-100 dark:bg-yellow-900/30";
    return "bg-red-100 dark:bg-red-900/30";
  };

  const StatusIcon = data.status.toLowerCase() === "healthy" ? CheckCircle :
    data.status.toLowerCase() === "degraded" ? AlertCircle : XCircle;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-full">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-default">{t.overview.clusterHealth}</h3>
        <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${getStatusBg(data.status)} ${getStatusColor(data.status)}`}>
          <StatusIcon className="w-3 h-3" />
          {data.status}
        </span>
      </div>
      {data.reason && (
        <p className="text-sm text-muted mb-4 truncate" title={data.reason}>{data.reason}</p>
      )}
      <div className="space-y-3">
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span className="text-muted">{t.overview.nodeReady}</span>
            <span className="text-default transition-all duration-300">{data.nodeReadyPct.toFixed(1)}%</span>
          </div>
          <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${data.nodeReadyPct >= 90 ? "bg-green-500" : data.nodeReadyPct >= 70 ? "bg-yellow-500" : "bg-red-500"}`}
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
              className={`h-full rounded-full transition-all duration-300 ${data.podHealthyPct >= 90 ? "bg-green-500" : data.podHealthyPct >= 70 ? "bg-yellow-500" : "bg-red-500"}`}
              style={{ width: `${Math.min(100, data.podHealthyPct)}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  );
});
