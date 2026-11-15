"use client";

import { RiskBadge } from "@/components/aiops/RiskBadge";
import { useI18n } from "@/i18n/context";
import type { Incident } from "@/api/aiops";
import { formatDuration, INCIDENT_STATE_COLORS } from "./incident-format";

interface IncidentFactsProps {
  incident: Incident;
}

/** 事件详情顶部的概况：状态、峰值严重度，以及时长、峰值风险、复发次数、升级时间、关闭原因 */
export function IncidentFacts({ incident }: IncidentFactsProps) {
  const { t } = useI18n();
  const closeReasonLabel = t.aiops.closeReasonLabel[incident.closeReason === "" ? "open" : incident.closeReason];

  const facts: { label: string; value: string; hint?: string; warn?: boolean }[] = [
    { label: t.aiops.startedAt, value: new Date(incident.startedAt).toLocaleString() },
    { label: t.aiops.duration, value: formatDuration(incident.durationS, t.aiops.minutes) },
    { label: t.aiops.peakRisk, value: incident.peakRisk.toFixed(1) },
    { label: t.aiops.recurrence, value: String(incident.recurrence) },
    {
      label: t.aiops.escalatedAt,
      value: incident.escalatedAt ? new Date(incident.escalatedAt).toLocaleString() : t.aiops.notEscalated,
    },
    {
      label: t.aiops.closeReason,
      // 没见过的取值原样显示，不吞掉
      value: closeReasonLabel ?? incident.closeReason,
      hint: incident.closeReason === "stale" ? t.aiops.closeReasonStaleHint : undefined,
      warn: incident.closeReason === "stale",
    },
  ];

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span
          title={t.aiops.stateDesc[incident.state as keyof typeof t.aiops.stateDesc] ?? ""}
          className={`px-2 py-1 rounded-full text-xs font-medium ${INCIDENT_STATE_COLORS[incident.state] ?? ""}`}
        >
          {t.aiops.state[incident.state as keyof typeof t.aiops.state] ?? incident.state}
        </span>
        <span title={t.aiops.severityTooltip}>
          <RiskBadge level={incident.severity} size="md" />
        </span>
      </div>

      <dl className="grid grid-cols-2 sm:grid-cols-3 gap-x-4 gap-y-3">
        {facts.map((fact) => (
          <div key={fact.label} title={fact.hint}>
            <dt className="text-xs text-muted">{fact.label}</dt>
            <dd
              className={`text-sm font-medium mt-0.5 ${
                fact.warn ? "text-orange-600 dark:text-orange-400" : "text-default"
              }`}
            >
              {fact.value}
            </dd>
          </div>
        ))}
      </dl>

      {incident.closeReason === "stale" && (
        <p className="text-xs text-orange-600 dark:text-orange-400">{t.aiops.closeReasonStaleHint}</p>
      )}
    </div>
  );
}
