"use client";

import { AlertTriangle, Activity, Clock, RotateCcw } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { IncidentStats as IncidentStatsType } from "@/api/aiops";

interface IncidentStatsProps {
  stats: IncidentStatsType;
}

function formatMTTR(seconds: number, minuteLabel: string): string {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} ${minuteLabel}`;
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  return `${hours}h ${mins}m`;
}

export function IncidentStats({ stats }: IncidentStatsProps) {
  const { t } = useI18n();

  const cards = [
    {
      icon: AlertTriangle,
      label: t.aiops.stats.total,
      value: stats.totalIncidents.toString(),
      color: "bg-blue-500/10 text-blue-500",
    },
    {
      icon: Activity,
      label: t.aiops.stats.active,
      value: stats.activeIncidents.toString(),
      color: stats.activeIncidents > 0 ? "bg-red-500/10 text-red-500" : "bg-emerald-500/10 text-emerald-500",
    },
    {
      icon: Clock,
      label: t.aiops.stats.mttr,
      value: formatMTTR(stats.mttr, t.aiops.minutes),
      color: "bg-yellow-500/10 text-yellow-500",
    },
    {
      icon: RotateCcw,
      label: t.aiops.stats.recurrenceRate,
      value: `${stats.recurrenceRate.toFixed(1)}%`,
      color: stats.recurrenceRate > 20 ? "bg-orange-500/10 text-orange-500" : "bg-emerald-500/10 text-emerald-500",
    },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
      {cards.map((card) => {
        const Icon = card.icon;
        return (
          <div key={card.label} className="bg-card rounded-xl border border-[var(--border-color)] p-3">
            <div className="flex items-center gap-2 mb-2">
              <div className={`p-1.5 rounded-lg ${card.color}`}>
                <Icon className="w-4 h-4" />
              </div>
              <span className="text-xs text-muted">{card.label}</span>
            </div>
            <div className="text-xl font-bold text-default">{card.value}</div>
          </div>
        );
      })}
    </div>
  );
}
