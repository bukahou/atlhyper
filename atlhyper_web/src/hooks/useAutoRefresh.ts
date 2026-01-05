/**
 * 自动刷新 Hook
 *
 * 提供定时刷新和手动刷新功能
 * 刷新间隔从环境变量 NEXT_PUBLIC_REFRESH_INTERVAL 读取
 */

import { useEffect, useCallback, useRef } from "react";
import { env } from "@/config/env";

interface UseAutoRefreshOptions {
  /** 是否启用自动刷新，默认 true */
  enabled?: boolean;
  /** 自定义刷新间隔（毫秒），默认使用环境变量 */
  interval?: number;
  /** 是否在挂载时立即执行一次，默认 true */
  immediate?: boolean;
}

/**
 * 自动刷新 Hook
 *
 * @param fetchFn 数据获取函数
 * @param options 配置选项
 * @returns { refresh, lastRefresh } 手动刷新函数和上次刷新时间
 *
 * @example
 * ```tsx
 * const { refresh, lastRefresh } = useAutoRefresh(fetchData);
 * ```
 */
export function useAutoRefresh(
  fetchFn: () => void | Promise<void>,
  options: UseAutoRefreshOptions = {}
) {
  const {
    enabled = true,
    interval = env.refreshInterval,
    immediate = true,
  } = options;

  const lastRefreshRef = useRef<Date | null>(null);
  const intervalIdRef = useRef<NodeJS.Timeout | null>(null);

  const refresh = useCallback(() => {
    lastRefreshRef.current = new Date();
    fetchFn();
  }, [fetchFn]);

  useEffect(() => {
    // 立即执行一次
    if (immediate) {
      refresh();
    }

    // 设置定时刷新
    if (enabled && interval > 0) {
      intervalIdRef.current = setInterval(refresh, interval);
    }

    // 清理
    return () => {
      if (intervalIdRef.current) {
        clearInterval(intervalIdRef.current);
        intervalIdRef.current = null;
      }
    };
  }, [enabled, interval, immediate, refresh]);

  return {
    /** 手动刷新 */
    refresh,
    /** 上次刷新时间 */
    lastRefresh: lastRefreshRef.current,
    /** 刷新间隔（秒） */
    intervalSeconds: Math.round(interval / 1000),
  };
}
