import { describe, expect, it } from "vitest";
import { demoPage, formatTimestamp } from "./format";

describe("formatTimestamp", () => {
  it("空串表示「没有」，显示为 -", () => {
    expect(formatTimestamp("")).toBe("-");
  });

  it("有值时按本地时间显示", () => {
    const iso = "2026-11-15T11:00:00Z";
    expect(formatTimestamp(iso)).toBe(new Date(iso).toLocaleString());
  });
});

describe("demoPage", () => {
  const items = [1, 2, 3, 4, 5];

  it("切出指定的一页，总数是整份列表的长度", () => {
    expect(demoPage(items, 0, 2)).toEqual({ items: [1, 2], total: 5 });
    expect(demoPage(items, 1, 2)).toEqual({ items: [3, 4], total: 5 });
  });

  it("最后一页不满、页码超出范围时不报错", () => {
    expect(demoPage(items, 2, 2)).toEqual({ items: [5], total: 5 });
    expect(demoPage(items, 9, 2)).toEqual({ items: [], total: 5 });
  });

  it("空列表", () => {
    expect(demoPage([], 0, 20)).toEqual({ items: [], total: 0 });
  });
});
