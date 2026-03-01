"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader } from "@/components/common";
import { Shield, User, Eye } from "lucide-react";
import { UserRole } from "@/types/auth";
import type { RolesTranslations } from "@/types/i18n";
import { PermissionBadge, PermissionMatrix } from "./PermissionMatrix";

// 角色定义（需要翻译的部分通过 t 获取）
function getRoles(t: RolesTranslations) {
  return [
    {
      id: UserRole.ADMIN,
      name: "Admin",
      description: t.adminDescription,
      icon: Shield,
      color: "text-red-500",
      bgColor: "bg-red-100 dark:bg-red-900/30",
    },
    {
      id: UserRole.OPERATOR,
      name: "Operator",
      description: t.operatorDescription,
      icon: User,
      color: "text-blue-500",
      bgColor: "bg-blue-100 dark:bg-blue-900/30",
    },
    {
      id: UserRole.VIEWER,
      name: "Viewer",
      description: t.viewerDescription,
      icon: Eye,
      color: "text-gray-500",
      bgColor: "bg-gray-100 dark:bg-gray-700",
    },
  ];
}

export default function RolesPage() {
  const { t } = useI18n();
  const rolesT = t.roles;

  const roles = getRoles(rolesT);

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.roles} description={rolesT.pageDescription} />

        {/* 角色卡片 */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
          {roles.map((role) => {
            const Icon = role.icon;
            return (
              <div
                key={role.id}
                className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5"
              >
                <div className="flex items-center gap-2 sm:gap-3 mb-2 sm:mb-3">
                  <div className={`p-1.5 sm:p-2 rounded-lg ${role.bgColor}`}>
                    <Icon className={`w-4 h-4 sm:w-5 sm:h-5 ${role.color}`} />
                  </div>
                  <div>
                    <h3 className="font-semibold text-default text-sm sm:text-base">{role.name}</h3>
                    <p className="text-[10px] sm:text-xs text-muted">Level {role.id}</p>
                  </div>
                </div>
                <p className="text-xs sm:text-sm text-secondary">{role.description}</p>
              </div>
            );
          })}
        </div>

        {/* 权限矩阵 */}
        <PermissionMatrix roles={roles} t={rolesT} />

        {/* 权限说明 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
          <h3 className="font-semibold text-default text-sm sm:text-base mb-3 sm:mb-4">{rolesT.permissionLevelDescription}</h3>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
            {(["full", "read", "partial", "none"] as const).map((perm) => {
              const labels: Record<string, { title: string; desc: string }> = {
                full: { title: rolesT.fullPermission, desc: rolesT.fullPermissionDesc },
                read: { title: rolesT.readOnlyPermission, desc: rolesT.readOnlyPermissionDesc },
                partial: { title: rolesT.partialPermission, desc: rolesT.partialPermissionDesc },
                none: { title: rolesT.noPermission, desc: rolesT.noPermissionDesc },
              };
              return (
                <div key={perm} className="flex items-start gap-2 sm:gap-3">
                  <div className="hidden sm:block flex-shrink-0"><PermissionBadge permission={perm} t={rolesT} /></div>
                  <div className="sm:hidden flex-shrink-0"><PermissionBadge permission={perm} t={rolesT} compact /></div>
                  <div className="min-w-0">
                    <p className="text-xs sm:text-sm font-medium text-default">{labels[perm].title}</p>
                    <p className="text-[10px] sm:text-xs text-muted hidden sm:block">{labels[perm].desc}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </Layout>
  );
}
