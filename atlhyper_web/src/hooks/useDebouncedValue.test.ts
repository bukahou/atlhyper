import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createDebouncer } from "./useDebouncedValue";

// hook 本身只是把这个去抖器接到组件生命周期上，去抖的逻辑都在纯对象里，不需要 DOM 就能测。
describe("createDebouncer", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("停止输入满一段时间后才交出值", () => {
    const settle = vi.fn();
    const debouncer = createDebouncer<string>(settle, 300);

    debouncer.push("t");
    vi.advanceTimersByTime(299);
    expect(settle).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(settle).toHaveBeenCalledTimes(1);
    expect(settle).toHaveBeenCalledWith("t");
  });

  it("连续输入只交出最后一个值：中间的每个字符都不会变成一次请求", () => {
    const settle = vi.fn();
    const debouncer = createDebouncer<string>(settle, 300);

    debouncer.push("t");
    vi.advanceTimersByTime(100);
    debouncer.push("ti");
    vi.advanceTimersByTime(100);
    debouncer.push("timeout");
    vi.advanceTimersByTime(299);
    expect(settle).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(settle).toHaveBeenCalledTimes(1);
    expect(settle).toHaveBeenCalledWith("timeout");
  });

  it("取消后不再交出还在等待的值（组件卸载时用）", () => {
    const settle = vi.fn();
    const debouncer = createDebouncer<string>(settle, 300);

    debouncer.push("timeout");
    debouncer.cancel();
    vi.advanceTimersByTime(10_000);

    expect(settle).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });
});
