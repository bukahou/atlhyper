"use client";

import { useI18n } from "@/i18n/context";
import { StatsCard } from "@/components/common";
import type { EventCards } from "@/types/cluster";

interface EventStatsCardsProps {
  /** 后端随事件列表一起给出的统计 */
  cards: EventCards;
}

export function EventStatsCards({ cards }: EventStatsCardsProps) {
  const { t } = useI18n();

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <StatsCard label={t.event.totalEvents} value={cards.totalEvents} />
      <StatsCard label={t.event.normalEvents} value={cards.info} iconColor="text-green-500" />
      <StatsCard label={t.event.warningEvents} value={cards.warning} iconColor="text-yellow-500" />
      <StatsCard label={t.event.criticalEvents} value={cards.critical + cards.error} iconColor="text-red-500" />
    </div>
  );
}
