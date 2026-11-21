import { describe, expect, it } from "vitest";
import type { ObserveFailureTranslations } from "@/types/i18n";
import { observeFailureCode, observeFailureReason } from "./observe-failure";

const texts: ObserveFailureTranslations = {
  notConfigured: "not-configured",
  invalidParams: "invalid-params",
  timeout: "timeout",
  unavailable: "unavailable",
  rejected: "rejected",
  failed: "failed",
};

function axiosLikeError(data: unknown) {
  return Object.assign(new Error("Request failed"), { response: { status: 503, data } });
}

describe("observeFailureCode", () => {
  it("取出规格里定义的失败类别", () => {
    expect(observeFailureCode(axiosLikeError({ error: "x", code: "timeout" }))).toBe("timeout");
  });

  it("不是观测失败（没有 code、别的 code、不是请求错误）时为 null", () => {
    expect(observeFailureCode(axiosLikeError({ error: "x" }))).toBeNull();
    expect(observeFailureCode(axiosLikeError({ error: "x", code: "ai_auth" }))).toBeNull();
    expect(observeFailureCode(new Error("network"))).toBeNull();
    expect(observeFailureCode(null)).toBeNull();
  });
});

describe("observeFailureReason", () => {
  it("每个类别都有自己的提示", () => {
    const codes = ["not_configured", "invalid_params", "timeout", "unavailable", "rejected", "failed"];
    const reasons = codes.map((code) => observeFailureReason(axiosLikeError({ code }), texts));
    expect(reasons).toEqual(["not-configured", "invalid-params", "timeout", "unavailable", "rejected", "failed"]);
  });

  it("⛔ 不显示后端给的错误文字", () => {
    const reason = observeFailureReason(axiosLikeError({ error: "dial tcp 10.0.0.9:9000", code: "unavailable" }), texts);
    expect(reason).toBe("unavailable");
  });

  it("不是观测失败时返回 null（调用方只显示通用的「加载失败」）", () => {
    expect(observeFailureReason(axiosLikeError({ error: "x" }), texts)).toBeNull();
  });
});
