"use client";

import { Drawer, StatusBadge } from "@/components/common";
import { useI18n } from "@/i18n/context";
import type { EventLog } from "@/types/cluster";
import { Clock, Hash, Server, Box, AlertTriangle } from "lucide-react";

interface EventDetailDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  event: EventLog;
}

function isCriticalEvent(event: EventLog): boolean {
  const criticalReasons = [
    "OOMKilling", "CrashLoopBackOff", "FailedScheduling",
    "FailedMount", "NodeNotReady", "FailedBinding",
  ];
  return event.severity === "Warning" && criticalReasons.includes(event.reason);
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3 py-2">
      <span className="text-sm text-muted shrink-0 w-28">{label}</span>
      <span className="text-sm text-default break-all">{value || "-"}</span>
    </div>
  );
}

function formatTime(ts?: string): string {
  if (!ts) return "-";
  return new Date(ts).toLocaleString();
}

export function EventDetailDrawer({ isOpen, onClose, event }: EventDetailDrawerProps) {
  const { t } = useI18n();
  const severityLabel = isCriticalEvent(event) ? "Critical" : event.severity === "Warning" ? "Warning" : "Normal";

  return (
    <Drawer isOpen={isOpen} onClose={onClose} title={`${event.kind}: ${event.name}`} size="lg">
      <div className="p-6 space-y-6">
        {/* Basic Info */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Box className="w-4 h-4 text-primary" />
            <h3 className="text-sm font-semibold text-default">{t.event.basicInfo}</h3>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 divide-y divide-[var(--border-color)]">
            <InfoRow label={t.common.name} value={event.name} />
            <InfoRow label={t.common.type} value={event.kind} />
            <InfoRow label={t.common.namespace} value={event.namespace || "-"} />
            <InfoRow
              label={t.common.status}
              value={<StatusBadge status={severityLabel} />}
            />
            <InfoRow label={t.event.reason} value={
              <span className="font-mono text-xs px-2 py-0.5 rounded bg-primary/10 text-primary">
                {event.reason}
              </span>
            } />
            {event.node && <InfoRow label="Node" value={event.node} />}
          </div>
        </section>

        {/* Event Details */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <AlertTriangle className="w-4 h-4 text-yellow-500" />
            <h3 className="text-sm font-semibold text-default">{t.event.eventInfo}</h3>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 space-y-4">
            {/* Message */}
            <div>
              <span className="text-xs text-muted uppercase tracking-wider">{t.alert.message}</span>
              <p className="mt-1 text-sm text-default leading-relaxed bg-card rounded-lg p-3 border border-[var(--border-color)]">
                {event.message}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              {/* Source */}
              <div className="flex items-center gap-2">
                <Server className="w-3.5 h-3.5 text-muted" />
                <div>
                  <span className="text-xs text-muted block">{t.event.source}</span>
                  <span className="text-sm text-default">{event.source || "-"}</span>
                </div>
              </div>

              {/* Count */}
              <div className="flex items-center gap-2">
                <Hash className="w-3.5 h-3.5 text-muted" />
                <div>
                  <span className="text-xs text-muted block">{t.event.count}</span>
                  <span className="text-sm text-default font-mono">{event.count ?? 1}</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Timestamps */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Clock className="w-4 h-4 text-blue-500" />
            <h3 className="text-sm font-semibold text-default">{t.common.time}</h3>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 divide-y divide-[var(--border-color)]">
            <InfoRow label={t.common.time} value={formatTime(event.eventTime || event.time)} />
            <InfoRow label={t.event.firstTime} value={formatTime(event.firstTimestamp)} />
            <InfoRow label={t.event.lastTime} value={formatTime(event.lastTimestamp)} />
          </div>
        </section>

        {/* Related Resource */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Box className="w-4 h-4 text-purple-500" />
            <h3 className="text-sm font-semibold text-default">{t.event.relatedResource}</h3>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 divide-y divide-[var(--border-color)]">
            <InfoRow label="Kind" value={event.kind} />
            <InfoRow label={t.common.namespace} value={event.namespace || "-"} />
            <InfoRow label={t.common.name} value={event.name} />
            {event.category && <InfoRow label="Category" value={event.category} />}
          </div>
        </section>
      </div>
    </Drawer>
  );
}
