import { describe, it, expect } from "vitest";
import { aiErrorMessage } from "./ai-error";

const texts = { analysisFailed: "generic", errorRoleUnassigned: "role", errorBudgetExhausted: "budget", errorUpstream: "upstream" };
const axiosErr = (code?: string) => ({ response: { data: { code } } });

describe("aiErrorMessage", () => {
  it("按错误码给出可操作提示", () => {
    expect(aiErrorMessage(axiosErr("ai_role_unassigned"), texts)).toBe("role");
    expect(aiErrorMessage(axiosErr("ai_budget_exhausted"), texts)).toBe("budget");
    expect(aiErrorMessage(axiosErr("ai_upstream"), texts)).toBe("upstream");
  });
  it("未知码 / 无响应 → 通用文案", () => {
    expect(aiErrorMessage(axiosErr("weird"), texts)).toBe("generic");
    expect(aiErrorMessage(new Error("network"), texts)).toBe("generic");
    expect(aiErrorMessage(undefined, texts)).toBe("generic");
  });
});
