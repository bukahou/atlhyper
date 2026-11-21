import { describe, expect, it } from "vitest";
import { commandOutcomeMessage, waitForCommand } from "./command-settle";
import type { CommandStatus } from "@/api/commands";

function scripted(...statuses: (CommandStatus["status"] | Error)[]) {
  let i = 0;
  return async (commandId: string): Promise<CommandStatus> => {
    const next = statuses[Math.min(i++, statuses.length - 1)];
    if (next instanceof Error) throw next;
    return { commandId, status: next, result: next === "failed" ? { success: false, error: "forbidden" } : undefined };
  };
}

const instant = { sleep: async () => {}, intervalMs: 1 };

describe("waitForCommand", () => {
  it("等到成功才返回（中途的 pending / running 不算）", async () => {
    await expect(waitForCommand("c1", scripted("pending", "running", "success"), instant)).resolves.toEqual({ status: "success" });
  });

  it("失败时带回 Agent 的原因", async () => {
    await expect(waitForCommand("c1", scripted("running", "failed"), instant)).resolves.toEqual({ status: "failed", error: "forbidden" });
  });

  it("查询出错不算终态，继续等", async () => {
    await expect(waitForCommand("c1", scripted(new Error("network"), "success"), instant)).resolves.toEqual({ status: "success" });
  });

  it("超过等待上限仍未结束：返回 waited", async () => {
    let t = 0;
    const outcome = await waitForCommand("c1", scripted("running"), { ...instant, timeoutMs: 10, now: () => (t += 5) });
    expect(outcome).toEqual({ status: "waited" });
  });

  it("中止时抛出，不再返回结果", async () => {
    const controller = new AbortController();
    controller.abort();
    await expect(waitForCommand("c1", scripted("running"), { ...instant, signal: controller.signal })).rejects.toBeDefined();
  });
});

describe("commandOutcomeMessage", () => {
  const texts = {
    outcomeFailed: "failed: {error}",
    outcomeStillRunning: "still running",
    statuses: { failed: "Failed", timeout: "Timed out" },
  };

  it("成功不提示", () => {
    expect(commandOutcomeMessage({ status: "success" }, texts)).toBeNull();
  });

  it("失败带上 Agent 回报的原因", () => {
    expect(commandOutcomeMessage({ status: "failed", error: "forbidden" }, texts)).toBe("failed: forbidden");
  });

  it("没有原因时用状态名，⛔ 不留空", () => {
    expect(commandOutcomeMessage({ status: "timeout" }, texts)).toBe("failed: Timed out");
  });

  it("等到上限仍未结束：提示稍后查看", () => {
    expect(commandOutcomeMessage({ status: "waited" }, texts)).toBe("still running");
  });
});
