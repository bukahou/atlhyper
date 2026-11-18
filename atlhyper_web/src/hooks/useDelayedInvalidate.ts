/**
 * 延时让查询失效
 *
 * 写操作（重启、扩缩容、封锁节点等）只是把指令放进队列，由 Agent 稍后执行。
 * 指令下发成功的那一刻去刷新，多半拿到的还是旧状态，所以要晚一点再刷新一次。
 */
import { useCallback, useEffect, useState } from "react";
import { useQueryClient, type QueryKey } from "@tanstack/react-query";

/** 指令从下发到在快照里看得到结果，大约需要的时间 */
export const COMMAND_SETTLE_DELAY_MS = 2000;

export interface DelayedTaskScheduler {
  /** 过 delayMs 毫秒后执行一次 task */
  schedule: (task: () => void, delayMs: number) => void;
  /** 取消所有还没到时间的任务 */
  cancelAll: () => void;
  /** 还没到时间的任务数 */
  pendingCount: () => number;
}

/**
 * 定时任务的簿记：记住每个还没触发的定时器，以便一次全部取消。
 *
 * 从 hook 里抽出来是为了不依赖 React 与 DOM 就能测定时与清理。
 */
export function createDelayedTaskScheduler(): DelayedTaskScheduler {
  const pendingTimers = new Set<ReturnType<typeof setTimeout>>();
  return {
    schedule(task, delayMs) {
      const timer = setTimeout(() => {
        pendingTimers.delete(timer);
        task();
      }, delayMs);
      pendingTimers.add(timer);
    },
    cancelAll() {
      for (const timer of pendingTimers) clearTimeout(timer);
      pendingTimers.clear();
    },
    pendingCount: () => pendingTimers.size,
  };
}

/**
 * 返回一个函数：调用后过 delayMs 毫秒，让给定的查询键（前缀匹配）失效并重新取数。
 *
 * 组件卸载时清理还没触发的定时器 —— 页面都离开了，没有谁还需要那次刷新；
 * 再回来时查询会自己重新取数。
 *
 * @example
 * const invalidateAfterCommand = useDelayedInvalidate();
 * restart.mutate(pod, { onSuccess: () => invalidateAfterCommand(clusterKeys.resourceScope("pod")) });
 */
export function useDelayedInvalidate(delayMs: number = COMMAND_SETTLE_DELAY_MS) {
  const queryClient = useQueryClient();
  // 用 useState 的惰性初始化拿一个随组件存活的簿记对象
  const [scheduler] = useState(createDelayedTaskScheduler);

  useEffect(() => () => scheduler.cancelAll(), [scheduler]);

  return useCallback(
    (queryKeys: readonly QueryKey[]) => {
      scheduler.schedule(() => {
        for (const queryKey of queryKeys) {
          void queryClient.invalidateQueries({ queryKey });
        }
      }, delayMs);
    },
    [queryClient, scheduler, delayMs],
  );
}
