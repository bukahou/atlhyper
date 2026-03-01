"use client";

import { useMemo } from "react";
import { useI18n } from "@/i18n/context";
import { StatsCard } from "@/components/common";
import type { EventLog } from "@/types/cluster";
import { isCriticalEvent } from "./EventTable";

interface EventStatsCardsProps {
  events: EventLog[];
}

export function EventStatsCards({ events }: EventStatsCardsProps) {
  const { t } = useI18n();

  const stats = useMemo(() => {
    let normal = 0;
    let warning = 0;
    let critical = 0;
    events.forEach((e) => {
      if (isCriticalEvent(e)) {
        critical++;
      } else if (e.severity === "Warning") {
        warning++;
      } else {
        normal++;
      }
    });
    return { total: events.length, normal, warning, critical };
  }, [events]);

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <StatsCard label={t.event.totalEvents} value={stats.total} />
      <StatsCard label={t.event.normalEvents} value={stats.normal} iconColor="text-green-500" />
      <StatsCard label={t.event.warningEvents} value={stats.warning} iconColor="text-yellow-500" />
      <StatsCard label={t.event.criticalEvents} value={stats.critical} iconColor="text-red-500" />
    </div>
  );
}
