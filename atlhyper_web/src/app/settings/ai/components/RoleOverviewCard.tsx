"use client";

import { useEffect, useState } from "react";
import { Users, Bot, Brain, MessageSquare } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { getRolesOverview, type RoleOverview } from "@/api/ai-provider";
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

export function RoleOverviewCard() {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;
  const { isAuthenticated } = useAuthStore();
  const [roles, setRoles] = useState<RoleOverview[]>([]);

  useEffect(() => {
    if (!isAuthenticated) return;
    getRolesOverview()
      .then((res) => setRoles(res.data.data))
      .catch(() => {});
  }, [isAuthenticated]);

  // Demo mode: show placeholder
  if (!isAuthenticated || roles.length === 0) {
    const mockRoles: RoleOverview[] = [
      { role: "background", roleName: aiT.roleBackground, provider: null },
      { role: "chat", roleName: aiT.roleChat, provider: null },
      { role: "analysis", roleName: aiT.roleAnalysis, provider: null },
    ];
    return <RoleCards roles={mockRoles} aiT={aiT} />;
  }

  return <RoleCards roles={roles} aiT={aiT} />;
}

function RoleCards({ roles, aiT }: { roles: RoleOverview[]; aiT: AISettingsPageTranslations }) {
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <h3 className="text-lg font-medium text-default">{aiT.roleOverview}</h3>
      </div>
      <div className="p-6 grid gap-4 md:grid-cols-3">
        {roles.map((role) => {
          const Icon = roleIcons[role.role] || Bot;
          return (
            <div
              key={role.role}
              className={`rounded-lg border p-4 ${roleStyles[role.role] || "border-[var(--border-color)]"}`}
            >
              <div className="flex items-center gap-2 mb-3">
                <Icon className={`w-5 h-5 ${roleIconColors[role.role] || "text-muted"}`} />
                <span className="font-medium text-default">{role.roleName}</span>
              </div>
              {role.provider ? (
                <div className="text-sm space-y-1">
                  <p className="text-default font-medium">{role.provider.name}</p>
                  <p className="text-muted">{role.provider.model}</p>
                </div>
              ) : (
                <p className="text-sm text-muted italic">{aiT.roleUnassigned}</p>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
