/**
 * 深度分析轮询决策（纯函数）
 *
 * 后端按事件记录最近一次分析的状态（GET /api/v2/aiops/ai/analyze/status）。
 * 前端每次拿到状态后据此决定：继续等 / 完成 / 以哪种原因结束。
 */

export type AnalysisState = "running" | "succeeded" | "failed" | "none";

export interface AnalysisStatus {
  incidentId: string;
  state: AnalysisState;
  errorCode?: string;
  startedAt?: string;
  finishedAt?: string;
}

/** 后端分析时限 5 分钟，前端多等 1 分钟余量 */
export const ANALYSIS_POLL_MAX_MS = 6 * 60 * 1000;
export const ANALYSIS_POLL_INTERVAL_MS = 5000;

export type PollDecision =
  | { kind: "continue" }
  | { kind: "succeeded" }
  | { kind: "failed"; code: string };

/**
 * @param status 本次查询结果
 * @param elapsedMs 自触发起已等待的时间
 */
export function decideAnalysisPoll(status: AnalysisStatus, elapsedMs: number): PollDecision {
  switch (status.state) {
    case "succeeded":
      return { kind: "succeeded" };
    case "failed":
      return { kind: "failed", code: status.errorCode || "ai_upstream" };
    case "none":
      // 触发过却查不到：Master 重启，分析已随之中断
      return { kind: "failed", code: "analysis_interrupted" };
    case "running":
    default:
      return elapsedMs >= ANALYSIS_POLL_MAX_MS ? { kind: "failed", code: "ai_timeout" } : { kind: "continue" };
  }
}

// ============================================================
// 一次分析从「看到它在跑」到「看到它结束」的跟踪
// ============================================================

/**
 * 前端对一次深度分析的跟踪状态。
 *
 * 后端只记「最近一次分析的状态」，光看它分不清「刚刚跑完」与「很久以前跑完的」。
 * 所以前端自己记：只有亲眼看到过 running，之后的结束才算数、才提示结果。
 */
export type AnalysisWatch =
  | { phase: "idle" }
  | { phase: "running"; startedAtMs: number }
  | { phase: "succeeded" }
  | { phase: "failed"; errorCode: string };

export const IDLE_ANALYSIS_WATCH: AnalysisWatch = { phase: "idle" };

/** 后端给的开始时间解析不出来时，退回到「现在」 */
function parseStartedAt(startedAt: string | undefined, nowMs: number): number {
  const parsed = startedAt ? Date.parse(startedAt) : Number.NaN;
  return Number.isNaN(parsed) ? nowMs : parsed;
}

/**
 * 拿到一次新的状态后，跟踪状态怎么走。
 *
 * - 之前没在跟：后端说在跑就开始跟（打开详情时分析仍在进行、或别处触发了分析）；
 *   其余状态都是过去的事，不提示
 * - 之前在跟：按 decideAnalysisPoll 判定继续等 / 成功 / 失败（含超时与中断）
 */
export function advanceAnalysisWatch(
  previous: AnalysisWatch | undefined,
  status: AnalysisStatus,
  nowMs: number,
): AnalysisWatch {
  if (previous?.phase !== "running") {
    return status.state === "running"
      ? { phase: "running", startedAtMs: parseStartedAt(status.startedAt, nowMs) }
      : IDLE_ANALYSIS_WATCH;
  }
  const decision = decideAnalysisPoll(status, nowMs - previous.startedAtMs);
  if (decision.kind === "continue") return previous;
  if (decision.kind === "succeeded") return { phase: "succeeded" };
  return { phase: "failed", errorCode: decision.code };
}

/**
 * 状态接口这一次没查到（网络抖动、Master 重启中）时，跟踪状态怎么走。
 *
 * 正在跟的分析继续等，直到超过前端时限才按超时结束；返回 null 表示这次失败不该被吞掉。
 */
export function analysisWatchAfterStatusError(previous: AnalysisWatch | undefined, nowMs: number): AnalysisWatch | null {
  if (previous?.phase !== "running") return null;
  return nowMs - previous.startedAtMs >= ANALYSIS_POLL_MAX_MS ? { phase: "failed", errorCode: "ai_timeout" } : previous;
}

/** 多久之后再查一次；false = 不再查（没在跟，或已经有结果） */
export function analysisWatchPollInterval(watch: AnalysisWatch | undefined): number | false {
  return watch?.phase === "running" ? ANALYSIS_POLL_INTERVAL_MS : false;
}
