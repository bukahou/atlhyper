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
