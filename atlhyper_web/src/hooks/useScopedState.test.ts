import { describe, expect, it } from "vitest";
import { resolveScopedValue } from "./useScopedState";

// 页码、brush 选区这类状态只在某一组筛选条件下有意义。
// 这里测的是「范围变了就回到默认值」的判定；hook 只是把它接到 useState 上。
describe("resolveScopedValue", () => {
  it("范围没变时用保存的值", () => {
    expect(resolveScopedValue({ scope: "search=a", value: 3 }, "search=a", 1)).toBe(3);
  });

  it("范围变了就回到默认值：换了筛选条件，页码回到第一页", () => {
    expect(resolveScopedValue({ scope: "search=a", value: 3 }, "search=b", 1)).toBe(1);
  });

  it("范围变回去时，之前的值不会复活成别的范围的值", () => {
    const held = { scope: "search=a", value: 3 };
    expect(resolveScopedValue(held, "search=b", 1)).toBe(1);
    expect(resolveScopedValue(held, "search=a", 1)).toBe(3);
  });

  it("值可以是 null（没有选区）", () => {
    expect(resolveScopedValue<[number, number] | null>({ scope: "a", value: [1, 2] }, "b", null)).toBeNull();
    expect(resolveScopedValue<[number, number] | null>({ scope: "a", value: [1, 2] }, "a", null)).toEqual([1, 2]);
  });
});
