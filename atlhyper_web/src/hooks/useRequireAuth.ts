/**
 * 需要登录的操作 Hook
 *
 * 用于包装需要登录才能执行的操作
 * 如果未登录，先弹出登录框，登录成功后自动执行操作
 */

import { useCallback } from "react";
import { useAuthStore } from "@/store/authStore";

/**
 * 返回一个包装函数，用于在执行操作前检查登录状态
 *
 * @example
 * const requireAuth = useRequireAuth();
 *
 * const handleRestart = (pod: PodItem) => {
 *   requireAuth(() => {
 *     setRestartTarget(pod);
 *   });
 * };
 */
export function useRequireAuth() {
  const { isAuthenticated, openLoginDialog } = useAuthStore();

  const requireAuth = useCallback(
    (action: () => void) => {
      if (!isAuthenticated) {
        openLoginDialog(action);
        return false;
      }
      action();
      return true;
    },
    [isAuthenticated, openLoginDialog]
  );

  return requireAuth;
}
