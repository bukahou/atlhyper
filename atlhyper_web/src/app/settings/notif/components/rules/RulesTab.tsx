"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { GitBranch, Plus } from "lucide-react";
import { ConfirmDialog, QueryView, toast } from "@/components/common";
import { useI18n } from "@/i18n/context";
import { notifyFailureText } from "@/lib/notify-failure";
import { movedRuleIds, type RuleMoveDirection } from "@/lib/notify-rule-input";
import { MOCK_NOTIFY_RECEIVERS, MOCK_NOTIFY_RULES } from "@/mock/notify";
import {
  deleteNotifyRuleMutation,
  notifyReceiversQuery,
  notifyRulesQuery,
  reorderNotifyRulesMutation,
} from "@/queries/notify-routing";
import type { NotifyReceiver, NotifyRule } from "@/types/notify";
import { EmptyCard } from "../shared/EmptyCard";
import type { NotifyTabProps } from "../shared/format";
import { CARD_CLASS, PRIMARY_BUTTON_CLASS } from "../shared/form-styles";
import { useNotifyClusterIds } from "../shared/useNotifyClusterIds";
import { RuleMatchPanel } from "./RuleMatchPanel";
import { RuleModal } from "./RuleModal";
import { RulesTable } from "./RulesTable";

/** 弹窗的目标：新建，或编辑某条规则 */
type RuleModalTarget = { mode: "create" } | { mode: "edit"; rule: NotifyRule };

const NO_RECEIVERS: NotifyReceiver[] = [];

/** 「路由规则」页签：按顺序匹配的规则、调序、试匹配 */
export function RulesTab({ isDemo }: NotifyTabProps) {
  const { t } = useI18n();
  const notifyT = t.notifications;
  const rulesT = notifyT.rules;
  const clusterIds = useNotifyClusterIds(isDemo);

  const rulesQuery = useQuery(notifyRulesQuery(!isDemo));
  // 接收人只用来把 id 显示成名称、给规则表单提供选项；取不到时规则照常显示（名称退回成 #id）
  const receiversQuery = useQuery(notifyReceiversQuery(!isDemo));
  const receivers = isDemo ? MOCK_NOTIFY_RECEIVERS : (receiversQuery.data ?? NO_RECEIVERS);

  // 写操作成功后工厂让列表重新取数，这里只管提示与关弹窗
  const reorderRules = useMutation(reorderNotifyRulesMutation());
  const deleteRule = useMutation(deleteNotifyRuleMutation());

  const [modalTarget, setModalTarget] = useState<RuleModalTarget | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<NotifyRule | null>(null);

  const reportFailure = (error: unknown) => toast.error(notifyFailureText(error, notifyT.errors));

  const handleMove = (rules: NotifyRule[], rule: NotifyRule, direction: RuleMoveDirection) => {
    const reordered = movedRuleIds(
      rules.map((listed) => listed.id),
      rule.id,
      direction,
    );
    if (reordered) reorderRules.mutate(reordered, { onError: reportFailure });
  };

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return;
    deleteRule.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(rulesT.deletedToast);
        setDeleteTarget(null);
      },
      onError: (error) => {
        setDeleteTarget(null);
        reportFailure(error);
      },
    });
  };

  const renderRules = (rules: NotifyRule[]) =>
    rules.length === 0 ? (
      <EmptyCard icon={GitBranch} title={rulesT.empty} hint={rulesT.emptyHint} />
    ) : (
      <div className={CARD_CLASS}>
        <RulesTable
          rules={rules}
          receivers={receivers}
          readOnly={isDemo}
          reordering={reorderRules.isPending}
          onMove={(rule, direction) => handleMove(rules, rule, direction)}
          onEdit={(rule) => setModalTarget({ mode: "edit", rule })}
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
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <p className="text-sm text-muted">{rulesT.orderHint}</p>
        <button
          type="button"
          onClick={() => setModalTarget({ mode: "create" })}
          // 演示模式下写操作不可用
          disabled={isDemo}
          className={`${PRIMARY_BUTTON_CLASS} flex-shrink-0 self-end sm:self-auto`}
        >
          <Plus className="w-4 h-4" />
          {rulesT.add}
        </button>
      </div>

      {isDemo ? renderRules(MOCK_NOTIFY_RULES) : <QueryView query={rulesQuery}>{renderRules}</QueryView>}

      <RuleMatchPanel clusterIds={clusterIds} readOnly={isDemo} t={notifyT} />

      {/* 新建 / 编辑弹窗：按目标整个重新挂载，表单初值直接取自目标 */}
      {modalTarget && (
        <RuleModal
          key={modalTarget.mode === "edit" ? `edit:${modalTarget.rule.id}` : "create"}
          editing={modalTarget.mode === "edit" ? modalTarget.rule : null}
          receivers={receivers}
          clusterIds={clusterIds}
          onClose={() => setModalTarget(null)}
          t={notifyT}
          commonT={t.common}
        />
      )}

      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        title={rulesT.deleteConfirmTitle}
        message={rulesT.deleteConfirmMessage.replace("{name}", deleteTarget?.name ?? "")}
        confirmText={t.common.delete}
        cancelText={t.common.cancel}
        loading={deleteRule.isPending}
        variant="danger"
      />
    </div>
  );
}
