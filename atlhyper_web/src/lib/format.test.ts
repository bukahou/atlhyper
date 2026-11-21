import { describe, expect, it } from "vitest";
import { formatTimestamp } from "./format";

describe("formatTimestamp", () => {
  it("空串表示「没有」，显示为 -", () => {
    expect(formatTimestamp("")).toBe("-");
  });

  it("有值时按浏览器本地的日期时间显示", () => {
    const iso = "2026-11-15T11:00:00Z";
    expect(formatTimestamp(iso)).toBe(new Date(iso).toLocaleString());
  });
});
