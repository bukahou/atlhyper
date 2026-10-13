import { Shield, User, Eye, Users } from "lucide-react";
import { UserRole } from "@/types/auth";
import type { RolesTranslations } from "@/types/i18n";

// 角色卡片 / 矩阵列（从低到高，与后端等级 0-3 一致）
export interface RoleInfo {
  level: number;
  name: string;
  description: string;
  icon: typeof Shield;
  color: string;
  bgColor: string;
}

export function getRoles(t: RolesTranslations): RoleInfo[] {
  return [
    { level: UserRole.MEMBER, name: t.memberName, description: t.memberDescription, icon: Users, color: "text-gray-500", bgColor: "bg-gray-100 dark:bg-gray-700" },
    { level: UserRole.VIEWER, name: "Viewer", description: t.viewerDescription, icon: Eye, color: "text-green-500", bgColor: "bg-green-100 dark:bg-green-900/30" },
    { level: UserRole.OPERATOR, name: "Operator", description: t.operatorDescription, icon: User, color: "text-blue-500", bgColor: "bg-blue-100 dark:bg-blue-900/30" },
    { level: UserRole.ADMIN, name: "Admin", description: t.adminDescription, icon: Shield, color: "text-red-500", bgColor: "bg-red-100 dark:bg-red-900/30" },
  ];
}
