/**
 * 等一条指令执行到终态。
 *
 * 写操作只是把指令放进队列，由 Agent 稍后执行：按指令状态等到成功 / 失败 / 超时再刷新，
 * 代替「固定等 2 秒再刷新」（Agent 慢时刷不出新状态，执行失败前端也不知道）。
 * 不依赖 React，便于测试；hook 见 hooks/useCommandSettle。
 */
import type { CommandStatus } from "@/api/commands";

export interface CommandOutcome {
  /** 终态；waited = 超过等待上限仍未结束（指令可能还在执行） */
  status: "success" | "failed" | "timeout" | "waited";
  /** 失败原因（Agent 回报的） */
  error?: string;
}

export interface WaitOptions {
  intervalMs?: number;
  timeoutMs?: number;
  signal?: AbortSignal;
  /** 测试注入 */
  sleep?: (ms: number, signal?: AbortSignal) => Promise<void>;
  now?: () => number;
}

const TERMINAL = new Set<CommandStatus["status"]>(["success", "failed", "timeout"]);

function defaultSleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener("abort", () => {
      clearTimeout(timer);
      reject(signal.reason);
    }, { once: true });
  });
}

/**
 * 轮询指令状态直到终态。查询本身出错（网络抖动）不算终态，继续等到上限。
 * signal 中止时抛出（调用方不再需要结果，例如组件已卸载）。
 */
export async function waitForCommand(
  commandId: string,
  fetchStatus: (commandId: string, signal?: AbortSignal) => Promise<CommandStatus>,
  { intervalMs = 500, timeoutMs = 30_000, signal, sleep = defaultSleep, now = Date.now }: WaitOptions = {},
): Promise<CommandOutcome> {
  const deadline = now() + timeoutMs;
  for (;;) {
    signal?.throwIfAborted();
    try {
      const current = await fetchStatus(commandId, signal);
      if (TERMINAL.has(current.status)) {
        const status = current.status as CommandOutcome["status"];
        return status === "success" ? { status } : { status, error: current.result?.error };
      }
    } catch (err) {
      if (signal?.aborted) throw err;
    }
    if (now() >= deadline) return { status: "waited" };
    await sleep(intervalMs, signal);
  }
}

/** 生成提示用到的文案（取自 t.commands） */
export interface OutcomeTexts {
  outcomeFailed: string;
  outcomeStillRunning: string;
  statuses: { failed: string; timeout: string };
}

/** 指令没有成功完成时给操作者看的提示；成功返回 null */
export function commandOutcomeMessage(outcome: CommandOutcome, texts: OutcomeTexts): string | null {
  switch (outcome.status) {
    case "success":
      return null;
    case "waited":
      return texts.outcomeStillRunning;
    default:
      return texts.outcomeFailed.replace("{error}", outcome.error || texts.statuses[outcome.status]);
  }
}
