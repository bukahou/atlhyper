"use client";

import { useCallback, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
// 流式对话（SSE）是一条持续输出的连接，不是「取一份数据」，所以不走 useQuery，这里直接调接口。
// 这个文件在守卫测试 no-direct-fetch.test.ts 的流式豁免名单里
import { streamChat } from "@/api/ai";
import type { ChatStats, Message, StreamSegment } from "@/components/ai/types";
import { aiChatKeys } from "@/queries/keys";

/** 一次流式回答的状态。它属于某一个对话：看别的对话时不显示 */
export interface ChatStreamState {
  conversationId: number;
  /** 回答还在输出 */
  streaming: boolean;
  /**
   * 回答期间显示的消息：发送那一刻已有的消息 + 刚发出去的用户消息。
   * 回答结束、消息历史重新取完之后置为 null，改回显示服务端的历史 ——
   * 这样回答期间无论消息历史怎么重取，刚发的那条都不会显示两遍。
   */
  messagesWhileStreaming: Message[] | null;
  segments: StreamSegment[];
  /** 回答结束后的用量统计 */
  stats?: ChatStats;
}

/**
 * 发送消息并接收流式回答。
 *
 * 同一时间只跟一条流：在另一个对话里再发消息时，先中止上一条。
 */
export function useChatStream(clusterId: string) {
  const queryClient = useQueryClient();
  const [stream, setStream] = useState<ChatStreamState | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const sendAndStream = useCallback(
    (conversationId: number, message: string, displayedMessages: Message[]) => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      const userMessage: Message = {
        id: Date.now(),
        conversationId,
        role: "user",
        content: message,
        createdAt: new Date().toISOString(),
      };
      setStream({
        conversationId,
        streaming: true,
        messagesWhileStreaming: [...displayedMessages, userMessage],
        segments: [],
      });

      // 只更新这一条流自己的状态：它已经被新的流取代时什么都不做
      const updateThisStream = (update: (current: ChatStreamState) => ChatStreamState) => {
        setStream((current) => (current !== null && abortRef.current === controller ? update(current) : current));
      };

      streamChat(
        { conversationId, clusterId, message },
        (segment) => updateThisStream((current) => ({ ...current, segments: [...current.segments, segment] })),
        (stats) => {
          updateThisStream((current) => ({ ...current, streaming: false, stats }));
          void queryClient.invalidateQueries({ queryKey: aiChatKeys.conversations() });
          // 等消息历史重新取完，再把临时显示的消息与分段换成服务端的历史
          void queryClient
            .invalidateQueries({ queryKey: aiChatKeys.messages(conversationId) })
            .catch(() => {})
            .then(() => {
              updateThisStream((current) => ({ ...current, messagesWhileStreaming: null, segments: [] }));
              if (abortRef.current === controller) abortRef.current = null;
            });
        },
        (errorMessage) => {
          updateThisStream((current) => ({
            ...current,
            streaming: false,
            segments: [...current.segments, { type: "error", content: errorMessage }],
          }));
        },
        controller.signal,
      );
    },
    [clusterId, queryClient],
  );

  /** 停止生成：中止连接，已经输出的内容留在界面上 */
  const stopStreaming = useCallback(() => {
    abortRef.current?.abort();
    setStream((current) => (current === null ? null : { ...current, streaming: false }));
  }, []);

  /** 丢弃已经结束的流的残留（分段、用量统计）。还在输出的流不动 —— 它属于自己的对话，切回去还能看到 */
  const discardFinishedStream = useCallback(() => {
    setStream((current) => (current !== null && current.streaming ? current : null));
  }, []);

  return { stream, sendAndStream, stopStreaming, discardFinishedStream };
}
