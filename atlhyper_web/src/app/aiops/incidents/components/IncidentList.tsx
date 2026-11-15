"use client";

import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import type { Incident } from "@/api/aiops";
import { formatDuration, formatShortTime, INCIDENT_STATE_COLORS } from "./incident-format";
import { SubjectTitle } from "./SubjectTitle";

const SEVERITY_COLORS: Record<string, string> = {
  low: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  medium: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  high: "bg-orange-500/15 text-orange-600 dark:text-orange-400",
  critical: "bg-red-500/15 text-red-600 dark:text-red-400",
};

/** ID | 状态 | 严重度 | 主体（+ 根因实体） | 峰值 | 开始 | 持续 | 复发 | 升级时间 */
const GRID_COLUMNS = "grid grid-cols-[72px_88px_64px_minmax(0,1fr)_64px_92px_76px_52px_92px] gap-2";
const BADGE_CLASS = "inline-flex items-center text-[10px] px-1.5 py-0.5 rounded-full font-medium whitespace-nowrap";

interface IncidentListProps {
  incidents: Incident[];
  onSelect: (id: string) => void;
}

export function IncidentList({ incidents, onSelect }: IncidentListProps) {
  const { t } = useI18n();

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-x-auto">
      {/* 列多：窄屏时整张表横向滚动，不把列挤扁 */}
      <div className="min-w-[860px]">
        {/* 表头 */}
        <div className={`${GRID_COLUMNS} px-5 py-2 text-[10px] text-muted uppercase tracking-wider border-b border-[var(--border-color)]/50`}>
          <span>{t.aiops.incidentId}</span>
          <span title={t.aiops.incidentStateTooltip} className="cursor-help border-b border-dotted border-current">{t.aiops.incidentState}</span>
          <span title={t.aiops.severityTooltip} className="cursor-help border-b border-dotted border-current">{t.aiops.severity}</span>
          <span>{t.aiops.subject}</span>
          <span>{t.aiops.peakRisk}</span>
          <span>{t.aiops.startedAt}</span>
          <span>{t.aiops.duration}</span>
          <span>{t.aiops.recurrence}</span>
          <span>{t.aiops.escalatedAt}</span>
        </div>

        {/* 行 */}
        {incidents.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
        ) : (
          incidents.map((inc) => (
            <button
              key={inc.id}
              onClick={() => onSelect(inc.id)}
              className={`w-full ${GRID_COLUMNS} px-5 py-2.5 text-sm text-left hover:bg-[var(--hover-bg)] transition-colors items-center border-b border-[var(--border-color)]/20 last:border-b-0`}
            >
              <span className="font-mono text-xs text-default">{inc.id}</span>
              <span className="flex flex-col items-start gap-1">
                <span
                  title={t.aiops.stateDesc[inc.state as keyof typeof t.aiops.stateDesc] ?? ""}
                  className={`${BADGE_CLASS} ${INCIDENT_STATE_COLORS[inc.state] ?? ""}`}
                >
                  {t.aiops.state[inc.state as keyof typeof t.aiops.state] ?? inc.state}
                </span>
                {/* 失联关闭不是真的恢复了，要一眼看得出来 */}
                {inc.closeReason === "stale" && (
                  <span
                    title={t.aiops.closeReasonStaleHint}
                    className={`${BADGE_CLASS} bg-orange-500/15 text-orange-600 dark:text-orange-400`}
                  >
                    {t.aiops.closeReasonLabel.stale}
                  </span>
                )}
              </span>
              <span
                title={t.aiops.severityDesc[inc.severity as keyof typeof t.aiops.severityDesc] ?? ""}
                className={`${BADGE_CLASS} justify-self-start ${SEVERITY_COLORS[inc.severity] ?? ""}`}
              >
                {t.aiops.severityLevel[inc.severity as keyof typeof t.aiops.severityLevel] ?? inc.severity}
              </span>
              <div className="min-w-0 space-y-0.5">
                <SubjectTitle incident={inc} />
                {/* 领头实体：主体的成员里风险最高的那个 */}
                <div className="flex items-center gap-1.5 text-xs text-muted min-w-0">
                  <span className="flex-shrink-0">{t.aiops.rootCause}:</span>
                  <EntityLink entityKey={inc.rootCause} />
                </div>
              </div>
              <span className="justify-self-start">
                <RiskBadge level={inc.peakRisk >= 80 ? "critical" : inc.peakRisk >= 50 ? "warning" : "low"} />
              </span>
              <span className="text-xs text-muted">{formatShortTime(inc.startedAt)}</span>
              <span className="text-xs text-muted">{formatDuration(inc.durationS, t.aiops.minutes)}</span>
              <span className="text-xs text-muted text-center tabular-nums">{inc.recurrence}</span>
              <span className="text-xs text-muted">
                {inc.escalatedAt ? formatShortTime(inc.escalatedAt) : t.aiops.notEscalated}
              </span>
            </button>
          ))
        )}
      </div>
    </div>
  );
}
