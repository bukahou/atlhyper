"use client";

import { useI18n } from "@/i18n/context";
import { levelBadgeClass } from "@/lib/risk";
import { INCIDENT_STATE_COLORS } from "./incident-format";

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
      className={`${BADGE_CLASS} justify-self-start ${levelBadgeClass(severity)}`}
    >
      {t.aiops.severityLevel[severity as keyof typeof t.aiops.severityLevel] ?? severity}
    </span>
  );
}
