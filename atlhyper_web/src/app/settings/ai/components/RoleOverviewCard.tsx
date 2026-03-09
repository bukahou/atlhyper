"use client";

import { useEffect, useState, useCallback } from "react";
import { Bot, Brain, MessageSquare, Users, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { UserRole } from "@/types/auth";
import { toast } from "@/components/common/Toast";
import {
  getRolesOverview,
  updateProviderRoles,
  type RoleOverview,
  type AIProvider,
} from "@/api/ai-provider";
import type { AISettingsPageTranslations } from "@/types/i18n";

const roleIcons: Record<string, typeof Bot> = {
  background: Brain,
  chat: MessageSquare,
  analysis: Users,
};

const roleStyles: Record<string, string> = {
  background: "border-blue-200 dark:border-blue-800",
  chat: "border-green-200 dark:border-green-800",
  analysis: "border-purple-200 dark:border-purple-800",
};

const roleIconColors: Record<string, string> = {
  background: "text-blue-500",
  chat: "text-green-500",
  analysis: "text-purple-500",
};

interface RoleOverviewCardProps {
  providers: AIProvider[];
  onRoleChanged?: () => void;
}

export function RoleOverviewCard({ providers, onRoleChanged }: RoleOverviewCardProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;
  const { isAuthenticated, user } = useAuthStore();
  const isAdmin = user?.role === UserRole.ADMIN;
  const [roles, setRoles] = useState<RoleOverview[]>([]);
  const [savingRole, setSavingRole] = useState<string | null>(null);

  const loadRoles = useCallback(async () => {
    if (!isAuthenticated) return;
    try {
      const res = await getRolesOverview();
      setRoles(res.data.data);
    } catch {
      // ignore
    }
  }, [isAuthenticated]);

  useEffect(() => {
    loadRoles();
  }, [loadRoles]);

  // 角色分配变更处理
  const handleRoleAssign = async (role: string, newProviderId: number | null) => {
    setSavingRole(role);
    try {
      // 找到当前持有该角色的 Provider，移除该角色
      const currentHolder = providers.find((p) => p.roles?.includes(role));
      if (currentHolder) {
        const newRoles = currentHolder.roles.filter((r) => r !== role);
        await updateProviderRoles(currentHolder.id, newRoles);
      }

      // 如果选择了新 Provider（非"未分配"），添加该角色
      if (newProviderId !== null) {
        const newHolder = providers.find((p) => p.id === newProviderId);
        if (newHolder) {
          const newRoles = [...(newHolder.roles || []).filter((r) => r !== role), role];
          await updateProviderRoles(newHolder.id, newRoles);
        }
      }

      toast.success(aiT.roleAssignSuccess);
      await loadRoles();
      onRoleChanged?.();
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      if (status === 409) {
        toast.error(aiT.roleAssignConflict);
      } else {
        toast.error(aiT.saveFailed);
      }
      await loadRoles();
    } finally {
      setSavingRole(null);
    }
  };

  // Demo mode or no data
  const displayRoles = roles.length > 0 ? roles : [
    { role: "background", roleName: aiT.roleBackground, provider: null },
    { role: "chat", roleName: aiT.roleChat, provider: null },
    { role: "analysis", roleName: aiT.roleAnalysis, provider: null },
  ];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <h3 className="text-lg font-medium text-default">{aiT.roleOverview}</h3>
      </div>
      <div className="p-6 grid gap-4 md:grid-cols-3">
        {displayRoles.map((role) => {
          const Icon = roleIcons[role.role] || Bot;
          const isSaving = savingRole === role.role;
          const currentProviderId = role.provider?.id ?? null;

          return (
            <div
              key={role.role}
              className={`rounded-lg border p-4 ${roleStyles[role.role] || "border-[var(--border-color)]"}`}
            >
              <div className="flex items-center gap-2 mb-3">
                <Icon className={`w-5 h-5 ${roleIconColors[role.role] || "text-muted"}`} />
                <span className="font-medium text-default">{role.roleName}</span>
                {isSaving && <Loader2 className="w-4 h-4 animate-spin text-muted" />}
              </div>

              {/* Provider selector */}
              {isAdmin && isAuthenticated ? (
                <select
                  value={currentProviderId ?? ""}
                  onChange={(e) => {
                    const val = e.target.value;
                    handleRoleAssign(role.role, val === "" ? null : Number(val));
                  }}
                  disabled={isSaving}
                  className="w-full px-2 py-1.5 text-sm rounded border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 disabled:opacity-50"
                >
                  <option value="" className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">{aiT.roleUnassigned}</option>
                  {providers.map((p) => (
                    <option key={p.id} value={p.id} className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">
                      {p.name} ({p.model})
                    </option>
                  ))}
                </select>
              ) : (
                // 非 Admin：只读展示
                role.provider ? (
                  <div className="text-sm space-y-1">
                    <p className="text-default font-medium">{role.provider.name}</p>
                    <p className="text-muted">{role.provider.model}</p>
                  </div>
                ) : (
                  <p className="text-sm text-muted italic">{aiT.roleUnassigned}</p>
                )
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
