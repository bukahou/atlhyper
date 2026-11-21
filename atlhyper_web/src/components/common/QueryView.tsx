"use client";

import type { ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { LoadingSpinner } from "./LoadingSpinner";
import { NoClusterState } from "./NoClusterState";
import { resolveViewState } from "./query-view-state";
import { observeFailureReason } from "@/lib/observe-failure";

interface QueryViewProps<T> {
  query: UseQueryResult<T>;
  /** 这块内容按集群取数：没有集群时显示统一的占位 */
  requireCluster?: boolean;
  /** 判断数据是否为空；不传则不单独处理空状态，交给 children 自己显示 */
  isEmpty?: (data: T) => boolean;
  /** 数据为空时显示的内容 */
  empty?: ReactNode;
  children: (data: T) => ReactNode;
}

/**
 * 把一个查询的状态显示出来：无集群 / 加载中 / 出错 / 刷新失败 / 空 / 就绪。
 *
 * 各页面不再各写一套 loading 与 error 分支。判定规则见 query-view-state.ts。
 */
export function QueryView<T>({ query, requireCluster = false, isEmpty, empty, children }: QueryViewProps<T>) {
  const { t } = useI18n();
  const { hasCluster, listLoaded } = useCurrentCluster();
  const data = query.data;
  const hasData = data !== undefined;

  const state = resolveViewState({
    requireCluster,
    clusterListLoaded: listLoaded,
    hasCluster,
    hasData,
    isEmpty: hasData && isEmpty !== undefined && isEmpty(data),
    isPending: query.isPending,
    isError: query.isError,
    isRefetchError: query.isRefetchError,
  });

  // 观测接口失败时后端给出失败类别：附上原因；其他接口没有类别，只显示通用文案
  const failureReason = query.error ? observeFailureReason(query.error, t.observeFailure) : null;

  if (state === "no-cluster") return <NoClusterState />;
  if (state === "error") {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <AlertTriangle className="w-8 h-8 mb-3 text-red-500" />
        <p className={`text-sm text-default ${failureReason ? "mb-1" : "mb-3"}`}>{t.common.loadFailed}</p>
        {failureReason && <p className="text-xs text-muted mb-3 max-w-md">{failureReason}</p>}
        <button
          onClick={() => void query.refetch()}
          className="px-4 py-2 text-sm rounded-lg bg-primary hover:bg-primary-hover text-white transition-colors"
        >
          {t.common.retry}
        </button>
      </div>
    );
  }
  if (state === "loading" || data === undefined) return <LoadingSpinner />;

  return (
    <>
      {state === "stale" && (
        <div className="mb-3 flex items-center gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-600 dark:text-amber-400">
          <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
          <span>
            {t.common.refreshFailed}
            {failureReason && `：${failureReason}`}
          </span>
        </div>
      )}
      {state === "empty" && empty !== undefined ? empty : children(data)}
    </>
  );
}
