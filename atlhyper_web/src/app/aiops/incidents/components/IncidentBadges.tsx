"use client";

import { useI18n } from "@/i18n/context";
import { INCIDENT_STATE_COLORS } from "./incident-format";

const SEVERITY_COLORS: Record<string, string> = {
  low: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  medium: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  high: "bg-orange-500/15 text-orange-600 dark:text-orange-400",
  critical: "bg-red-500/15 text-red-600 dark:text-red-400",
};

export const BADGE_CLASS = "inline-flex items-center text-[10px] px-1.5 py-0.5 rounded-full font-medium whitespace-nowrap";

/** 事件状态徽标（列表与影响范围共用）；没见过的状态原样显示 */
export function IncidentStateBadge({ state }: { state: string }) {
  const { t } = useI18n();
  return (
    <span
      title={t.aiops.stateDesc[state as keyof typeof t.aiops.stateDesc] ?? ""}
      className={`${BADGE_CLASS} ${INCIDENT_STATE_COLORS[state] ?? ""}`}
    >
      {t.aiops.state[state as keyof typeof t.aiops.state] ?? state}
    </span>
  );
}

/** 峰值严重度徽标（列表与影响范围共用）；没见过的取值原样显示 */
export function IncidentSeverityBadge({ severity }: { severity: string }) {
  const { t } = useI18n();
  return (
    <span
      title={t.aiops.severityDesc[severity as keyof typeof t.aiops.severityDesc] ?? ""}
      className={`${BADGE_CLASS} justify-self-start ${SEVERITY_COLORS[severity] ?? ""}`}
    >
      {t.aiops.severityLevel[severity as keyof typeof t.aiops.severityLevel] ?? severity}
    </span>
  );
}
