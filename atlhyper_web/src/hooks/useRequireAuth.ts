/**
 * 需要权限的操作 Hook
 *
 * 用于包装需要权限才能执行的操作（默认 cluster.operate）：
 * - 有权限：直接执行
 * - 访客：弹出登录框，登录成功后执行（后端仍会按权限校验）
 * - 已登录但权限不足：提示联系管理员
 */

import { useCallback } from "react";
import { useAuthStore } from "@/store/authStore";
import { useCan } from "@/hooks/useCan";
import { toast } from "@/components/common/Toast";
import { useI18n } from "@/i18n/context";
import type { Permission } from "@/types/auth";

/**
 * @example
 * const requireAuth = useRequireAuth();
 * requireAuth(() => setRestartTarget(pod));                  // 需要 cluster.operate
 * requireAuth(() => loadSecret(name), "secret.read");        // 指定权限
 */
export function useRequireAuth() {
  const { isAuthenticated, openLoginDialog } = useAuthStore();
  const can = useCan();
  const { t } = useI18n();

  const requireAuth = useCallback(
    (action: () => void, permission: Permission = "cluster.operate") => {
      if (can(permission)) {
        action();
        return true;
      }
      if (!isAuthenticated) {
        openLoginDialog(action);
        return false;
      }
      toast.error(t.permission.forbidden);
      return false;
    },
    [can, isAuthenticated, openLoginDialog, t]
  );

  return requireAuth;
}
