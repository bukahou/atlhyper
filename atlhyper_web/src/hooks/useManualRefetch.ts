/**
 * 手动刷新
 *
 * 页头的刷新按钮只在用户点它的时候转圈；后台轮询是静默的，不该让按钮跟着闪。
 * 所以不用 useIsFetching（它分不清是谁触发的），自己记「这一次是用户点的」。
 */
import { useCallback, useState } from "react";
import { useQueryClient, type QueryKey } from "@tanstack/react-query";

/**
 * 返回刷新函数与「用户触发的刷新是否还在进行」。
 *
 * 只重取当前页面上正在显示的查询（键以 queryKeyPrefix 开头的）。
 * queryKeyPrefix 要用 queries/keys.ts 里的常量（如 apmKeys.all），不要每次渲染新建数组。
 */
export function useManualRefetch(queryKeyPrefix: QueryKey) {
  const queryClient = useQueryClient();
  const [isRefetching, setIsRefetching] = useState(false);

  const refetch = useCallback(() => {
    setIsRefetching(true);
    void queryClient
      .refetchQueries({ queryKey: queryKeyPrefix, type: "active" })
      .finally(() => setIsRefetching(false));
  }, [queryClient, queryKeyPrefix]);

  return { refetch, isRefetching };
}
