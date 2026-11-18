/**
 * 权限错误处理 Hook
 *
 * 用于监听后端返回的权限错误（401/403），并触发相应的 UI 反馈
 *
 * 使用方式：
 * 1. 在根组件（如 Layout）中调用 useAuthError() 监听全局权限错误
 * 2. 当后端返回 401 时，自动弹出登录对话框
 * 3. 当后端返回 403 时，显示权限不足提示
 */

import { useEffect, useCallback, useState } from "react";
import { authErrorManager, type AuthError } from "@/api/request";
import { useAuthStore } from "@/store/authStore";
import { toast } from "@/components/common/Toast";
import { useI18n } from "@/i18n/context";

export function useAuthError() {
  const { openLoginDialog, logout, refreshMe } = useAuthStore();
  const { t } = useI18n();
  const [lastError, setLastError] = useState<AuthError | null>(null);

  const handleAuthError = useCallback(
    (error: AuthError) => {
      setLastError(error);

      if (error.type === "unauthorized") {
        // 401: Token 过期或无效 — 先清理登录状态，再弹登录框
        logout();
        openLoginDialog();
      } else if (error.type === "forbidden") {
        // 403: 已登录但权限不足 —— 提示，并刷新权限（角色可能刚被调整）
        console.warn("[Auth] 权限不足:", error.message);
        toast.error(t.permission.forbidden);
        void refreshMe();
      }
    },
    [logout, openLoginDialog, refreshMe, t]
  );

  useEffect(() => {
    // 订阅权限错误事件
    const unsubscribe = authErrorManager.subscribe(handleAuthError);
    return unsubscribe;
  }, [handleAuthError]);

  return { lastError };
}
