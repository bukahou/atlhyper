"use client";

import { useMemo } from "react";
import { useI18n } from "@/i18n/context";
import { DataTable, StatusBadge, type TableColumn } from "@/components/common";
import type { EventLog } from "@/types/cluster";

function getSeverityLabel(severity: string): string {
  if (severity === "Warning") {
    return "Warning";
  }
  if (severity === "Critical" || severity === "Error") {
    return "Critical";
  }
  return "Normal";
}

export function isCriticalEvent(event: EventLog): boolean {
  const criticalReasons = [
    "OOMKilling", "CrashLoopBackOff", "FailedScheduling",
    "FailedMount", "NodeNotReady", "FailedBinding",
  ];
  return event.severity === "Warning" && criticalReasons.includes(event.reason);
}

interface EventTableProps {
  events: EventLog[];
  loading: boolean;
  error: string;
  onRowClick: (event: EventLog) => void;
}

export function EventTable({ events, loading, error, onRowClick }: EventTableProps) {
  const { t } = useI18n();

  const columns: TableColumn<EventLog>[] = [
    {
      key: "time",
      header: t.common.time,
      render: (e) => {
        const d = new Date(e.eventTime || e.time);
        return (
          <span className="text-xs text-muted whitespace-nowrap font-mono">
            {d.toLocaleString()}
          </span>
        );
      },
    },
    {
      key: "severity",
      header: t.common.status,
      render: (e) => {
        const label = isCriticalEvent(e) ? "Critical" : getSeverityLabel(e.severity);
        return <StatusBadge status={label} />;
      },
    },
    {
      key: "kind",
      header: t.common.type,
      render: (e) => <span className="text-sm font-medium">{e.kind}</span>,
    },
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (e) => (
        <div className="max-w-[200px]">
          <span className="font-medium text-default truncate block">{e.name}</span>
        </div>
      ),
    },
    {
      key: "namespace",
      header: t.common.namespace,
      mobileVisible: false,
      render: (e) => <span className="text-sm">{e.namespace || "-"}</span>,
    },
    {
      key: "reason",
      header: t.event.reason,
      render: (e) => <span className="text-sm">{e.reason}</span>,
    },
    {
      key: "source",
      header: t.event.source,
      mobileVisible: false,
      render: (e) => <span className="text-xs text-muted">{e.source || "-"}</span>,
    },
    {
      key: "message",
      header: t.alert.message,
      mobileVisible: false,
      render: (e) => (
        <div className="max-w-[300px]">
          <span className="text-sm text-muted truncate block">{e.message}</span>
        </div>
      ),
    },
    {
      key: "count",
      header: t.event.count,
      mobileVisible: false,
      render: (e) => <span className="text-sm font-mono">{e.count ?? 1}</span>,
    },
  ];

  const sortedEvents = useMemo(() => {
    return [...events].sort((a, b) => {
      const ta = new Date(a.eventTime || a.time).getTime();
      const tb = new Date(b.eventTime || b.time).getTime();
      return tb - ta;
    });
  }, [events]);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <DataTable
        columns={columns}
        data={sortedEvents}
        loading={loading}
        error={error}
        keyExtractor={(e, i) => `${i}-${e.kind}/${e.namespace}/${e.name}/${e.eventTime}`}
        onRowClick={onRowClick}
        pageSize={15}
      />
    </div>
  );
}
