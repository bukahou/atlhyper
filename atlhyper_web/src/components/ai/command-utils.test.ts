import { describe, expect, it } from "vitest";
import { groupConversation, parseRoundsFromSegments } from "./command-utils";
import type { Message, StreamSegment } from "./types";

describe("parseRoundsFromSegments", () => {
  it("按 round 分轮、按 callId 配对：同一轮并行调用同名工具也不会配错，失败看 isError", () => {
    const segments: StreamSegment[] = [
      { type: "text", round: 1, content: "先查一下" },
      { type: "tool_call", round: 1, callId: "c1", tool: "query_cluster", params: "{}" },
      { type: "tool_call", round: 1, callId: "c2", tool: "query_cluster", params: "{}" },
      // 结果的到达顺序与调用顺序相反
      { type: "tool_result", round: 1, callId: "c2", tool: "query_cluster", content: "boom", isError: true },
      { type: "tool_result", round: 1, callId: "c1", tool: "query_cluster", content: "ok" },
      { type: "text", round: 2, content: "结论" },
    ];
    const { rounds, finalText } = parseRoundsFromSegments(segments);
    expect(rounds).toHaveLength(1);
    expect(rounds[0].thinking).toBe("先查一下");
    expect(rounds[0].commands.map((c) => [c.id, c.status, c.result])).toEqual([
      ["c1", "success", "ok"],
      ["c2", "failed", "boom"],
    ]);
    expect(finalText).toBe("结论");
  });

  it("最后一轮还在调用工具时，没有最终回答；提示代码单独收集", () => {
    const { rounds, finalText, notices } = parseRoundsFromSegments([
      { type: "notice", round: 1, notice: "historyTrimmed" },
      { type: "text", round: 1, content: "查" },
      { type: "tool_call", round: 1, callId: "c1", tool: "t" },
    ]);
    expect(finalText).toBe("");
    expect(rounds[0].commands[0].status).toBe("running");
    expect(notices).toEqual(["historyTrimmed"]);
  });
});

describe("groupConversation", () => {
  const base = { conversationId: 1, createdAt: "2026-11-20T00:00:00Z" };
  const messages: Message[] = [
    { ...base, id: 1, role: "user", content: "集群怎么样" },
    { ...base, id: 2, role: "assistant", content: "先查", toolCalls: [{ id: "a", name: "t", params: "{}" }, { id: "b", name: "t", params: "{}" }] },
    { ...base, id: 3, role: "tool", content: "", toolResult: { callId: "a", name: "t", content: "ok" } },
    { ...base, id: 4, role: "tool", content: "", toolResult: { callId: "b", name: "t", content: "boom", isError: true } },
    { ...base, id: 5, role: "assistant", content: "结论" },
    { ...base, id: 6, role: "user", content: "再问" },
    { ...base, id: 7, role: "assistant", content: "再查", toolCalls: [{ id: "c", name: "t", params: "{}" }] },
  ];

  it("一条带调用的 assistant 消息 = 一轮；tool 消息按 callId 对回；最后一条不带调用的是最终回答", () => {
    const items = groupConversation(messages);
    expect(items.map((i) => i.kind)).toEqual(["user", "answer", "user", "answer"]);
    const first = items[1];
    if (first.kind !== "answer") throw new Error("应为回答");
    expect(first.answer.finalText).toBe("结论");
    expect(first.answer.rounds).toHaveLength(1);
    expect(first.answer.rounds[0].commands.map((c) => c.status)).toEqual(["success", "failed"]);
  });

  it("没有结果的调用（那次回答中途断了）按失败显示，不显示成一直在执行", () => {
    const last = groupConversation(messages)[3];
    if (last.kind !== "answer") throw new Error("应为回答");
    expect(last.answer.rounds[0].commands[0].status).toBe("failed");
    expect(last.answer.finalText).toBe("");
  });
});
