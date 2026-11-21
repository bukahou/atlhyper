"use client";

import Link from "next/link";
import { BellOff, ExternalLink, ShieldOff } from "lucide-react";
import { DataTable, type TableColumn } from "@/components/common";
import { incidentLink } from "@/lib/signal-link";
import type { NotificationsTranslations } from "@/types/i18n";
import type { NotifyAlert } from "@/types/notify";
import { formatTimestamp } from "@/lib/format";
import { NOWRAP_CLASS } from "../shared/form-styles";
import { SeverityBadge } from "../shared/SeverityBadge";

interface AlertsTableProps {
  alerts: NotifyAlert[];
  t: NotificationsTranslations;
}

const FLAG_CLASS = "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium whitespace-nowrap";

/** 当前页的告警。被抑制、被静默的告警照常列出，用醒目的标记说明它为什么没有发出 */
export function AlertsTable({ alerts, t }: AlertsTableProps) {
  const alertsT = t.alerts;

  const suppressedReason = (suppressedBy: string) =>
    suppressedBy === "cluster-offline"
      ? alertsT.suppressedByClusterOffline
      : alertsT.suppressedByOther.replace("{reason}", suppressedBy);

  const columns: TableColumn<NotifyAlert>[] = [
    {
      key: "severity",
      header: alertsT.severity,
      render: (alert) => <SeverityBadge severity={alert.severity} labels={t.severity} />,
    },
    {
      key: "kind",
      header: alertsT.kind,
      render: (alert) => <span className={`text-sm text-default ${NOWRAP_CLASS}`}>{t.kind[alert.kind] ?? alert.kind}</span>,
    },
    {
      key: "title",
      header: alertsT.title,
      mobileTitle: true,
      render: (alert) => (
        <div className="min-w-[14rem] space-y-1">
          <div className="text-sm text-default">{alert.title}</div>
          {alert.summary && <div className="text-xs text-muted">{alert.summary}</div>}
          <div className="flex flex-wrap items-center gap-1.5">
            {alert.suppressedBy !== "" && (
              <span
                className={`${FLAG_CLASS} bg-purple-500/15 text-purple-600 dark:text-purple-400`}
                title={suppressedReason(alert.suppressedBy)}
              >
                <ShieldOff className="w-3 h-3" />
                {alertsT.suppressed}
              </span>
            )}
            {alert.silencedBy > 0 && (
              <span
                className={`${FLAG_CLASS} bg-gray-500/15 text-gray-600 dark:text-gray-300`}
                title={alertsT.silencedHint.replace("{id}", String(alert.silencedBy))}
              >
                <BellOff className="w-3 h-3" />
                {alertsT.silenced}
              </span>
            )}
            {alert.incidentId !== "" && (
              <Link
                href={incidentLink(alert.incidentId)}
                className="inline-flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400 hover:underline"
              >
                <ExternalLink className="w-3 h-3" />
                {alertsT.viewIncident}
              </Link>
            )}
          </div>
        </div>
      ),
    },
    {
      key: "subject",
      header: alertsT.subject,
      render: (alert) => <span className="font-mono text-xs text-default break-all">{alert.subject || "-"}</span>,
    },
    {
      key: "clusterId",
      header: alertsT.cluster,
      render: (alert) => <span className={`font-mono text-xs text-default ${NOWRAP_CLASS}`}>{alert.clusterId}</span>,
    },
    {
      key: "startsAt",
      header: alertsT.startsAt,
      mobileVisible: false,
      render: (alert) => <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>{formatTimestamp(alert.startsAt)}</span>,
    },
    {
      key: "endsAt",
      header: alertsT.endsAt,
      mobileVisible: false,
      render: (alert) => <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>{formatTimestamp(alert.endsAt)}</span>,
    },
    {
      key: "notifyCount",
      header: alertsT.notifyCount,
      render: (alert) => <span className="text-sm text-default tabular-nums">{alert.notifyCount}</span>,
    },
    {
      key: "lastNotifiedAt",
      header: alertsT.lastNotifiedAt,
      mobileVisible: false,
      render: (alert) => (
        <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>
          {alert.lastNotifiedAt ? formatTimestamp(alert.lastNotifiedAt) : alertsT.neverNotified}
        </span>
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={alerts}
      loading={false}
      error=""
      keyExtractor={(alert) => alert.fingerprint}
      // 分页由后端做：这里拿到的就是一页，表格自己不再分页
      pageSize={Math.max(alerts.length, 1)}
      showPagination={false}
    />
  );
}
