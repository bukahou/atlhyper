import { describe, expect, it } from "vitest";
import type { Message } from "@/components/ai/types";
import {
  type ChatStreamEvent,
  type ChatStreamState,
  reduceChatStream,
  showStreamingBubble,
  startChatStream,
} from "./chat-stream-state";

const history: Message[] = [
  { id: 1, conversationId: 7, role: "user", content: "之前的问题", createdAt: "2026-11-19T00:00:00Z" },
];
const userMessage: Message = {
  id: 2,
  conversationId: 7,
  role: "user",
  content: "节点状态怎么样",
  createdAt: "2026-11-19T00:01:00Z",
};

function apply(state: ChatStreamState | null, events: ChatStreamEvent[]) {
  return events.reduce(reduceChatStream, state);
}

describe("reduceChatStream", () => {
  it("回答结束、历史重取完后，改回显示服务端历史（临时消息与分段清空）", () => {
    const started = startChatStream(1, 7, history, userMessage);
    const finished = apply(started, [
      { type: "segment", streamId: 1, segment: { type: "text", content: "一切正常" } },
      { type: "done", streamId: 1, stats: { rounds: 0, totalToolCalls: 0, inputTokens: 10, outputTokens: 5 } },
      { type: "historyReloaded", streamId: 1 },
    ]);
    expect(finished).not.toBeNull();
    expect(finished!.streaming).toBe(false);
    expect(finished!.messagesWhileStreaming).toBeNull();
    expect(finished!.segments).toEqual([]);
    expect(finished!.stats?.outputTokens).toBe(5);
  });

  it("旧流的事件不影响新流", () => {
    const second = startChatStream(2, 7, history, userMessage);
    const state = apply(second, [
      { type: "segment", streamId: 1, segment: { type: "text", content: "旧回答" } },
      { type: "done", streamId: 1, stats: undefined },
      { type: "historyReloaded", streamId: 1 },
      { type: "error", streamId: 1, message: "旧错误" },
    ]);
    expect(state).toEqual(second);
  });

  it("停止生成：保留已输出的内容并标记已停止", () => {
    const state = apply(startChatStream(1, 7, history, userMessage), [
      { type: "segment", streamId: 1, segment: { type: "text", content: "正在查" } },
      { type: "stopped", streamId: 1 },
    ]);
    expect(state!.streaming).toBe(false);
    expect(state!.stopped).toBe(true);
    expect(state!.segments).toEqual([{ type: "text", content: "正在查" }]);
    expect(showStreamingBubble(state)).toBe(true);
  });

  it("出错：保留已输出的内容并追加错误分段", () => {
    const state = apply(startChatStream(1, 7, history, userMessage), [
      { type: "segment", streamId: 1, segment: { type: "text", content: "一半" } },
      { type: "error", streamId: 1, message: "AI 服务错误" },
    ]);
    expect(state!.streaming).toBe(false);
    expect(state!.segments.at(-1)).toEqual({ type: "error", content: "AI 服务错误" });
    expect(showStreamingBubble(state)).toBe(true);
  });

  it("丢弃残留：结束了的流清掉，还在输出的流保留", () => {
    const streaming = startChatStream(1, 7, history, userMessage);
    expect(reduceChatStream(streaming, { type: "discardFinished" })).toBe(streaming);
    const stopped = reduceChatStream(streaming, { type: "stopped", streamId: 1 });
    expect(reduceChatStream(stopped, { type: "discardFinished" })).toBeNull();
  });
});

describe("showStreamingBubble", () => {
  it("正常结束、换回服务端历史后不再显示流式气泡", () => {
    const finished = apply(startChatStream(1, 7, history, userMessage), [
      { type: "segment", streamId: 1, segment: { type: "text", content: "完成" } },
      { type: "done", streamId: 1, stats: undefined },
      { type: "historyReloaded", streamId: 1 },
    ]);
    expect(showStreamingBubble(finished)).toBe(false);
  });

  it("还在输出时显示（哪怕还没有任何分段，显示加载动画）", () => {
    expect(showStreamingBubble(startChatStream(1, 7, history, userMessage))).toBe(true);
  });

  it("没有流时不显示", () => {
    expect(showStreamingBubble(null)).toBe(false);
  });
});
