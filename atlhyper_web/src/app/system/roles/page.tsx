"use client";

import { Fragment } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader } from "@/components/common";
import { Shield, User, Eye, Check, X, Info } from "lucide-react";
import { UserRole } from "@/types/auth";

// 角色定义
const roles = [
  {
    id: UserRole.ADMIN,
    name: "Admin",
    description: "系统管理员，拥有全部权限",
    icon: Shield,
    color: "text-red-500",
    bgColor: "bg-red-100 dark:bg-red-900/30",
  },
  {
    id: UserRole.OPERATOR,
    name: "Operator",
    description: "操作员，可执行日常运维操作",
    icon: User,
    color: "text-blue-500",
    bgColor: "bg-blue-100 dark:bg-blue-900/30",
  },
  {
    id: UserRole.VIEWER,
    name: "Viewer",
    description: "观察者，只读权限",
    icon: Eye,
    color: "text-gray-500",
    bgColor: "bg-gray-100 dark:bg-gray-700",
  },
];

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

const permissions: ResourcePermission[] = [
  // 用户管理
  { resource: "用户管理", category: "系统", admin: "full", operator: "none", viewer: "none", note: "查看用户列表" },
  { resource: "角色分配", category: "系统", admin: "full", operator: "none", viewer: "none" },
  { resource: "审计日志", category: "系统", admin: "full", operator: "read", viewer: "read" },
  { resource: "通知配置", category: "系统", admin: "full", operator: "read", viewer: "read" },
  // 集群资源
  { resource: "Pods", category: "集群", admin: "full", operator: "full", viewer: "read" },
  { resource: "Nodes", category: "集群", admin: "full", operator: "read", viewer: "read" },
  { resource: "Deployments", category: "集群", admin: "full", operator: "full", viewer: "read" },
  { resource: "Services", category: "集群", admin: "full", operator: "full", viewer: "read" },
  { resource: "Namespaces", category: "集群", admin: "full", operator: "read", viewer: "read" },
  { resource: "Ingress", category: "集群", admin: "full", operator: "full", viewer: "read" },
  { resource: "ConfigMaps", category: "集群", admin: "full", operator: "full", viewer: "read" },
  // 监控告警
  { resource: "指标查看", category: "监控", admin: "full", operator: "read", viewer: "read" },
  { resource: "日志查看", category: "监控", admin: "full", operator: "read", viewer: "read" },
  { resource: "告警规则", category: "监控", admin: "full", operator: "partial", viewer: "read", note: "Operator 可静默告警" },
  // AI 功能
  { resource: "AI 诊断", category: "AI", admin: "full", operator: "full", viewer: "read" },
  { resource: "AI 工作台", category: "AI", admin: "full", operator: "full", viewer: "read" },
];

// 权限标记组件
function PermissionBadge({ permission }: { permission: Permission }) {
  switch (permission) {
    case "full":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
          <Check className="w-3 h-3" />
          完全
        </span>
      );
    case "read":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400">
          <Eye className="w-3 h-3" />
          只读
        </span>
      );
    case "partial":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
          <Info className="w-3 h-3" />
          部分
        </span>
      );
    case "none":
      return (
        <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400">
          <X className="w-3 h-3" />
          无
        </span>
      );
  }
}

export default function RolesPage() {
  const { t } = useI18n();

  // 按分类分组
  const categories = [...new Set(permissions.map((p) => p.category))];

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title="角色权限" description="系统角色及其权限说明" />

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
            <h3 className="font-semibold text-default">权限矩阵</h3>
            <p className="text-sm text-muted mt-1">各角色对系统资源的访问权限</p>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-[var(--background)]">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500 w-[200px]">
                    资源
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
                    备注
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
                      .map((perm, idx) => (
                        <tr
                          key={`${category}-${perm.resource}`}
                          className="border-t border-[var(--border-color)] hover:bg-[var(--background)]"
                        >
                          <td className="px-4 py-3 text-sm text-default">
                            {perm.resource}
                          </td>
                          <td className="px-4 py-3 text-center">
                            <PermissionBadge permission={perm.admin} />
                          </td>
                          <td className="px-4 py-3 text-center">
                            <PermissionBadge permission={perm.operator} />
                          </td>
                          <td className="px-4 py-3 text-center">
                            <PermissionBadge permission={perm.viewer} />
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
          <h3 className="font-semibold text-default mb-4">权限级别说明</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="flex items-start gap-3">
              <PermissionBadge permission="full" />
              <div>
                <p className="text-sm font-medium text-default">完全权限</p>
                <p className="text-xs text-muted">可查看、创建、修改、删除</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <PermissionBadge permission="read" />
              <div>
                <p className="text-sm font-medium text-default">只读权限</p>
                <p className="text-xs text-muted">仅可查看，不可修改</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <PermissionBadge permission="partial" />
              <div>
                <p className="text-sm font-medium text-default">部分权限</p>
                <p className="text-xs text-muted">有限的操作权限</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <PermissionBadge permission="none" />
              <div>
                <p className="text-sm font-medium text-default">无权限</p>
                <p className="text-xs text-muted">不可访问此资源</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
