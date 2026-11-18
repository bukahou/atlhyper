import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createDelayedTaskScheduler } from "./useDelayedInvalidate";

// hook 本身只是把这份簿记接到组件生命周期上（卸载时 cancelAll），
// 定时与清理的逻辑都在这个纯对象里，不需要 DOM 就能测。
describe("createDelayedTaskScheduler", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("到时间才执行，且只执行一次", () => {
    const scheduler = createDelayedTaskScheduler();
    const task = vi.fn();

    scheduler.schedule(task, 2000);
    vi.advanceTimersByTime(1999);
    expect(task).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(task).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(10_000);
    expect(task).toHaveBeenCalledTimes(1);
  });

  it("取消后，还没到时间的任务都不再执行（组件卸载时用）", () => {
    const scheduler = createDelayedTaskScheduler();
    const first = vi.fn();
    const second = vi.fn();

    scheduler.schedule(first, 2000);
    scheduler.schedule(second, 3000);
    scheduler.cancelAll();
    vi.advanceTimersByTime(10_000);

    expect(first).not.toHaveBeenCalled();
    expect(second).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("多个任务互不影响；已执行的不占着定时器", () => {
    const scheduler = createDelayedTaskScheduler();
    const first = vi.fn();
    const second = vi.fn();

    scheduler.schedule(first, 1000);
    scheduler.schedule(second, 2000);
    vi.advanceTimersByTime(1000);
    expect(first).toHaveBeenCalledTimes(1);
    expect(second).not.toHaveBeenCalled();
    expect(scheduler.pendingCount()).toBe(1);

    vi.advanceTimersByTime(1000);
    expect(second).toHaveBeenCalledTimes(1);
    expect(scheduler.pendingCount()).toBe(0);
  });

  it("取消之后还能继续安排新任务（开发模式下 effect 会先清理再重新挂载）", () => {
    const scheduler = createDelayedTaskScheduler();
    scheduler.cancelAll();
    const task = vi.fn();

    scheduler.schedule(task, 500);
    vi.advanceTimersByTime(500);

    expect(task).toHaveBeenCalledTimes(1);
  });
});
