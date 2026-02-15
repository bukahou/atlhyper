"use client";

import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import type { Incident } from "@/api/aiops";

const STATE_COLORS: Record<string, string> = {
  warning: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  incident: "bg-red-500/15 text-red-600 dark:text-red-400",
  recovery: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  stable: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
};

const SEVERITY_COLORS: Record<string, string> = {
  low: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  medium: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  high: "bg-orange-500/15 text-orange-600 dark:text-orange-400",
  critical: "bg-red-500/15 text-red-600 dark:text-red-400",
};

interface IncidentListProps {
  incidents: Incident[];
  onSelect: (id: string) => void;
}

function formatDuration(seconds: number, minuteLabel: string): string {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} ${minuteLabel}`;
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  return `${hours}h ${mins}m`;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}`;
}

export function IncidentList({ incidents, onSelect }: IncidentListProps) {
  const { t } = useI18n();

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      {/* 表头 */}
      <div className="grid grid-cols-[80px_70px_70px_2fr_70px_100px_80px_60px] gap-2 px-5 py-2 text-[10px] text-muted uppercase tracking-wider border-b border-[var(--border-color)]/50">
        <span>{t.aiops.incidentId}</span>
        <span>{t.aiops.incidentState}</span>
        <span>{t.aiops.severity}</span>
        <span>{t.aiops.rootCause}</span>
        <span>{t.aiops.peakRisk}</span>
        <span>{t.aiops.startedAt}</span>
        <span>{t.aiops.duration}</span>
        <span>{t.aiops.recurrence}</span>
      </div>

      {/* 行 */}
      {incidents.length === 0 ? (
        <div className="px-5 py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
      ) : (
        incidents.map((inc) => (
          <button
            key={inc.id}
            onClick={() => onSelect(inc.id)}
            className="w-full grid grid-cols-[80px_70px_70px_2fr_70px_100px_80px_60px] gap-2 px-5 py-2.5 text-sm hover:bg-[var(--hover-bg)] transition-colors items-center border-b border-[var(--border-color)]/20 last:border-b-0"
          >
            <span className="font-mono text-xs text-default">{inc.id}</span>
            <span className={`inline-flex items-center text-[10px] px-1.5 py-0.5 rounded-full font-medium ${STATE_COLORS[inc.state] ?? ""}`}>
              {t.aiops.state[inc.state as keyof typeof t.aiops.state] ?? inc.state}
            </span>
            <span className={`inline-flex items-center text-[10px] px-1.5 py-0.5 rounded-full font-medium ${SEVERITY_COLORS[inc.severity] ?? ""}`}>
              {t.aiops.severityLevel[inc.severity as keyof typeof t.aiops.severityLevel] ?? inc.severity}
            </span>
            <div className="min-w-0">
              <EntityLink entityKey={inc.rootCause} />
            </div>
            <RiskBadge level={inc.peakRisk >= 80 ? "critical" : inc.peakRisk >= 50 ? "warning" : "low"} />
            <span className="text-xs text-muted">{formatTime(inc.startedAt)}</span>
            <span className="text-xs text-muted">{formatDuration(inc.durationS, t.aiops.minutes)}</span>
            <span className="text-xs text-muted text-center">{inc.recurrence}</span>
          </button>
        ))
      )}
    </div>
  );
}
