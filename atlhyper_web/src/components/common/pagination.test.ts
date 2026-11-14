import { describe, expect, it } from "vitest";
import { clampPage } from "./pagination";

describe("clampPage", () => {
  it("范围内的页码原样返回", () => {
    expect(clampPage(1, 5)).toBe(1);
    expect(clampPage(3, 5)).toBe(3);
    expect(clampPage(5, 5)).toBe(5);
  });

  it("数据变少后页码超出总页数：停在最后一页", () => {
    expect(clampPage(7, 3)).toBe(3);
  });

  it("没有数据时仍是第 1 页", () => {
    expect(clampPage(4, 0)).toBe(1);
  });

  it("页码不会小于 1", () => {
    expect(clampPage(0, 5)).toBe(1);
  });
});
