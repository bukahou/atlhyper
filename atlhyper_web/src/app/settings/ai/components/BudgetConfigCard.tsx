"use client";

import { useEffect, useState } from "react";
import { Loader2, Save } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { UserRole } from "@/types/auth";
import { toast } from "@/components/common/Toast";
import { getBudgets, updateBudget, listProviders, type RoleBudget, type AIProvider } from "@/api/ai-provider";

const ROLE_STYLES: Record<string, string> = {
  background: "border-blue-200 dark:border-blue-800",
  chat: "border-green-200 dark:border-green-800",
  analysis: "border-purple-200 dark:border-purple-800",
};

const ROLE_LABELS: Record<string, string> = {
  background: "roleBackground",
  chat: "roleChat",
  analysis: "roleAnalysis",
};

const SEVERITY_OPTIONS = ["critical", "high", "medium", "low", "off"] as const;

// Token 进度条
function TokenProgress({ used, limit, label }: { used: number; limit: number; label: string }) {
  const pct = limit > 0 ? Math.min(100, (used / limit) * 100) : 0;
  const barColor = pct > 90 ? "bg-red-500" : pct > 70 ? "bg-yellow-500" : "bg-emerald-500";

  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs text-muted">
        <span>{label}</span>
        <span>
          {used.toLocaleString()} / {limit > 0 ? limit.toLocaleString() : "∞"}
        </span>
      </div>
      {limit > 0 && (
        <div className="h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
          <div className={`h-full rounded-full transition-all ${barColor}`} style={{ width: `${pct}%` }} />
        </div>
      )}
    </div>
  );
}

