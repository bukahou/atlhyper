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
