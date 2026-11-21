"use client";

import { DataTable, StatusBadge, type TableColumn } from "@/components/common";
import type { NotificationsTranslations } from "@/types/i18n";
import type { NotifyDelivery, NotifyDeliveryStatus } from "@/types/notify";
import { formatTimestamp } from "@/lib/format";
import { NOWRAP_CLASS } from "../shared/form-styles";

interface DeliveriesTableProps {
  deliveries: NotifyDelivery[];
  t: NotificationsTranslations;
}

const STATUS_BADGE_TYPE: Record<NotifyDeliveryStatus, "success" | "warning" | "error" | "info" | "default"> = {
  success: "success",
  pending: "info",
  failed: "warning",
  abandoned: "error",
  "no-receiver": "default",
};

/** 接收人的类型：slack / email 有译名，其他值（将来的新类型）原样显示 */
function receiverTypeLabel(receiverType: string, t: NotificationsTranslations): string {
  if (receiverType === "slack") return t.receivers.typeSlack;
  if (receiverType === "email") return t.receivers.typeEmail;
  return receiverType;
}

/** 当前页的送达记录：每条通知发给了谁、成没成功、失败的原因 */
export function DeliveriesTable({ deliveries, t }: DeliveriesTableProps) {
  const deliveriesT = t.deliveries;

  const columns: TableColumn<NotifyDelivery>[] = [
    {
      key: "alertTitle",
      header: deliveriesT.alertTitle,
      mobileTitle: true,
      render: (delivery) => <span className="text-sm text-default block min-w-[14rem]">{delivery.alertTitle}</span>,
    },
    {
      key: "receiver",
      header: deliveriesT.receiver,
      render: (delivery) =>
        delivery.receiverName ? (
          <div className={NOWRAP_CLASS}>
            <span className="text-sm text-default">{delivery.receiverName}</span>
            <span className="text-xs text-muted ml-1.5">{receiverTypeLabel(delivery.receiverType, t)}</span>
          </div>
        ) : (
          <span className="text-sm text-muted">-</span>
        ),
    },
    {
      key: "kind",
      header: deliveriesT.kind,
      render: (delivery) => (
        <span className={`text-sm text-default ${NOWRAP_CLASS}`}>{t.kind[delivery.kind] ?? delivery.kind}</span>
      ),
    },
    {
      key: "status",
      header: deliveriesT.status,
      render: (delivery) => (
        <span className={NOWRAP_CLASS}>
          <StatusBadge
            status={deliveriesT.statusLabel[delivery.status] ?? delivery.status}
            type={STATUS_BADGE_TYPE[delivery.status] ?? "default"}
          />
        </span>
      ),
    },
    {
      key: "attempts",
      header: deliveriesT.attempts,
      render: (delivery) => <span className="text-sm text-default tabular-nums">{delivery.attempts}</span>,
    },
    {
      key: "lastError",
      header: deliveriesT.lastError,
      render: (delivery) => (
        <span className="text-xs text-red-600 dark:text-red-400 break-all block max-w-xs" title={delivery.lastError}>
          {delivery.lastError || <span className="text-muted">-</span>}
        </span>
      ),
    },
    {
      key: "createdAt",
      header: deliveriesT.createdAt,
      render: (delivery) => (
        <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>{formatTimestamp(delivery.createdAt)}</span>
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={deliveries}
      loading={false}
      error=""
      keyExtractor={(delivery) => String(delivery.id)}
      // 分页由后端做：这里拿到的就是一页，表格自己不再分页
      pageSize={Math.max(deliveries.length, 1)}
      showPagination={false}
    />
  );
}
