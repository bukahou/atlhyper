import { describe, it, expect } from "vitest";
import { aiErrorMessage, aiCodeMessage } from "./ai-error";

const texts = { analysisFailed: "generic", errorRoleUnassigned: "role", errorBudgetExhausted: "budget", errorUpstream: "upstream", errorTimeout: "timeout", analysisInterrupted: "interrupted" };
const axiosErr = (code?: string) => ({ response: { data: { code } } });

describe("aiErrorMessage", () => {
  it("按错误码给出可操作提示", () => {
    expect(aiErrorMessage(axiosErr("ai_role_unassigned"), texts)).toBe("role");
    expect(aiErrorMessage(axiosErr("ai_budget_exhausted"), texts)).toBe("budget");
    expect(aiErrorMessage(axiosErr("ai_upstream"), texts)).toBe("upstream");
    expect(aiErrorMessage(axiosErr("ai_timeout"), texts)).toBe("timeout");
  });
  it("未知码 / 无响应 → 通用文案", () => {
    expect(aiErrorMessage(axiosErr("weird"), texts)).toBe("generic");
    expect(aiErrorMessage(new Error("network"), texts)).toBe("generic");
    expect(aiErrorMessage(undefined, texts)).toBe("generic");
  });
});

describe("aiCodeMessage", () => {
  it("深度分析状态的错误码", () => {
    expect(aiCodeMessage("analysis_interrupted", texts)).toBe("interrupted");
    expect(aiCodeMessage("ai_role_unassigned", texts)).toBe("role");
  });
});
