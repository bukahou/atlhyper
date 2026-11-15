"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Plus, Users } from "lucide-react";
import { ConfirmDialog, QueryView, toast } from "@/components/common";
import { useI18n } from "@/i18n/context";
import { notifyFailureText } from "@/lib/notify-failure";
import { receiverSettingsOf, sealCredential } from "@/lib/notify-receiver-input";
import { MOCK_NOTIFY_RECEIVERS, MOCK_NOTIFY_SMTP } from "@/mock/notify";
import {
  deleteNotifyReceiverMutation,
  notifyReceiversQuery,
  saveNotifyReceiverMutation,
  testNotifyReceiverMutation,
} from "@/queries/notify-routing";
import { notifySMTPQuery } from "@/queries/notify-smtp";
import type { NotifyReceiver } from "@/types/notify";
import { EmptyCard } from "../shared/EmptyCard";
import type { NotifyTabProps } from "../shared/format";
import { CARD_CLASS, PRIMARY_BUTTON_CLASS } from "../shared/form-styles";
import { ReceiverModal } from "./ReceiverModal";
import { ReceiversTable } from "./ReceiversTable";
import { SmtpSettingsCard } from "./SmtpSettingsCard";

/** 弹窗的目标：新建，或编辑某个接收人 */
type ReceiverModalTarget = { mode: "create" } | { mode: "edit"; receiver: NotifyReceiver };

/** 「接收人」页签：顶部是邮件发件设置（全局 SMTP）；下面是 Slack 频道与邮件收件组的增删改、启停与测试发送 */
export function ReceiversTab({ isDemo }: NotifyTabProps) {
  const { t } = useI18n();
  const notifyT = t.notifications;
  const receiversT = notifyT.receivers;

  const receiversQuery = useQuery(notifyReceiversQuery(!isDemo));
  const smtpQuery = useQuery(notifySMTPQuery(!isDemo));
  // 只有确定「没配置」才提示；还在加载或取数失败时不下结论
  const smtp = isDemo ? MOCK_NOTIFY_SMTP : smtpQuery.data;
  const smtpMissing = smtp?.configured === false;
  // 写操作成功后工厂让列表重新取数，这里只管提示与关弹窗
  const toggleReceiver = useMutation(saveNotifyReceiverMutation());
  const testReceiver = useMutation(testNotifyReceiverMutation());
  const deleteReceiver = useMutation(deleteNotifyReceiverMutation());

  const [modalTarget, setModalTarget] = useState<ReceiverModalTarget | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<NotifyReceiver | null>(null);

  const reportFailure = (error: unknown) => toast.error(notifyFailureText(error, notifyT.errors));

  // 启停走修改接口：其余各项原样带上，凭据留空 = 保持原值（邮件接收人没有凭据）
  const handleToggleEnabled = (receiver: NotifyReceiver, enabled: boolean) => {
    toggleReceiver.mutate(
      {
        receiverId: receiver.id,
        settings: { ...receiverSettingsOf(receiver), enabled },
        takeCredential: sealCredential(""),
      },
      { onError: reportFailure },
    );
  };

  const handleTest = (receiver: NotifyReceiver) => {
    testReceiver.mutate(receiver.id, {
      onSuccess: () => toast.success(receiversT.testSent),
      onError: reportFailure,
    });
  };

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return;
    deleteReceiver.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(receiversT.deletedToast);
        setDeleteTarget(null);
      },
      // 仍被规则或默认策略引用时后端拒绝删除，并写明被谁引用
      onError: (error) => {
        setDeleteTarget(null);
        reportFailure(error);
      },
    });
  };

  const renderReceivers = (receivers: NotifyReceiver[]) =>
    receivers.length === 0 ? (
      <EmptyCard icon={Users} title={receiversT.empty} hint={receiversT.emptyHint} />
    ) : (
      <div className={CARD_CLASS}>
        <ReceiversTable
          receivers={receivers}
          smtpMissing={smtpMissing}
          readOnly={isDemo}
          testingReceiverId={testReceiver.isPending ? (testReceiver.variables ?? null) : null}
          toggling={toggleReceiver.isPending}
          onToggleEnabled={handleToggleEnabled}
          onTest={handleTest}
          onEdit={(receiver) => setModalTarget({ mode: "edit", receiver })}
          onDelete={setDeleteTarget}
          t={notifyT}
          editLabel={t.common.edit}
          deleteLabel={t.common.delete}
          actionLabel={t.common.action}
        />
      </div>
    );

  return (
    <div className="space-y-4">
      <SmtpSettingsCard
        smtpQuery={smtpQuery}
        demoSMTP={isDemo ? MOCK_NOTIFY_SMTP : null}
        t={notifyT}
        commonT={t.common}
      />

      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => setModalTarget({ mode: "create" })}
          // 演示模式下写操作不可用
          disabled={isDemo}
          className={PRIMARY_BUTTON_CLASS}
        >
          <Plus className="w-4 h-4" />
          {receiversT.add}
        </button>
      </div>

      {isDemo ? (
        renderReceivers(MOCK_NOTIFY_RECEIVERS)
      ) : (
        <QueryView query={receiversQuery}>{renderReceivers}</QueryView>
      )}

      {/* 新建 / 编辑弹窗：按目标整个重新挂载，表单初值直接取自目标 */}
      {modalTarget && (
        <ReceiverModal
          key={modalTarget.mode === "edit" ? `edit:${modalTarget.receiver.id}` : "create"}
          editing={modalTarget.mode === "edit" ? modalTarget.receiver : null}
          smtpMissing={smtpMissing}
          onClose={() => setModalTarget(null)}
          t={notifyT}
          commonT={t.common}
        />
      )}

      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        title={receiversT.deleteConfirmTitle}
        message={receiversT.deleteConfirmMessage.replace("{name}", deleteTarget?.name ?? "")}
        confirmText={t.common.delete}
        cancelText={t.common.cancel}
        loading={deleteReceiver.isPending}
        variant="danger"
      />
    </div>
  );
}
