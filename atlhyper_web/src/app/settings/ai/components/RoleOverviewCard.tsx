"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { Bot, Brain, MessageSquare, Users, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { toast } from "@/components/common/Toast";
import type { AIProvider } from "@/api/ai-provider";
import { useCan } from "@/hooks/useCan";
import { mockRolesOverview } from "@/mock/ai-settings";
import { aiRolesOverviewQuery, assignAIRoleMutation } from "@/queries/ai-settings";

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
  /** 演示模式（没有 ai.settings.read）：显示演示数据，不发请求 */
  demo: boolean;
  providers: AIProvider[];
}

export function RoleOverviewCard({ demo, providers }: RoleOverviewCardProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;
  const canViewAI = !demo;
  const isAdmin = useCan()("ai.settings.write");
  const rolesQuery = useQuery(aiRolesOverviewQuery(canViewAI));
  const roles = demo ? mockRolesOverview : (rolesQuery.data ?? []);
  // 分配无论成败，工厂都会让角色总览与 Provider 列表重新取数
  const assignRole = useMutation(assignAIRoleMutation());
  const savingRole = assignRole.isPending ? assignRole.variables.role : null;

  // 角色分配变更处理
  const handleRoleAssign = (role: string, newProviderId: number | null) => {
    assignRole.mutate(
      { role, providerId: newProviderId, providers },
      {
        onSuccess: () => toast.success(aiT.roleAssignSuccess),
        onError: (err: unknown) => {
          const status = (err as { response?: { status?: number } })?.response?.status;
          toast.error(status === 409 ? aiT.roleAssignConflict : aiT.saveFailed);
        },
      },
    );
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
              {isAdmin && canViewAI ? (
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
