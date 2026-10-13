"use client";

import { Fragment } from "react";
import { Check, X } from "lucide-react";
import type { RolesTranslations } from "@/types/i18n";
import type { PermissionTable } from "@/types/auth";
import type { RoleInfo } from "./roles";

interface PermissionMatrixProps {
  roles: RoleInfo[];
  table: PermissionTable | null;
  error: boolean;
  t: RolesTranslations;
}

// 拥有 / 无 标记（形状 + 文字，不只靠颜色）
function GrantMark({ granted, t, compact }: { granted: boolean; t: RolesTranslations; compact?: boolean }) {
  const base = compact
    ? "inline-flex items-center justify-center w-6 h-6 rounded-full"
    : "inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium";
  return granted ? (
    <span className={`${base} bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400`} title={t.granted}>
      <Check className="w-3 h-3" />
      {!compact && t.granted}
    </span>
  ) : (
    <span className={`${base} bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400`} title={t.notGranted}>
      <X className="w-3 h-3" />
      {!compact && t.notGranted}
    </span>
  );
}

export function PermissionMatrix({ roles, table, error, t }: PermissionMatrixProps) {
  const tierTitle: Record<number, string> = { 0: t.tierMember, 1: t.tierViewer, 2: t.tierOperator, 3: t.tierAdmin };
  const rows = table?.rows ?? [];
  const tiers = table?.levels ?? [];
  const label = (perm: string) => t.permissionLabels[perm as keyof typeof t.permissionLabels] ?? perm;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-3 sm:px-4 py-3 border-b border-[var(--border-color)]">
        <h3 className="font-semibold text-default text-sm sm:text-base">{t.permissionMatrix}</h3>
        <p className="text-xs sm:text-sm text-muted mt-1">{t.permissionMatrixDescription}</p>
      </div>

      {error && <p className="px-4 py-6 text-sm text-muted">{t.loadFailed}</p>}

      {/* 移动端卡片视图 */}
      <div className="sm:hidden">
        {tiers.map((tier) => (
          <div key={tier}>
            <div className="px-3 py-2 bg-[var(--background)] text-xs font-semibold text-muted tracking-wider">
              {tierTitle[tier] ?? `Level ${tier}`}
            </div>
            <div className="divide-y divide-[var(--border-color)]">
              {rows
                .filter((row) => row.minLevel === tier)
                .map((row) => (
                  <div key={row.permission} className="px-3 py-2.5">
                    <p className="text-sm text-default font-medium mb-1.5">{label(row.permission)}</p>
                    <div className="flex items-center gap-3">
                      {roles.map((role) => (
                        <div key={role.level} className="flex items-center gap-1">
                          <role.icon className={`w-3 h-3 ${role.color}`} />
                          <GrantMark granted={role.level >= row.minLevel} t={t} compact />
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
            </div>
          </div>
        ))}
      </div>

      {/* 桌面端表格视图 */}
      <div className="hidden sm:block overflow-x-auto">
        <table className="w-full">
          <thead className="bg-[var(--background)]">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-muted">{t.permissionColumn}</th>
              {roles.map((role) => (
                <th key={role.level} className="px-4 py-3 text-center text-sm font-medium text-muted w-[120px]">
                  <div className="flex items-center justify-center gap-2">
                    <role.icon className={`w-4 h-4 ${role.color}`} />
                    {role.name}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {tiers.map((tier) => (
              <Fragment key={tier}>
                <tr className="bg-[var(--background)]">
                  <td colSpan={roles.length + 1} className="px-4 py-2 text-xs font-semibold text-muted tracking-wider">
                    {tierTitle[tier] ?? `Level ${tier}`}
                  </td>
                </tr>
                {rows
                  .filter((row) => row.minLevel === tier)
                  .map((row) => (
                    <tr key={row.permission} className="border-t border-[var(--border-color)] hover:bg-[var(--background)]">
                      <td className="px-4 py-3 text-sm text-default">
                        {label(row.permission)}
                        <span className="ml-2 text-xs text-muted font-mono">{row.permission}</span>
                      </td>
                      {roles.map((role) => (
                        <td key={role.level} className="px-4 py-3 text-center">
                          <GrantMark granted={role.level >= row.minLevel} t={t} />
                        </td>
                      ))}
                    </tr>
                  ))}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
