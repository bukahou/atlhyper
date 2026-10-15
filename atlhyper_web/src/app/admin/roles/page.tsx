"use client";

import { useEffect, useState } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader } from "@/components/common";
import { getPermissionTable } from "@/api/auth";
import type { PermissionTable } from "@/types/auth";
import { PermissionMatrix } from "./PermissionMatrix";
import { getRoles } from "./roles";

export default function RolesPage() {
  const { t } = useI18n();
  const rolesT = t.roles;
  const roles = getRoles(rolesT);

  // 权限表由后端下发（唯一权限源），页面不另写一份
  const [table, setTable] = useState<PermissionTable | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => {
    getPermissionTable()
      .then((res) => setTable(res.data.data))
      .catch(() => setError(true));
  }, []);

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.roles} description={rolesT.pageDescription} />

        {/* 角色卡片 */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
          {roles.map((role) => {
            const Icon = role.icon;
            return (
              <div key={role.level} className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
                <div className="flex items-center gap-2 sm:gap-3 mb-2 sm:mb-3">
                  <div className={`p-1.5 sm:p-2 rounded-lg ${role.bgColor}`}>
                    <Icon className={`w-4 h-4 sm:w-5 sm:h-5 ${role.color}`} />
                  </div>
                  <div>
                    <h3 className="font-semibold text-default text-sm sm:text-base">{role.name}</h3>
                    <p className="text-[10px] sm:text-xs text-muted">Level {role.level}</p>
                  </div>
                </div>
                <p className="text-xs sm:text-sm text-secondary">{role.description}</p>
              </div>
            );
          })}
        </div>

        <PermissionMatrix roles={roles} table={table} error={error} t={rolesT} />
      </div>
    </Layout>
  );
}
