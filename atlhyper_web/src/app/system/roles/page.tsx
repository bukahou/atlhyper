"use client";

import { Fragment } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader } from "@/components/common";
import { Shield, User, Eye, Check, X, Info } from "lucide-react";
import { UserRole } from "@/types/auth";
import type { RolesTranslations } from "@/types/i18n";

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

// 权限类型
type Permission = "full" | "read" | "none" | "partial";

// 资源权限定义
interface ResourcePermission {
  resource: string;
  category: string;
  admin: Permission;
  operator: Permission;
  viewer: Permission;
  note?: string;
}

// 获取权限列表（需要翻译的部分通过 t 获取）
function getPermissions(t: RolesTranslations): ResourcePermission[] {
  return [
    // 用户管理
    { resource: t.userManagement, category: t.categorySystem, admin: "full", operator: "none", viewer: "none", note: t.noteViewUserList },
    { resource: t.roleAssignment, category: t.categorySystem, admin: "full", operator: "none", viewer: "none" },
    { resource: t.auditLogs, category: t.categorySystem, admin: "full", operator: "read", viewer: "read" },
    { resource: t.notificationConfig, category: t.categorySystem, admin: "full", operator: "read", viewer: "read" },
    // 集群资源
    { resource: "Pods", category: t.categoryCluster, admin: "full", operator: "full", viewer: "read" },
    { resource: "Nodes", category: t.categoryCluster, admin: "full", operator: "read", viewer: "read" },
    { resource: "Deployments", category: t.categoryCluster, admin: "full", operator: "full", viewer: "read" },
    { resource: "Services", category: t.categoryCluster, admin: "full", operator: "full", viewer: "read" },
    { resource: "Namespaces", category: t.categoryCluster, admin: "full", operator: "read", viewer: "read" },
    { resource: "Ingress", category: t.categoryCluster, admin: "full", operator: "full", viewer: "read" },
    { resource: "ConfigMaps", category: t.categoryCluster, admin: "full", operator: "full", viewer: "read" },
    // 监控告警
    { resource: t.metricsView, category: t.categoryMonitoring, admin: "full", operator: "read", viewer: "read" },
    { resource: t.logsView, category: t.categoryMonitoring, admin: "full", operator: "read", viewer: "read" },
    { resource: t.alertRules, category: t.categoryMonitoring, admin: "full", operator: "partial", viewer: "read", note: t.noteOperatorSilenceAlert },
    // AI 功能
    { resource: t.aiDiagnosis, category: t.categoryAI, admin: "full", operator: "full", viewer: "read" },
    { resource: t.aiWorkbench, category: t.categoryAI, admin: "full", operator: "full", viewer: "read" },
  ];
}

// 权限标记组件
function PermissionBadge({ permission, t }: { permission: Permission; t: RolesTranslations }) {
  switch (permission) {
    case "full":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
          <Check className="w-3 h-3" />
          {t.permissionFull}
        </span>
      );
    case "read":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400">
          <Eye className="w-3 h-3" />
          {t.permissionReadOnly}
        </span>
      );
    case "partial":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
          <Info className="w-3 h-3" />
          {t.permissionPartial}
        </span>
      );
    case "none":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400">
          <X className="w-3 h-3" />
          {t.permissionNone}
        </span>
      );
  }
}

export default function RolesPage() {
  const { t } = useI18n();
  const rolesT = t.roles;

  // 获取翻译后的数据
  const roles = getRoles(rolesT);
  const permissions = getPermissions(rolesT);

  // 按分类分组
  const categories = [...new Set(permissions.map((p) => p.category))];

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.roles} description={rolesT.pageDescription} />

        {/* 角色卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {roles.map((role) => {
            const Icon = role.icon;
            return (
              <div
                key={role.id}
                className="bg-card rounded-xl border border-[var(--border-color)] p-5"
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className={`p-2 rounded-lg ${role.bgColor}`}>
                    <Icon className={`w-5 h-5 ${role.color}`} />
                  </div>
                  <div>
                    <h3 className="font-semibold text-default">{role.name}</h3>
                    <p className="text-xs text-muted">Level {role.id}</p>
                  </div>
                </div>
                <p className="text-sm text-secondary">{role.description}</p>
              </div>
            );
          })}
        </div>

        {/* 权限矩阵 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <div className="px-4 py-3 border-b border-[var(--border-color)]">
            <h3 className="font-semibold text-default">{rolesT.permissionMatrix}</h3>
            <p className="text-sm text-muted mt-1">{rolesT.permissionMatrixDescription}</p>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-[var(--background)]">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500 w-[200px]">
                    {rolesT.resource}
                  </th>
                  {roles.map((role) => (
                    <th
                      key={role.id}
                      className="px-4 py-3 text-center text-sm font-medium text-gray-500 w-[120px]"
                    >
                      <div className="flex items-center justify-center gap-2">
                        <role.icon className={`w-4 h-4 ${role.color}`} />
                        {role.name}
                      </div>
                    </th>
                  ))}
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                    {rolesT.notes}
                  </th>
                </tr>
              </thead>
              <tbody>
                {categories.map((category) => (
                  <Fragment key={category}>
                    {/* 分类标题 */}
                    <tr className="bg-[var(--background)]">
                      <td
                        colSpan={5}
                        className="px-4 py-2 text-xs font-semibold text-muted uppercase tracking-wider"
                      >
                        {category}
                      </td>
                    </tr>
                    {/* 该分类下的资源 */}
                    {permissions
                      .filter((p) => p.category === category)
                      .map((perm) => (
                        <tr
                          key={`${category}-${perm.resource}`}
                          className="border-t border-[var(--border-color)] hover:bg-[var(--background)]"
                        >
                          <td className="px-4 py-3 text-sm text-default">
                            {perm.resource}
                          </td>
                          <td className="px-4 py-3 text-center">
                            <PermissionBadge permission={perm.admin} t={rolesT} />
                          </td>
                          <td className="px-4 py-3 text-center">
                            <PermissionBadge permission={perm.operator} t={rolesT} />
                          </td>
                          <td className="px-4 py-3 text-center">
                            <PermissionBadge permission={perm.viewer} t={rolesT} />
                          </td>
                          <td className="px-4 py-3 text-sm text-muted">
                            {perm.note || "-"}
                          </td>
                        </tr>
                      ))}
                  </Fragment>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* 权限说明 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
          <h3 className="font-semibold text-default mb-4">{rolesT.permissionLevelDescription}</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="flex items-start gap-3">
              <PermissionBadge permission="full" t={rolesT} />
              <div>
                <p className="text-sm font-medium text-default">{rolesT.fullPermission}</p>
                <p className="text-xs text-muted">{rolesT.fullPermissionDesc}</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <PermissionBadge permission="read" t={rolesT} />
              <div>
                <p className="text-sm font-medium text-default">{rolesT.readOnlyPermission}</p>
                <p className="text-xs text-muted">{rolesT.readOnlyPermissionDesc}</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <PermissionBadge permission="partial" t={rolesT} />
              <div>
                <p className="text-sm font-medium text-default">{rolesT.partialPermission}</p>
                <p className="text-xs text-muted">{rolesT.partialPermissionDesc}</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <PermissionBadge permission="none" t={rolesT} />
              <div>
                <p className="text-sm font-medium text-default">{rolesT.noPermission}</p>
                <p className="text-xs text-muted">{rolesT.noPermissionDesc}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
