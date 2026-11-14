"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { toast } from "@/components/common/Toast";
import type { AIProvider, RoleBudget } from "@/api/ai-provider";
import { useCan } from "@/hooks/useCan";
import { mockBudgets } from "@/mock/ai-settings";
import { aiBudgetsQuery, updateAIBudgetMutation } from "@/queries/ai-settings";
import { RoleBudgetCard, type BudgetEdits } from "./RoleBudgetCard";

const NO_EDITS: BudgetEdits = {};

interface BudgetConfigCardProps {
  /** 演示模式（没有 ai.settings.read）：显示演示数据，不发请求 */
  demo: boolean;
  /** 备用 Provider 的候选。由页面传进来：Provider 列表全页共用一份查询 */
  providers: AIProvider[];
}

export function BudgetConfigCard({ demo, providers }: BudgetConfigCardProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;
  const isAdmin = useCan()("ai.settings.write");
  const budgetsQuery = useQuery(aiBudgetsQuery(!demo));
  const budgets = demo ? mockBudgets : (budgetsQuery.data ?? []);
  const loading = !demo && budgetsQuery.isPending;
  // 保存成功后工厂让预算重新取数，取完才算结束
  const saveBudget = useMutation(updateAIBudgetMutation());
  const savingRole = saveBudget.isPending ? saveBudget.variables.role : null;

  // 各角色还没保存的修改
  const [edits, setEdits] = useState<Record<string, BudgetEdits>>({});

  const setEdit = (role: string, field: string, value: number | string | null) => {
    setEdits((prev) => ({
      ...prev,
      [role]: { ...prev[role], [field]: value },
    }));
  };

  const handleSave = (budget: RoleBudget) => {
    const edit = edits[budget.role] || {};
    saveBudget.mutate(
      {
        role: budget.role,
        changes: {
          dailyInputTokenLimit: edit.dailyInputTokenLimit as number | undefined,
          dailyOutputTokenLimit: edit.dailyOutputTokenLimit as number | undefined,
          dailyCallLimit: edit.dailyCallLimit as number | undefined,
          monthlyInputTokenLimit: edit.monthlyInputTokenLimit as number | undefined,
          monthlyOutputTokenLimit: edit.monthlyOutputTokenLimit as number | undefined,
          monthlyCallLimit: edit.monthlyCallLimit as number | undefined,
          autoTriggerMinSeverity: edit.autoTriggerMinSeverity as string | undefined,
          autoTriggerMode: edit.autoTriggerMode as string | undefined,
          scheduleStartTime: edit.scheduleStartTime as string | undefined,
          scheduleEndTime: edit.scheduleEndTime as string | undefined,
          fallbackProviderId:
            edit.fallbackProviderId !== undefined ? (edit.fallbackProviderId as number | null) : undefined,
        },
      },
      {
        onSuccess: () => {
          toast.success(aiT.budgetSaved);
          setEdits((prev) => {
            const next = { ...prev };
            delete next[budget.role];
            return next;
          });
        },
        onError: () => toast.error(aiT.budgetSaveFailed),
      },
    );
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <h3 className="text-lg font-medium text-default">{aiT.budgetConfig}</h3>
        <p className="text-sm text-muted mt-1">{aiT.budgetConfigDesc}</p>
      </div>
      <div className="p-6 space-y-4">
        {loading ? (
          <Loader2 className="w-5 h-5 animate-spin text-muted mx-auto" />
        ) : budgets.length === 0 ? (
          // 没有预算数据时卡片照常显示（不隐藏整张卡片）
          <p className="text-center text-sm text-muted py-4">{t.common.noData}</p>
        ) : (
          budgets.map((budget) => (
            <RoleBudgetCard
              key={budget.role}
              budget={budget}
              edits={edits[budget.role] ?? NO_EDITS}
              onEdit={(field, value) => setEdit(budget.role, field, value)}
              providers={providers}
              isAdmin={isAdmin}
              saving={savingRole === budget.role}
              onSave={() => handleSave(budget)}
            />
          ))
        )}
      </div>
    </div>
  );
}
