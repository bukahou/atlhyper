"use client";

import { Edit2, Trash2, Power, PowerOff, Shield, User, Eye } from "lucide-react";
import type { UserListItem } from "@/types/auth";
import { UserRole } from "@/types/auth";
import type { useI18n } from "@/i18n/context";

const roleConfig = {
  [UserRole.ADMIN]: {
    label: "Admin",
    color: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
    icon: Shield,
  },
  [UserRole.OPERATOR]: {
    label: "Operator",
    color: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
    icon: User,
  },
  [UserRole.VIEWER]: {
    label: "Viewer",
    color: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300",
    icon: Eye,
  },
};

interface UserRowProps {
  user: UserListItem;
  isAdmin: boolean;
  onEditRole: (user: UserListItem) => void;
  onToggleStatus: (user: UserListItem) => void;
  onDelete: (user: UserListItem) => void;
  t: ReturnType<typeof useI18n>["t"];
  isMobile?: boolean;
}

export function UserRow({ user, isAdmin, onEditRole, onToggleStatus, onDelete, t, isMobile }: UserRowProps) {
  const config = roleConfig[user.role as keyof typeof roleConfig] || roleConfig[UserRole.VIEWER];
  const RoleIcon = config.icon;

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return "-";
    return new Date(dateStr).toLocaleString();
  };

  const formatShortDate = (dateStr: string | null) => {
    if (!dateStr) return "-";
    const date = new Date(dateStr);
    return `${date.getMonth() + 1}/${date.getDate()} ${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
  };

  // 移动端卡片视图
  if (isMobile) {
    return (
      <div className="p-3 hover:bg-[var(--background)] transition-colors">
        {/* 头部：用户信息 + 状态标签 */}
        <div className="flex items-start justify-between gap-3 mb-2">
          <div className="flex items-center gap-2.5 min-w-0 flex-1">
            <div className="w-9 h-9 rounded-full bg-primary/20 flex items-center justify-center flex-shrink-0">
              <RoleIcon className="w-4 h-4 text-primary" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="font-medium text-default text-sm truncate">{user.username}</div>
              {user.email && (
                <div className="text-xs text-muted truncate">{user.email}</div>
              )}
            </div>
          </div>
          <div className="flex items-center gap-1.5 flex-shrink-0">
            <span className={`inline-flex px-1.5 py-0.5 text-[10px] font-medium rounded-full ${config.color}`}>
              {config.label}
            </span>
            <span className={`inline-flex px-1.5 py-0.5 text-[10px] font-medium rounded-full ${
              user.status === 1
                ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
            }`}>
              {user.status === 1 ? t.users.statusActive : t.users.statusDisabled}
            </span>
          </div>
        </div>

        {/* 中部：时间信息 */}
        <div className="flex items-center gap-4 text-xs text-muted mb-2">
          <span>{t.users.createdAt}: {formatShortDate(user.createdAt)}</span>
          <span>{t.users.lastLogin}: {formatShortDate(user.lastLogin)}</span>
        </div>

        {/* 底部：操作按钮 */}
        {isAdmin && (
          <div className="flex items-center gap-1 pt-2 border-t border-[var(--border-color)]">
            <button
              onClick={() => onEditRole(user)}
              className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs text-muted hover:text-primary hover:bg-primary/10 rounded-lg transition-colors"
            >
              <Edit2 className="w-3.5 h-3.5" />
              {t.users.changeRole}
            </button>
            {user.username !== "admin" && (
              <button
                onClick={() => onToggleStatus(user)}
                className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs rounded-lg transition-colors ${
                  user.status === 1
                    ? "text-muted hover:text-yellow-500 hover:bg-yellow-500/10"
                    : "text-muted hover:text-green-500 hover:bg-green-500/10"
                }`}
              >
                {user.status === 1 ? (
                  <>
                    <PowerOff className="w-3.5 h-3.5" />
                    {t.users.disable}
                  </>
                ) : (
                  <>
                    <Power className="w-3.5 h-3.5" />
                    {t.users.enable}
                  </>
                )}
              </button>
            )}
            {user.username !== "admin" && (
              <button
                onClick={() => onDelete(user)}
                className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs text-muted hover:text-red-500 hover:bg-red-500/10 rounded-lg transition-colors ml-auto"
              >
                <Trash2 className="w-3.5 h-3.5" />
                {t.common.delete}
              </button>
            )}
          </div>
        )}
      </div>
    );
  }

  // 桌面端表格行
  return (
    <tr className="hover:bg-[var(--background)]">
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
            <RoleIcon className="w-4 h-4 text-primary" />
          </div>
          <div>
            <div className="font-medium text-default">{user.username}</div>
            {user.displayName && (
              <div className="text-xs text-muted">{user.displayName}</div>
            )}
          </div>
        </div>
      </td>
      <td className="px-4 py-3 text-sm text-secondary">
        {user.email || "-"}
      </td>
      <td className="px-4 py-3">
        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${config.color}`}>
          {config.label}
        </span>
      </td>
      <td className="px-4 py-3">
        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
          user.status === 1
            ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
            : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
        }`}>
          {user.status === 1 ? t.users.statusActive : t.users.statusDisabled}
        </span>
      </td>
      <td className="px-4 py-3 text-sm text-secondary">
        {formatDate(user.createdAt)}
      </td>
      <td className="px-4 py-3 text-sm text-secondary">
        <div>{formatDate(user.lastLogin)}</div>
        {user.lastLoginIP && (
          <div className="text-xs text-muted">{user.lastLoginIP}</div>
        )}
      </td>
      {isAdmin && (
        <td className="px-4 py-3">
          <div className="flex items-center gap-1">
            <button
              onClick={() => onEditRole(user)}
              className="p-2 hover-bg rounded-lg"
              title={t.users.changeRole}
            >
              <Edit2 className="w-4 h-4 text-muted hover:text-primary" />
            </button>
            {user.username !== "admin" && (
              <button
                onClick={() => onToggleStatus(user)}
                className="p-2 hover-bg rounded-lg"
                title={user.status === 1 ? t.users.disable : t.users.enable}
              >
                {user.status === 1 ? (
                  <PowerOff className="w-4 h-4 text-muted hover:text-yellow-500" />
                ) : (
                  <Power className="w-4 h-4 text-muted hover:text-green-500" />
                )}
              </button>
            )}
            {user.username !== "admin" && (
              <button
                onClick={() => onDelete(user)}
                className="p-2 hover-bg rounded-lg"
                title={t.users.deleteUser}
              >
                <Trash2 className="w-4 h-4 text-muted hover:text-red-500" />
              </button>
            )}
          </div>
        </td>
      )}
    </tr>
  );
}
