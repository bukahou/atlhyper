"use client";

import { AlertTriangle, ArrowRight, TrendingUp, Target, CheckCircle, RotateCcw } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { EntityLink } from "@/components/aiops/EntityLink";
import type { IncidentTimeline } from "@/api/aiops";

const EVENT_ICONS: Record<string, { icon: typeof AlertTriangle; color: string }> = {
  anomaly_detected: { icon: AlertTriangle, color: "text-yellow-500 bg-yellow-500/10" },
  state_change: { icon: ArrowRight, color: "text-blue-500 bg-blue-500/10" },
  metric_spike: { icon: TrendingUp, color: "text-red-500 bg-red-500/10" },
  root_cause_identified: { icon: Target, color: "text-purple-500 bg-purple-500/10" },
  recovery_started: { icon: CheckCircle, color: "text-emerald-500 bg-emerald-500/10" },
  recurrence: { icon: RotateCcw, color: "text-orange-500 bg-orange-500/10" },
};

interface TimelineViewProps {
  timeline: IncidentTimeline[];
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  return `${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}`;
}

export function TimelineView({ timeline }: TimelineViewProps) {
  const { t } = useI18n();

  if (timeline.length === 0) {
    return <div className="text-sm text-muted py-4 text-center">{t.aiops.noData}</div>;
  }

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-3">{t.aiops.timeline}</h4>
      <div className="relative">
        {/* 垂直线 */}
        <div className="absolute left-[68px] top-0 bottom-0 w-px bg-[var(--border-color)]" />

        <div className="space-y-0">
          {timeline.map((event) => {
            const eventConfig = EVENT_ICONS[event.eventType] ?? EVENT_ICONS.anomaly_detected;
            const Icon = eventConfig.icon;
            const eventLabel =
              t.aiops.timelineEvent[event.eventType as keyof typeof t.aiops.timelineEvent] ?? event.eventType;

            return (
              <div key={event.id} className="flex items-start gap-3 py-2">
                {/* 时间 */}
                <span className="text-xs text-muted font-mono w-12 text-right flex-shrink-0 pt-0.5">
                  {formatTimestamp(event.timestamp)}
                </span>

                {/* 图标 */}
                <div className={`w-7 h-7 rounded-full flex items-center justify-center flex-shrink-0 z-10 ${eventConfig.color}`}>
                  <Icon className="w-3.5 h-3.5" />
                </div>

                {/* 内容 */}
                <div className="flex-1 min-w-0 pt-0.5">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-default">{eventLabel}</span>
                    <EntityLink entityKey={event.entityKey} showType={false} />
                  </div>
                  <p className="text-xs text-muted mt-0.5">{event.detail}</p>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
