"use client";

import { AlertTriangle, Loader2, Pencil, Send, Trash2 } from "lucide-react";
import { DataTable, StatusBadge, type TableColumn } from "@/components/common";
import type { NotificationsTranslations } from "@/types/i18n";
import type { NotifyReceiver } from "@/types/notify";
import { formatTimestamp } from "@/lib/format";
import { ICON_BUTTON_CLASS, NOWRAP_CLASS } from "../shared/form-styles";
import { Switch } from "../shared/Switch";

interface ReceiversTableProps {
  receivers: NotifyReceiver[];
  /** 全局发件设置还没配置：邮件接收人那几行标出「发不出去」 */
  smtpMissing: boolean;
  /** 演示数据：开关与操作按钮照常显示但不可用 */
  readOnly: boolean;
  /** 正在发测试消息的接收人；null = 没有 */
  testingReceiverId: number | null;
  /** 启用开关正在保存 */
  toggling: boolean;
  onToggleEnabled: (receiver: NotifyReceiver, enabled: boolean) => void;
  onTest: (receiver: NotifyReceiver) => void;
  onEdit: (receiver: NotifyReceiver) => void;
  onDelete: (receiver: NotifyReceiver) => void;
  t: NotificationsTranslations;
  editLabel: string;
  deleteLabel: string;
  actionLabel: string;
}

/**
 * 发到哪里。Slack 只有「Webhook 是否已配置」，地址本身接口不返回；
 * 邮件只显示收件人（发件服务器与账号是全局的发件设置，不属于某个接收人）。
 */
function ReceiverTarget({
  receiver,
  smtpMissing,
  t,
}: {
  receiver: NotifyReceiver;
  smtpMissing: boolean;
  t: NotificationsTranslations["receivers"];
}) {
  const warning = (text: string) => (
    <span className="inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
      <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
      {text}
    </span>
  );

  if (receiver.type === "slack") {
    return receiver.slack?.webhookConfigured ? (
      <span className="text-sm text-default">{t.webhookConfigured}</span>
    ) : (
      warning(t.webhookNotConfigured)
    );
  }
  const toAddresses = receiver.email?.toAddresses ?? [];
  return (
    <div className="space-y-0.5">
      <div className="text-sm text-default break-all">{toAddresses.length > 0 ? toAddresses.join(", ") : "-"}</div>
      <div className="text-xs text-muted">{t.recipientsCount.replace("{count}", String(toAddresses.length))}</div>
      {smtpMissing && warning(t.smtpNotConfigured)}
    </div>
  );
}

/** 接收人列表：一行一个接收人，可启停、测试发送、编辑、删除 */
export function ReceiversTable({
  receivers,
  smtpMissing,
  readOnly,
  testingReceiverId,
  toggling,
  onToggleEnabled,
  onTest,
  onEdit,
  onDelete,
  t,
  editLabel,
  deleteLabel,
  actionLabel,
}: ReceiversTableProps) {
  const receiversT = t.receivers;

  const columns: TableColumn<NotifyReceiver>[] = [
    {
      key: "name",
      header: receiversT.name,
      mobileTitle: true,
      render: (receiver) => <span className="text-sm font-medium text-default">{receiver.name}</span>,
    },
    {
      key: "type",
      header: receiversT.type,
      render: (receiver) => (
        <span className={NOWRAP_CLASS}>
          <StatusBadge
            status={receiver.type === "slack" ? receiversT.typeSlack : receiversT.typeEmail}
            type={receiver.type === "slack" ? "info" : "default"}
          />
        </span>
      ),
    },
    {
      key: "target",
      header: receiversT.target,
      render: (receiver) => <ReceiverTarget receiver={receiver} smtpMissing={smtpMissing} t={receiversT} />,
    },
    {
      key: "enabled",
      header: receiversT.enabled,
      render: (receiver) => (
        <Switch
          checked={receiver.enabled}
          onChange={(enabled) => onToggleEnabled(receiver, enabled)}
          disabled={readOnly || toggling}
          label={receiver.enabled ? t.enabledLabel : t.disabledLabel}
        />
      ),
    },
    {
      key: "updatedAt",
      header: receiversT.updatedAt,
      mobileVisible: false,
      render: (receiver) => (
        <span className={`text-sm text-muted ${NOWRAP_CLASS}`}>{formatTimestamp(receiver.updatedAt)}</span>
      ),
    },
    {
      key: "actions",
      header: actionLabel,
      render: (receiver) => (
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => onTest(receiver)}
            disabled={readOnly || testingReceiverId !== null}
            className={ICON_BUTTON_CLASS}
            title={receiversT.test}
            aria-label={receiversT.test}
          >
            {testingReceiverId === receiver.id ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Send className="w-4 h-4" />
            )}
          </button>
          <button
            type="button"
            onClick={() => onEdit(receiver)}
            disabled={readOnly}
            className={ICON_BUTTON_CLASS}
            title={editLabel}
            aria-label={editLabel}
          >
            <Pencil className="w-4 h-4" />
          </button>
          <button
            type="button"
            onClick={() => onDelete(receiver)}
            disabled={readOnly}
            className={`${ICON_BUTTON_CLASS} hover:text-red-500`}
            title={deleteLabel}
            aria-label={deleteLabel}
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={receivers}
      loading={false}
      error=""
      keyExtractor={(receiver) => String(receiver.id)}
      pageSize={20}
    />
  );
}
