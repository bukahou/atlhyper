import { describe, expect, it } from "vitest";
import { formatRiskScore, isElevatedRisk, riskLevelColor } from "./risk";

describe("risk", () => {
  it("颜色只按后端给的等级取，不在前端按分数再判一次", () => {
    expect(riskLevelColor("critical")).toBe("#ef4444");
    expect(riskLevelColor("high")).toBe("#f97316");
    expect(riskLevelColor("medium")).toBe(riskLevelColor("warning"));
    expect(riskLevelColor("healthy")).toBe("#22c55e");
  });
  it("没有等级或不认识的等级按健康显示", () => {
    expect(riskLevelColor(undefined)).toBe("#22c55e");
    expect(riskLevelColor("whatever")).toBe("#22c55e");
  });
  it("高风险 = high / critical", () => {
    expect(isElevatedRisk("high")).toBe(true);
    expect(isElevatedRisk("critical")).toBe(true);
    expect(isElevatedRisk("medium")).toBe(false);
    expect(isElevatedRisk(undefined)).toBe(false);
  });
  it("分数取整显示", () => {
    expect(formatRiskScore(45.6)).toBe("46");
  });
});
