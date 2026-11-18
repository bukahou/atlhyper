"use client";

import { RiskBadge } from "@/components/aiops/RiskBadge";
import { useI18n } from "@/i18n/context";
import type { IncidentDetail } from "@/api/aiops";
import { incidentSubject } from "@/lib/entity-key";
import { INCIDENT_STATE_COLORS } from "./incident-format";
import { headlineSentence, impactItems, verdictMetaParts } from "./incident-view-format";

/** 结论块的边框与底色：跟着事件状态走 */
const VERDICT_TONES: Record<string, string> = {
  warning: "border-yellow-500/30 bg-yellow-500/5",
  incident: "border-red-500/30 bg-red-500/5",
  recovery: "border-blue-500/30 bg-blue-500/5",
  stable: "border-[var(--border-color)] bg-[var(--background)]",
};

interface IncidentVerdictProps {
  incident: IncidentDetail;
  /** 时间线里有合并的检测记录（见 headlineSentence） */
  hasDetections: boolean;
}

/** 结论：状态 · 严重度 · 已持续 · 开始 / 升级时间，一句话说清出了什么事，再给影响数字 */
export function IncidentVerdict({ incident, hasDetections }: IncidentVerdictProps) {
  const { t } = useI18n();
  const texts = t.aiops.detail;
  const subject = incidentSubject(incident);
  const typeLabels: Record<string, string> = t.aiops.subjectTypeLabel;
  const closed = incident.closeReason !== "";
  const stale = incident.closeReason === "stale";
  const closeReasonLabels: Record<string, string> = t.aiops.closeReasonLabel;
  // 没见过的取值原样显示，不吞掉
  const closeReasonLabel = closeReasonLabels[incident.closeReason] ?? incident.closeReason;

  const sentence = headlineSentence(
    {
      headline: incident.headline,
      impact: incident.impact,
      subjectType: subject.type,
      subjectTypeLabel: typeLabels[subject.type] ?? subject.type,
      hasDetections,
    },
    texts,
  );
  const impacts = impactItems(incident.impact, incident.peakRisk, texts);
  const meta = verdictMetaParts(incident, texts, t.aiops.minutes);

  return (
    <div className={`rounded-xl border px-4 py-3.5 ${VERDICT_TONES[incident.state] ?? VERDICT_TONES.stable}`}>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5 mb-2">
        <span
          title={t.aiops.stateDesc[incident.state as keyof typeof t.aiops.stateDesc] ?? ""}
          className={`px-2 py-0.5 rounded-full text-xs font-medium ${INCIDENT_STATE_COLORS[incident.state] ?? ""}`}
        >
          {t.aiops.state[incident.state as keyof typeof t.aiops.state] ?? incident.state}
        </span>
        <span title={t.aiops.severityTooltip}>
          <RiskBadge level={incident.severity} size="md" />
        </span>
        <span className="text-xs text-muted">{meta.join(" · ")}</span>
        {/* 关闭原因只在事件关闭后显示；失联关闭要提醒「不代表已恢复」 */}
        {closed && (
          <span
            title={stale ? t.aiops.closeReasonStaleHint : undefined}
            className={`text-xs ${stale ? "text-orange-600 dark:text-orange-400" : "text-muted"}`}
          >
            · {closeReasonLabel}
          </span>
        )}
      </div>

      <p
        className={`text-[15px] leading-relaxed ${
          incident.headline ? "font-semibold text-default" : "font-medium text-muted"
        }`}
      >
        {sentence}
      </p>

      <div className="mt-2.5 flex flex-wrap gap-x-5 gap-y-1 text-xs text-muted tabular-nums">
        {impacts.map((item) => (
          <span key={item}>{item}</span>
        ))}
      </div>

      {stale && <p className="mt-2 text-xs text-orange-600 dark:text-orange-400">{t.aiops.closeReasonStaleHint}</p>}
    </div>
  );
}
