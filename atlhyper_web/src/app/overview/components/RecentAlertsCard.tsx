"use client";

import { memo } from "react";
import { AlertTriangle } from "lucide-react";
import type { TransformedOverview } from "@/types/overview";
import type { useI18n } from "@/i18n/context";

export type AlertItem = TransformedOverview["recentAlerts"][number];

interface RecentAlertsCardProps {
  alerts: TransformedOverview["recentAlerts"];
  onAlertClick: (alert: AlertItem) => void;
  t: ReturnType<typeof useI18n>["t"];
}

export const RecentAlertsCard = memo(function RecentAlertsCard({
  alerts,
  onAlertClick,
  t,
}: RecentAlertsCardProps) {
  const getSeverityStyle = (severity: string) => {
    const s = severity.toLowerCase();
    if (s === "critical") return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";
    if (s === "warning") return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400";
    return "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400";
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-[320px] flex flex-col">
      <h3 className="text-lg font-semibold text-default mb-4 flex-shrink-0">{t.overview.recentAlerts}</h3>
      <div className="flex-1 overflow-y-auto space-y-3 pr-2">
        {alerts.length === 0 ? (
          <div className="text-center py-8 text-muted">{t.overview.noRecentAlerts}</div>
        ) : (
          alerts.map((alert, index) => (
            <div
              key={index}
              className="flex items-start gap-3 p-3 bg-[var(--background)] rounded-lg cursor-pointer hover:bg-[var(--background-hover)] transition-colors"
              onClick={() => onAlertClick(alert)}
            >
              <AlertTriangle className={`w-4 h-4 flex-shrink-0 mt-0.5 ${
                alert.severity === "critical" ? "text-red-500" :
                alert.severity === "warning" ? "text-yellow-500" : "text-blue-500"
              }`} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${getSeverityStyle(alert.severity)}`}>
                    {alert.severity}
                  </span>
                  <span className="text-xs text-muted">{alert.namespace}</span>
                </div>
                <p className="text-sm text-default truncate" title={alert.message}>
                  {alert.message || alert.reason}
                </p>
                <p className="text-xs text-muted mt-1">
                  {new Date(alert.time).toLocaleString()}
                </p>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
});
