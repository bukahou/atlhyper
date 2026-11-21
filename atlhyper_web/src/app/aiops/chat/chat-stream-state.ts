import type { ChatStats, Message, StreamSegment } from "@/components/ai/types";

/** 一次流式回答的状态。它属于某一个对话：看别的对话时不显示 */
export interface ChatStreamState {
  /** 这条流的编号。事件只作用于编号相同的流，被新流取代后旧流的事件一律忽略 */
  streamId: number;
  conversationId: number;
  /** 回答还在输出 */
  streaming: boolean;
  /** 用户点了停止生成 */
  stopped: boolean;
  /**
   * 回答期间显示的消息：发送那一刻已有的消息 + 刚发出去的用户消息。
   * 回答正常结束、消息历史重新取完之后置为 null，改回显示服务端的历史 ——
   * 这样回答期间无论消息历史怎么重取，刚发的那条都不会显示两遍。
   */
  messagesWhileStreaming: Message[] | null;
  segments: StreamSegment[];
  /** 回答结束后的用量统计 */
  stats?: ChatStats;
}

export type ChatStreamEvent =
  | { type: "segment"; streamId: number; segment: StreamSegment }
  | { type: "done"; streamId: number; stats: ChatStats | undefined }
  /** 正常结束后消息历史已重新取完：改回显示服务端历史 */
  | { type: "historyReloaded"; streamId: number }
  | { type: "error"; streamId: number; message: string }
  | { type: "stopped"; streamId: number }
  /** 丢弃已经结束的流的残留；还在输出的流不动 —— 它属于自己的对话，切回去还能看到 */
  | { type: "discardFinished" };

export function startChatStream(
  streamId: number,
  conversationId: number,
  displayedMessages: Message[],
  userMessage: Message,
): ChatStreamState {
  return {
    streamId,
    conversationId,
    streaming: true,
    stopped: false,
    messagesWhileStreaming: [...displayedMessages, userMessage],
    segments: [],
  };
}

/**
 * 流式回答的状态迁移。
 *
 * 「这个事件属于不属于当前这条流」只按状态里的 streamId 判断：React 会把函数式更新
 * 推迟到下一次渲染才执行，执行那一刻再去读外部的可变引用（如中止控制器）结果已经变了。
 */
export function reduceChatStream(current: ChatStreamState | null, event: ChatStreamEvent): ChatStreamState | null {
  if (current === null) return null;
  if (event.type === "discardFinished") return current.streaming ? current : null;
  if (event.streamId !== current.streamId) return current;

  switch (event.type) {
    case "segment":
      return { ...current, segments: [...current.segments, event.segment] };
    case "done":
      return { ...current, streaming: false, stats: event.stats };
    case "historyReloaded":
      return { ...current, messagesWhileStreaming: null, segments: [] };
    case "error":
      return {
        ...current,
        streaming: false,
        segments: [...current.segments, { type: "error", content: event.message }],
      };
    case "stopped":
      return { ...current, streaming: false, stopped: true };
  }
}

/**
 * 是否显示流式气泡：还在输出，或者停止 / 出错后还留着已输出的内容。
 * 正常结束并换回服务端历史后分段已清空，回答由历史里的消息显示。
 */
export function showStreamingBubble(stream: ChatStreamState | null): boolean {
  if (stream === null) return false;
  return stream.streaming || stream.segments.length > 0;
}
