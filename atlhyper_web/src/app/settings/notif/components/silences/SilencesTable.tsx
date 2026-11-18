"use client";

import { BellOff } from "lucide-react";
import { DataTable, type TableColumn } from "@/components/common";
import type { NotificationsTranslations } from "@/types/i18n";
import type { NotifySilence } from "@/types/notify";
import { formatTimestamp } from "../shared/format";
import { NOWRAP_CLASS } from "../shared/form-styles";

interface SilencesTableProps {
  silences: NotifySilence[];
  /** 演示数据：操作按钮照常显示但不可用 */
  readOnly: boolean;
  onExpire: (silence: NotifySilence) => void;
  t: NotificationsTranslations;
  actionLabel: string;
}

/** 静默的范围：三个条件各占一行，留空的显示为「全部」 */
function SilenceScope({ silence, t }: { silence: NotifySilence; t: NotificationsTranslations["silences"] }) {
  const rows = [
    { label: t.cluster, value: silence.clusterId, all: t.allClusters },
    { label: t.namespace, value: silence.namespace, all: t.allNamespaces },
    { label: t.subject, value: silence.subject, all: t.allSubjects },
  ];
  return (
    <div className="space-y-0.5 min-w-[12rem]">
      {rows.map((row) => (
        <div key={row.label} className="text-xs">
          <span className="text-muted">{row.label}: </span>
          {row.value ? (
            <span className="font-mono text-default break-all">{row.value}</span>
          ) : (
            <span className="text-muted">{row.all}</span>
          )}
        </div>
      ))}
    </div>
  );
}

/** 某个状态下的静默。静默不删除，只会过期；生效中与未开始的可以提前结束 */
export function SilencesTable({ silences, readOnly, onExpire, t, actionLabel }: SilencesTableProps) {
  const silencesT = t.silences;

  const columns: TableColumn<NotifySilence>[] = [
    {
      key: "scope",
      header: silencesT.scope,
      mobileTitle: true,
      render: (silence) => <SilenceScope silence={silence} t={silencesT} />,
    },
    {
      key: "startsAt",
      header: silencesT.startsAt,
      render: (silence) => (
        <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>{formatTimestamp(silence.startsAt)}</span>
      ),
    },
    {
      key: "endsAt",
      header: silencesT.endsAt,
      render: (silence) => <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>{formatTimestamp(silence.endsAt)}</span>,
    },
    {
      key: "comment",
      header: silencesT.comment,
      render: (silence) => <span className="text-sm text-default block max-w-xs break-words">{silence.comment}</span>,
    },
    {
      key: "createdBy",
      header: silencesT.createdBy,
      mobileVisible: false,
      render: (silence) => <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>{silence.createdBy}</span>,
    },
    {
      key: "actions",
      header: actionLabel,
      render: (silence) =>
        silence.state === "expired" ? (
          <span className="text-sm text-muted">-</span>
        ) : (
          <button
            type="button"
            onClick={() => onExpire(silence)}
            disabled={readOnly}
            className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-lg border border-[var(--border-color)] text-default hover-bg whitespace-nowrap disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <BellOff className="w-3.5 h-3.5" />
            {silencesT.expire}
          </button>
        ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={silences}
      loading={false}
      error=""
      keyExtractor={(silence) => String(silence.id)}
      pageSize={20}
    />
  );
}
