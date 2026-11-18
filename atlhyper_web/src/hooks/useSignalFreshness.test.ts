import { describe, expect, it } from "vitest";
import type { FreshnessResponse } from "@/types/observe";
import { pickSignalFreshness } from "./useSignalFreshness";

describe("pickSignalFreshness", () => {
  const freshness: FreshnessResponse = {
    collectorHealthy: true,
    signals: [
      { signal: "metrics", lagSeconds: 5, status: "live" },
      { signal: "traces", lagSeconds: 4000, status: "idle" },
    ],
  };

  it("挑出本页关心的那个信号", () => {
    expect(pickSignalFreshness(freshness, "traces")).toEqual({ signal: "traces", lagSeconds: 4000, status: "idle" });
  });

  it("接口里没有这个信号、或还没有数据（演示模式返回 null）时是 null，徽章显示「未知」", () => {
    expect(pickSignalFreshness(freshness, "logs")).toBeNull();
    expect(pickSignalFreshness(null, "metrics")).toBeNull();
    expect(pickSignalFreshness(undefined, "metrics")).toBeNull();
  });
});
