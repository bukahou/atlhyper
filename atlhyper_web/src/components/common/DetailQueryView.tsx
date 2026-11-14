"use client";

import type { ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { LoadingSpinner } from "./LoadingSpinner";
import { resolveViewState } from "./query-view-state";

interface DetailQueryViewProps<T> {
  /** 详情查询；数据为 null 表示这个资源不存在（演示数据里找不到时就是这样） */
  query: UseQueryResult<T | null>;
  /** 错误对象里没有可读信息时显示的文案 */
  errorFallback: string;
  children: (detail: T) => ReactNode;
}

/**
 * 详情抽屉里的查询状态：加载中 / 出错 / 刷新失败 / 就绪。
 *
 * 判定规则与 QueryView 共用（query-view-state.ts），样式沿用各详情弹窗原来的写法，
 * 16 个弹窗不再各写一套 loading 与 error 分支。详情一定是从某个集群的列表点进来的，
 * 所以这里没有「无集群」状态。
 */
export function DetailQueryView<T>({ query, errorFallback, children }: DetailQueryViewProps<T>) {
  const { t } = useI18n();
  const detail = query.data;

  const state = resolveViewState({
    requireCluster: false,
    clusterListLoaded: true,
    hasCluster: true,
    hasData: detail !== undefined,
    isEmpty: detail === null,
    isPending: query.isPending,
    isError: query.isError,
    isRefetchError: query.isRefetchError,
  });

  if (state === "error") {
    return (
      <div className="p-6 text-center text-red-500">
        {query.error instanceof Error ? query.error.message : errorFallback}
      </div>
    );
  }
  if (state === "loading" || detail === undefined) {
    return (
      <div className="py-12">
        <LoadingSpinner />
      </div>
    );
  }
  if (detail === null) return null;
  if (state !== "stale") return <>{children(detail)}</>;

  // 刷新失败：继续显示上一次的数据，顶部提示。外层撑满高度，内容区保持原来的滚动方式
  return (
    <div className="flex flex-col h-full">
      <div className="mx-4 mt-3 flex shrink-0 items-center gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-600 dark:text-amber-400">
        <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
        <span>{t.common.refreshFailed}</span>
      </div>
      <div className="flex-1 min-h-0">{children(detail)}</div>
    </div>
  );
}
