/**
 * 延时任务的簿记
 *
 * 部署页「立即同步」只是提交了一个同步任务，没有可轮询的指令 ID，只能过几秒再刷新状态与历史。
 * 集群写操作（重启、扩缩容、封锁节点等）有指令 ID，按指令状态刷新，见 useCommandSettle。
 */

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
