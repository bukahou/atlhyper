import { describe, it, expect } from "vitest";
import { decideAnalysisPoll, ANALYSIS_POLL_MAX_MS } from "./analysis-poll";

const st = (state: "running" | "succeeded" | "failed" | "none", errorCode?: string) => ({ incidentId: "i", state, errorCode });

describe("decideAnalysisPoll", () => {
  it("进行中且未超时 → 继续等（不再 60 秒就放弃）", () => {
    expect(decideAnalysisPoll(st("running"), 61_000)).toEqual({ kind: "continue" });
  });
  it("成功 → 完成", () => {
    expect(decideAnalysisPoll(st("succeeded"), 11_000)).toEqual({ kind: "succeeded" });
  });
  it("失败 → 带后端错误码结束；缺码按模型调用失败处理", () => {
    expect(decideAnalysisPoll(st("failed", "ai_role_unassigned"), 1_000)).toEqual({ kind: "failed", code: "ai_role_unassigned" });
    expect(decideAnalysisPoll(st("failed"), 1_000)).toEqual({ kind: "failed", code: "ai_upstream" });
  });
  it("触发过却查不到 → 分析已中断（Master 重启）", () => {
    expect(decideAnalysisPoll(st("none"), 5_000)).toEqual({ kind: "failed", code: "analysis_interrupted" });
  });
  it("进行中但超过前端时限 → 超时结束", () => {
    expect(decideAnalysisPoll(st("running"), ANALYSIS_POLL_MAX_MS)).toEqual({ kind: "failed", code: "ai_timeout" });
  });
});

import {
  advanceAnalysisWatch,
  analysisWatchAfterStatusError,
  analysisWatchPollInterval,
  ANALYSIS_POLL_INTERVAL_MS,
  IDLE_ANALYSIS_WATCH,
} from "./analysis-poll";

describe("advanceAnalysisWatch", () => {
  const now = Date.parse("2026-11-14T10:00:00Z");

  it("之前没在跟：后端说在跑就开始跟，开始时间取后端给的", () => {
    const status = { incidentId: "i", state: "running" as const, startedAt: "2026-11-14T09:58:00Z" };
    expect(advanceAnalysisWatch(undefined, status, now)).toEqual({
      phase: "running",
      startedAtMs: Date.parse("2026-11-14T09:58:00Z"),
    });
    expect(advanceAnalysisWatch(IDLE_ANALYSIS_WATCH, st("running"), now)).toEqual({ phase: "running", startedAtMs: now });
  });

  it("之前没在跟：过去的成功 / 失败 / 没有记录都不提示", () => {
    expect(advanceAnalysisWatch(undefined, st("succeeded"), now)).toEqual(IDLE_ANALYSIS_WATCH);
    expect(advanceAnalysisWatch(undefined, st("failed", "ai_upstream"), now)).toEqual(IDLE_ANALYSIS_WATCH);
    expect(advanceAnalysisWatch(undefined, st("none"), now)).toEqual(IDLE_ANALYSIS_WATCH);
    // 上一次的失败提示已经显示过，再查到同一个失败不重复进入失败
    expect(advanceAnalysisWatch({ phase: "failed", errorCode: "ai_upstream" }, st("failed", "ai_upstream"), now)).toEqual(
      IDLE_ANALYSIS_WATCH,
    );
  });

  it("正在跟：仍在跑且没超时 → 原样继续", () => {
    const watching = { phase: "running" as const, startedAtMs: now - 60_000 };
    expect(advanceAnalysisWatch(watching, st("running"), now)).toBe(watching);
  });

  it("正在跟：成功 / 失败 / 中断 / 超时各自结束", () => {
    const watching = { phase: "running" as const, startedAtMs: now - 60_000 };
    expect(advanceAnalysisWatch(watching, st("succeeded"), now)).toEqual({ phase: "succeeded" });
    expect(advanceAnalysisWatch(watching, st("failed", "ai_role_unassigned"), now)).toEqual({
      phase: "failed",
      errorCode: "ai_role_unassigned",
    });
    expect(advanceAnalysisWatch(watching, st("none"), now)).toEqual({ phase: "failed", errorCode: "analysis_interrupted" });
    const overdue = { phase: "running" as const, startedAtMs: now - ANALYSIS_POLL_MAX_MS };
    expect(advanceAnalysisWatch(overdue, st("running"), now)).toEqual({ phase: "failed", errorCode: "ai_timeout" });
  });
});

describe("analysisWatchAfterStatusError", () => {
  const now = 1_000_000_000;

  it("正在跟：状态接口暂时不可用就继续等，超过时限才按超时结束", () => {
    const watching = { phase: "running" as const, startedAtMs: now - 60_000 };
    expect(analysisWatchAfterStatusError(watching, now)).toBe(watching);
    const overdue = { phase: "running" as const, startedAtMs: now - ANALYSIS_POLL_MAX_MS };
    expect(analysisWatchAfterStatusError(overdue, now)).toEqual({ phase: "failed", errorCode: "ai_timeout" });
  });

  it("没在跟：不吞掉这次失败", () => {
    expect(analysisWatchAfterStatusError(undefined, now)).toBeNull();
    expect(analysisWatchAfterStatusError(IDLE_ANALYSIS_WATCH, now)).toBeNull();
  });
});

describe("analysisWatchPollInterval", () => {
  it("只有正在跟的时候才轮询：完成、失败、超时即停", () => {
    expect(analysisWatchPollInterval({ phase: "running", startedAtMs: 0 })).toBe(ANALYSIS_POLL_INTERVAL_MS);
    expect(analysisWatchPollInterval(undefined)).toBe(false);
    expect(analysisWatchPollInterval(IDLE_ANALYSIS_WATCH)).toBe(false);
    expect(analysisWatchPollInterval({ phase: "succeeded" })).toBe(false);
    expect(analysisWatchPollInterval({ phase: "failed", errorCode: "ai_timeout" })).toBe(false);
  });
});