export function BudgetConfigCard() {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;
  const { user, isAuthenticated } = useAuthStore();
  const isAdmin = user?.role === UserRole.ADMIN;
  const [budgets, setBudgets] = useState<RoleBudget[]>([]);
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingRole, setSavingRole] = useState<string | null>(null);

  // Editable state per role
  const [edits, setEdits] = useState<Record<string, Record<string, number | string | null>>>({});

  useEffect(() => {
    if (!isAuthenticated) {
      setLoading(false);
      return;
    }
    Promise.all([
      getBudgets().then((res) => setBudgets(res.data.data)).catch(() => {}),
      listProviders().then((res) => setProviders(res.data.providers)).catch(() => {}),
    ]).finally(() => setLoading(false));
  }, [isAuthenticated]);

  const getEdit = (role: string, field: string) => edits[role]?.[field];

  const setEdit = (role: string, field: string, value: number | string | null) => {
    setEdits((prev) => ({
      ...prev,
      [role]: { ...prev[role], [field]: value },
    }));
  };

  const handleSave = async (budget: RoleBudget) => {
    setSavingRole(budget.role);
    const edit = edits[budget.role] || {};
    try {
      await updateBudget(budget.role, {
        dailyInputTokenLimit: edit.dailyInputTokenLimit as number | undefined,
        dailyOutputTokenLimit: edit.dailyOutputTokenLimit as number | undefined,
        dailyCallLimit: edit.dailyCallLimit as number | undefined,
        monthlyInputTokenLimit: edit.monthlyInputTokenLimit as number | undefined,
        monthlyOutputTokenLimit: edit.monthlyOutputTokenLimit as number | undefined,
        monthlyCallLimit: edit.monthlyCallLimit as number | undefined,
        autoTriggerMinSeverity: edit.autoTriggerMinSeverity as string | undefined,
        fallbackProviderId: edit.fallbackProviderId !== undefined
          ? (edit.fallbackProviderId as number | null)
          : undefined,
      });
      toast.success(aiT.budgetSaved);
      const res = await getBudgets();
      setBudgets(res.data.data);
      setEdits((prev) => {
        const next = { ...prev };
        delete next[budget.role];
        return next;
      });
    } catch {
      toast.error(aiT.budgetSaveFailed);
    } finally {
      setSavingRole(null);
    }
  };

  if (loading) {
    return (
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
        <Loader2 className="w-5 h-5 animate-spin text-muted mx-auto" />
      </div>
    );
  }

  if (budgets.length === 0) return null;

  const LimitInput = ({ role, field, budget }: { role: string; field: string; budget: RoleBudget }) => {
    const val = (getEdit(role, field) as number) ?? (budget as unknown as Record<string, number>)[field];
    return (
      <input
        type="number"
        value={val}
        onChange={(e) => setEdit(role, field, parseInt(e.target.value) || 0)}
        disabled={!isAdmin}
        className="w-full px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-default disabled:opacity-50"
      />
    );
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <h3 className="text-lg font-medium text-default">{aiT.budgetConfig}</h3>
        <p className="text-sm text-muted mt-1">{aiT.budgetConfigDesc}</p>
      </div>
      <div className="p-6 space-y-4">
        {budgets.map((budget) => {
          const roleLabelKey = ROLE_LABELS[budget.role] as keyof typeof aiT | undefined;
          const roleLabel = (roleLabelKey ? aiT[roleLabelKey] : budget.role) as string;
          const severity = (getEdit(budget.role, "autoTriggerMinSeverity") as string) ?? budget.autoTriggerMinSeverity;
          const fallbackId = (getEdit(budget.role, "fallbackProviderId") as number | null | undefined) ?? budget.fallbackProviderId;
          const hasChanges = edits[budget.role] && Object.keys(edits[budget.role]).length > 0;

          return (
            <div
              key={budget.role}
              className={`rounded-lg border p-4 ${ROLE_STYLES[budget.role] || "border-[var(--border-color)]"}`}
            >
              <div className="flex items-center justify-between mb-3">
                <h4 className="font-medium text-default">{roleLabel}</h4>
                {isAdmin && (
                  <button
                    onClick={() => handleSave(budget)}
                    disabled={!hasChanges || savingRole === budget.role}
                    className="flex items-center gap-1.5 px-3 py-1 text-xs rounded-lg bg-violet-600 text-white hover:bg-violet-700 disabled:opacity-50 transition-colors"
                  >
                    {savingRole === budget.role ? (
                      <Loader2 className="w-3 h-3 animate-spin" />
                    ) : (
                      <Save className="w-3 h-3" />
                    )}
                    {aiT.save}
                  </button>
                )}
              </div>

              {/* 使用状况 (进度条) */}
              <div className="grid gap-2 md:grid-cols-3 mb-4">
                <TokenProgress
                  label={aiT.dailyInputTokens}
                  used={budget.dailyInputTokensUsed}
                  limit={budget.dailyInputTokenLimit}
                />
                <TokenProgress
                  label={aiT.dailyOutputTokens}
                  used={budget.dailyOutputTokensUsed}
                  limit={budget.dailyOutputTokenLimit}
                />
                <TokenProgress
                  label={aiT.dailyCalls}
                  used={budget.dailyCallsUsed}
                  limit={budget.dailyCallLimit}
                />
                <TokenProgress
                  label={aiT.monthlyInputTokens}
                  used={budget.monthlyInputTokensUsed}
                  limit={budget.monthlyInputTokenLimit}
                />
                <TokenProgress
                  label={aiT.monthlyOutputTokens}
                  used={budget.monthlyOutputTokensUsed}
                  limit={budget.monthlyOutputTokenLimit}
                />
                <TokenProgress
                  label={aiT.monthlyCalls}
                  used={budget.monthlyCallsUsed}
                  limit={budget.monthlyCallLimit}
                />
              </div>

              {/* 限额配置 */}
              <div className="grid gap-3 md:grid-cols-3 text-sm">
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.dailyInputTokenLimit}</label>
                  <LimitInput role={budget.role} field="dailyInputTokenLimit" budget={budget} />
                </div>
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.dailyOutputTokenLimit}</label>
                  <LimitInput role={budget.role} field="dailyOutputTokenLimit" budget={budget} />
                </div>
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.dailyCallLimit}</label>
                  <LimitInput role={budget.role} field="dailyCallLimit" budget={budget} />
                </div>
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.monthlyInputTokenLimit}</label>
                  <LimitInput role={budget.role} field="monthlyInputTokenLimit" budget={budget} />
                </div>
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.monthlyOutputTokenLimit}</label>
                  <LimitInput role={budget.role} field="monthlyOutputTokenLimit" budget={budget} />
                </div>
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.monthlyCallLimit}</label>
                  <LimitInput role={budget.role} field="monthlyCallLimit" budget={budget} />
                </div>
              </div>

              {/* 配置项 */}
              <div className="grid gap-3 md:grid-cols-2 mt-3">
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.autoTriggerMinSeverity}</label>
                  <p className="text-[10px] text-muted mb-1">{aiT.autoTriggerMinSeverityDesc}</p>
                  <select
                    value={severity}
                    onChange={(e) => setEdit(budget.role, "autoTriggerMinSeverity", e.target.value)}
                    disabled={!isAdmin}
                    className="w-full px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 disabled:opacity-50"
                  >
                    {SEVERITY_OPTIONS.map((s) => {
                      const labels: Record<string, string> = {
                        critical: aiT.severityCritical,
                        high: aiT.severityHigh,
                        medium: aiT.severityMedium,
                        low: aiT.severityLow,
                        off: aiT.severityOff,
                      };
                      return (
                        <option key={s} value={s} className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">
                          {labels[s] || s}
                        </option>
                      );
                    })}
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-muted mb-1">{aiT.fallbackProvider}</label>
                  <select
                    value={fallbackId ?? ""}
                    onChange={(e) => {
                      const val = e.target.value;
                      setEdit(budget.role, "fallbackProviderId", val ? parseInt(val) : null);
                    }}
                    disabled={!isAdmin}
                    className="w-full px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-default disabled:opacity-50"
                  >
                    <option value="">{aiT.noFallback}</option>
                    {providers.map((p) => (
                      <option key={p.id} value={p.id}>{p.name} ({p.model})</option>
                    ))}
                  </select>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
