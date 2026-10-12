"use client";

/**
 * 权限锁定占位
 *
 * 真实数据页内，当前主体无权查看的部分统一用它占位（不显示演示数据）。
 * 访客：提示登录；已登录但不够：提示联系管理员提升。
 */

import { Lock } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { useCan } from "@/hooks/useCan";
import type { Permission } from "@/types/auth";
import { LoadingSpinner } from "./LoadingSpinner";

interface PermissionLockProps {
  permission: Permission;
  children: React.ReactNode;
  className?: string;
}

export function PermissionLock({ permission, children, className = "" }: PermissionLockProps) {
  const { t } = useI18n();
  const can = useCan();
  const { meLoaded, isAuthenticated, openLoginDialog } = useAuthStore();

  if (!meLoaded) {
    return (
      <div className={`flex items-center justify-center py-12 ${className}`}>
        <LoadingSpinner />
      </div>
    );
  }
  if (can(permission)) return <>{children}</>;

  const pt = t.permission;

  return (
    <div className={`flex flex-col items-center justify-center gap-3 py-12 px-4 text-center ${className}`}>
      <div className="w-12 h-12 rounded-full bg-[var(--background)] border border-[var(--border-color)] flex items-center justify-center">
        <Lock className="w-5 h-5 text-muted" />
      </div>
      <div className="text-sm font-medium text-default">{pt.lockedTitle}</div>
      <div className="text-xs text-muted max-w-md">{isAuthenticated ? pt.hintMember : pt.hintGuest}</div>
      {!isAuthenticated && (
        <button
          type="button"
          onClick={() => openLoginDialog()}
          className="mt-1 px-4 py-1.5 text-sm rounded-lg bg-primary text-white hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
        >
          {pt.loginButton}
        </button>
      )}
    </div>
  );
}
