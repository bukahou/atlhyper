"use client";

import { Loader2, Save } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { AIProvider, RoleBudget } from "@/api/ai-provider";

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

/** 一个角色的预算里还没保存的修改：字段名 → 新值 */
export type BudgetEdits = Record<string, number | string | null>;

/** 限额字段的名字（都是数字） */
type BudgetLimitField =
  | "dailyInputTokenLimit"
  | "dailyOutputTokenLimit"
  | "dailyCallLimit"
  | "monthlyInputTokenLimit"
  | "monthlyOutputTokenLimit"
  | "monthlyCallLimit";

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

// 限额输入框
function LimitInput({ value, onChange, disabled }: { value: number; onChange: (limit: number) => void; disabled: boolean }) {
  return (
    <input
      type="number"
      value={value}
      onChange={(e) => onChange(parseInt(e.target.value) || 0)}
      disabled={disabled}
      className="w-full px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-default disabled:opacity-50"
    />
  );
}

interface RoleBudgetCardProps {
  budget: RoleBudget;
  /** 这个角色还没保存的修改 */
  edits: BudgetEdits;
  onEdit: (field: string, value: number | string | null) => void;
  /** 备用 Provider 的候选 */
  providers: AIProvider[];
  isAdmin: boolean;
  saving: boolean;
  onSave: () => void;
}

/** 一个角色的预算：已用量、限额、触发方式、备用 Provider */
export function RoleBudgetCard({ budget, edits, onEdit, providers, isAdmin, saving, onSave }: RoleBudgetCardProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;

  const roleLabelKey = ROLE_LABELS[budget.role] as keyof typeof aiT | undefined;
  const roleLabel = (roleLabelKey ? aiT[roleLabelKey] : budget.role) as string;
  const limitOf = (field: BudgetLimitField) => (edits[field] as number | undefined) ?? budget[field];
  const severity = (edits.autoTriggerMinSeverity as string) ?? budget.autoTriggerMinSeverity;
  const triggerMode = (edits.autoTriggerMode as string) ?? budget.autoTriggerMode ?? "auto";
  const schedStart = (edits.scheduleStartTime as string) ?? budget.scheduleStartTime ?? "";
  const schedEnd = (edits.scheduleEndTime as string) ?? budget.scheduleEndTime ?? "";
  // 备用 Provider 可以改成「不设」（null）：要看有没有改过，不能用 ?? 判断，否则改成「不设」后下拉框会跳回已保存的值
  const fallbackId =
    "fallbackProviderId" in edits ? (edits.fallbackProviderId as number | null) : budget.fallbackProviderId;
  const hasChanges = Object.keys(edits).length > 0;

  return (
    <div
      className={`rounded-lg border p-4 ${ROLE_STYLES[budget.role] || "border-[var(--border-color)]"}`}
    >
      <div className="flex items-center justify-between mb-3">
        <h4 className="font-medium text-default">{roleLabel}</h4>
        {isAdmin && (
          <button
            onClick={onSave}
            disabled={!hasChanges || saving}
            className="flex items-center gap-1.5 px-3 py-1 text-xs rounded-lg bg-violet-600 text-white hover:bg-violet-700 disabled:opacity-50 transition-colors"
          >
            {saving ? (
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
          <LimitInput value={limitOf("dailyInputTokenLimit")} onChange={(limit) => onEdit("dailyInputTokenLimit", limit)} disabled={!isAdmin} />
        </div>
        <div>
          <label className="block text-xs text-muted mb-1">{aiT.dailyOutputTokenLimit}</label>
          <LimitInput value={limitOf("dailyOutputTokenLimit")} onChange={(limit) => onEdit("dailyOutputTokenLimit", limit)} disabled={!isAdmin} />
        </div>
        <div>
          <label className="block text-xs text-muted mb-1">{aiT.dailyCallLimit}</label>
          <LimitInput value={limitOf("dailyCallLimit")} onChange={(limit) => onEdit("dailyCallLimit", limit)} disabled={!isAdmin} />
        </div>
        <div>
          <label className="block text-xs text-muted mb-1">{aiT.monthlyInputTokenLimit}</label>
          <LimitInput value={limitOf("monthlyInputTokenLimit")} onChange={(limit) => onEdit("monthlyInputTokenLimit", limit)} disabled={!isAdmin} />
        </div>
        <div>
          <label className="block text-xs text-muted mb-1">{aiT.monthlyOutputTokenLimit}</label>
          <LimitInput value={limitOf("monthlyOutputTokenLimit")} onChange={(limit) => onEdit("monthlyOutputTokenLimit", limit)} disabled={!isAdmin} />
        </div>
        <div>
          <label className="block text-xs text-muted mb-1">{aiT.monthlyCallLimit}</label>
          <LimitInput value={limitOf("monthlyCallLimit")} onChange={(limit) => onEdit("monthlyCallLimit", limit)} disabled={!isAdmin} />
        </div>
      </div>

      {/* 触发模式（仅 analysis 角色显示） */}
      {budget.role === "analysis" && (
        <div className="mt-3 space-y-2">
          <div>
            <label className="block text-xs text-muted mb-1">{aiT.autoTriggerMode}</label>
            <p className="text-[10px] text-muted mb-1">{aiT.autoTriggerModeDesc}</p>
            <div className="flex gap-1">
              {(["auto", "manual", "schedule"] as const).map((mode) => {
                const labels = { auto: aiT.triggerModeAuto, manual: aiT.triggerModeManual, schedule: aiT.triggerModeSchedule };
                return (
                  <button
                    key={mode}
                    onClick={() => onEdit("autoTriggerMode", mode)}
                    disabled={!isAdmin}
                    className={`px-3 py-1 text-xs rounded-lg border transition-colors disabled:opacity-50 ${
                      triggerMode === mode
                        ? "bg-violet-600 text-white border-violet-600"
                        : "border-[var(--border-color)] text-muted hover:bg-gray-100 dark:hover:bg-gray-800"
                    }`}
                  >
                    {labels[mode]}
                  </button>
                );
              })}
            </div>
          </div>
          {triggerMode === "schedule" && (
            <div>
              <label className="block text-xs text-muted mb-1">{aiT.scheduleWindow}</label>
              <p className="text-[10px] text-muted mb-1">{aiT.scheduleWindowDesc}</p>
              <div className="flex items-center gap-2">
                <input
                  type="time"
                  value={schedStart}
                  onChange={(e) => onEdit("scheduleStartTime", e.target.value)}
                  disabled={!isAdmin}
                  className="px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-default disabled:opacity-50"
                />
                <span className="text-xs text-muted">～</span>
                <input
                  type="time"
                  value={schedEnd}
                  onChange={(e) => onEdit("scheduleEndTime", e.target.value)}
                  disabled={!isAdmin}
                  className="px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-default disabled:opacity-50"
                />
              </div>
            </div>
          )}
        </div>
      )}

      {/* 配置项 */}
      <div className="grid gap-3 md:grid-cols-2 mt-3">
        <div>
          <label className="block text-xs text-muted mb-1">{aiT.autoTriggerMinSeverity}</label>
          <p className="text-[10px] text-muted mb-1">{aiT.autoTriggerMinSeverityDesc}</p>
          <select
            value={severity}
            onChange={(e) => onEdit("autoTriggerMinSeverity", e.target.value)}
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
              onEdit("fallbackProviderId", val ? parseInt(val) : null);
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
}
