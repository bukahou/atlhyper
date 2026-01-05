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

export function useAuthError() {
  const { openLoginDialog } = useAuthStore();
  const [lastError, setLastError] = useState<AuthError | null>(null);

  const handleAuthError = useCallback(
    (error: AuthError) => {
      setLastError(error);

      if (error.type === "unauthorized") {
        // 401: 需要登录
        openLoginDialog();
      } else if (error.type === "forbidden") {
        // 403: 权限不足 - 可以在这里添加 toast 提示
        // 目前只记录错误，具体 UI 反馈由调用方决定
        console.warn("[Auth] 权限不足:", error.message);
      }
    },
    [openLoginDialog]
  );

  useEffect(() => {
    // 订阅权限错误事件
    const unsubscribe = authErrorManager.subscribe(handleAuthError);
    return unsubscribe;
  }, [handleAuthError]);

  return { lastError };
}

/**
 * 权限错误边界 Hook
 *
 * 用于包装 API 调用，统一处理权限错误
 */
export function useApiCall() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const execute = useCallback(async <T>(
    apiCall: () => Promise<T>,
    options?: {
      onSuccess?: (data: T) => void;
      onError?: (error: Error) => void;
    }
  ): Promise<T | null> => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiCall();
      options?.onSuccess?.(result);
      return result;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "操作失败";
      setError(errorMessage);
      options?.onError?.(err as Error);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return { execute, loading, error };
}
