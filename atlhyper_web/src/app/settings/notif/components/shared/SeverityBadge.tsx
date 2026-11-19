"use client";

import type { NotifySeverity } from "@/types/notify";
import { levelBadgeClass } from "@/lib/risk";

interface SeverityBadgeProps {
  severity: NotifySeverity;
  labels: Record<NotifySeverity, string>;
}

/** 严重度徽标，配色与 AIOps 事件页一致 */
export function SeverityBadge({ severity, labels }: SeverityBadgeProps) {
  return (
    <span
      className={`inline-flex px-2 py-1 text-xs font-medium rounded-full whitespace-nowrap ${
        levelBadgeClass(severity)
      }`}
    >
      {labels[severity] ?? severity}
    </span>
  );
}
