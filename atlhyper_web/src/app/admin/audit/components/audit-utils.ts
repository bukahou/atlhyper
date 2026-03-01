import type { AuditTranslations } from "@/types/i18n";
import { UserRole } from "@/types/auth";

/** 获取角色标签 */
export function getRoleLabel(role: number, auditT: AuditTranslations): string {
  if (role === 0) return auditT.roles.guest;
  if (role === UserRole.VIEWER) return auditT.roles.viewer;
  if (role === UserRole.OPERATOR) return auditT.roles.operator;
  if (role === UserRole.ADMIN) return auditT.roles.admin;
  return `Role ${role}`;
}

/** 获取资源标签 */
export function getResourceLabel(resource: string, auditT: AuditTranslations): string {
  const key = resource as keyof typeof auditT.resources;
  return auditT.resources[key] || resource;
}

/** 获取操作的显示标签 */
export function getActionLabel(action: string, resource: string, auditT: AuditTranslations): string {
  // 构造 actionLabels 的键名，例如 login + user -> loginUser
  const labelKey = `${action}${resource.charAt(0).toUpperCase()}${resource.slice(1)}` as keyof typeof auditT.actionLabels;
  const label = auditT.actionLabels[labelKey];
  if (label) return label;

  // 回退：显示 action + resource
  const resourceName = getResourceLabel(resource, auditT);
  const actionKey = action as keyof typeof auditT.actionNames;
  const actionName = auditT.actionNames[actionKey] || action;
  return `${actionName} ${resourceName}`;
}
