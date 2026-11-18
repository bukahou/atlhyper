import { describe, expect, it } from "vitest";
import { formatDuration, formatShortTime } from "./incident-format";

describe("formatDuration", () => {
  it("不到一小时显示分钟数（四舍五入）", () => {
    expect(formatDuration(0, "分钟")).toBe("0 分钟");
    expect(formatDuration(89, "分钟")).toBe("1 分钟");
    expect(formatDuration(59 * 60, "分钟")).toBe("59 分钟");
  });

  it("一小时及以上显示小时与分钟", () => {
    expect(formatDuration(60 * 60, "分钟")).toBe("1h 0m");
    expect(formatDuration(2 * 60 * 60 + 5 * 60, "分钟")).toBe("2h 5m");
  });
});

describe("formatShortTime", () => {
  it("月/日 时:分，时与分补零", () => {
    // 用本地时间构造，测试不依赖运行机器的时区
    expect(formatShortTime(new Date(2026, 10, 5, 3, 7).toISOString())).toBe("11/5 03:07");
  });
});
