"use client";

import { User, LogIn, LogOut, Settings } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { UserRole } from "@/types/auth";

// 角色显示名称
const getRoleName = (role: number): string => {
  switch (role) {
    case UserRole.ADMIN:
      return "Admin";
    case UserRole.OPERATOR:
      return "Operator";
    case UserRole.VIEWER:
      return "Viewer";
    default:
      return "User";
  }
};

export function UserMenu() {
  const { t } = useI18n();
  const { isAuthenticated, user, openLoginDialog, logout } = useAuthStore();

  if (!isAuthenticated) {
    return (
      <button
        onClick={() => openLoginDialog()}
        className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium text-primary hover:bg-primary/10 transition-colors"
      >
        <LogIn className="w-4 h-4" />
        {t.common.login}
      </button>
    );
  }

  return (
    <div className="relative group">
      <button className="flex items-center gap-2 p-2 rounded-lg hover-bg">
        <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
          <User className="w-4 h-4 text-primary" />
        </div>
        <span className="text-sm font-medium text-secondary hidden md:block">
          {user?.displayName || user?.username || "User"}
        </span>
      </button>
      <div className="absolute right-0 mt-2 w-48 dropdown-menu rounded-lg shadow-lg border opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50">
        <div className="px-4 py-3 border-b border-[var(--border-color)]">
          <p className="text-sm font-medium text-default">
            {user?.displayName || user?.username}
          </p>
          {user?.email && (
            <p className="text-xs text-secondary truncate">
              {user.email}
            </p>
          )}
          <p className="text-xs text-muted">
            {user ? getRoleName(user.role) : ""}
          </p>
        </div>
        <button
          className="w-full px-4 py-2 text-left text-sm text-secondary hover-bg flex items-center gap-2"
        >
          <Settings className="w-4 h-4" />
          Settings
        </button>
        <button
          onClick={logout}
          className="w-full px-4 py-2 text-left text-sm text-red-600 hover-bg flex items-center gap-2 rounded-b-lg"
        >
          <LogOut className="w-4 h-4" />
          {t.common.logout}
        </button>
      </div>
    </div>
  );
}
