"use client";

import type { NotifySeverity } from "@/types/notify";

const SEVERITY_COLORS: Record<NotifySeverity, string> = {
  low: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  medium: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  high: "bg-orange-500/15 text-orange-600 dark:text-orange-400",
  critical: "bg-red-500/15 text-red-600 dark:text-red-400",
};

interface SeverityBadgeProps {
  severity: NotifySeverity;
  labels: Record<NotifySeverity, string>;
}

/** 严重度徽标，配色与 AIOps 事件页一致 */
export function SeverityBadge({ severity, labels }: SeverityBadgeProps) {
  return (
    <span
      className={`inline-flex px-2 py-1 text-xs font-medium rounded-full whitespace-nowrap ${
        SEVERITY_COLORS[severity] ?? "bg-gray-500/15 text-muted"
      }`}
    >
      {labels[severity] ?? severity}
    </span>
  );
}
