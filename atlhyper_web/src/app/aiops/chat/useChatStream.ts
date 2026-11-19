"use client";

import { useCallback, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
// 流式对话（SSE）是一条持续输出的连接，不是「取一份数据」，所以不走 useQuery，这里直接调接口。
// 这个文件在守卫测试 no-direct-fetch.test.ts 的流式豁免名单里
import { streamChat, type ChatAlert } from "@/api/ai";
import type { Message } from "@/components/ai/types";
import { aiChatKeys } from "@/queries/keys";
import { type ChatStreamEvent, type ChatStreamState, reduceChatStream, startChatStream } from "./chat-stream-state";

export type { ChatStreamState } from "./chat-stream-state";

/**
 * 发送消息并接收流式回答。
 *
 * 同一时间只跟一条流：在另一个对话里再发消息时，先中止上一条。
 * 每条流有自己的编号，所有状态更新都带编号交给 reduceChatStream 判断归属。
 */
export function useChatStream(clusterId: string) {
  const queryClient = useQueryClient();
  const [stream, setStream] = useState<ChatStreamState | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const nextStreamIdRef = useRef(1);
  const currentStreamIdRef = useRef(0);

  const dispatch = useCallback((event: ChatStreamEvent) => {
    setStream((current) => reduceChatStream(current, event));
  }, []);

  /** 服务端已经存下的内容（用户消息、做完的工具轮次）以历史为准，重新取一次 */
  const refreshHistory = useCallback(
    (conversationId: number) => {
      void queryClient.invalidateQueries({ queryKey: aiChatKeys.conversations() });
      return queryClient.invalidateQueries({ queryKey: aiChatKeys.messages(conversationId) }).catch(() => {});
    },
    [queryClient],
  );

  const sendAndStream = useCallback(
    /**
     * message 是显示在对话里的用户消息；alertAnalysis 有值时把告警交给后端拼提示词（message 只用于显示）
     */
    (conversationId: number, message: string, displayedMessages: Message[], alertAnalysis?: { alerts: ChatAlert[]; language: string }) => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      const streamId = nextStreamIdRef.current++;
      currentStreamIdRef.current = streamId;

      const userMessage: Message = {
        id: Date.now(),
        conversationId,
        role: "user",
        content: message,
        createdAt: new Date().toISOString(),
      };
      setStream(startChatStream(streamId, conversationId, displayedMessages, userMessage));

      streamChat(
        alertAnalysis
          ? { conversationId, clusterId, alerts: alertAnalysis.alerts, language: alertAnalysis.language }
          : { conversationId, clusterId, message },
        (segment) => dispatch({ type: "segment", streamId, segment }),
        (stats) => {
          dispatch({ type: "done", streamId, stats });
          // 等消息历史重新取完，再把临时显示的消息与分段换成服务端的历史
          void refreshHistory(conversationId).then(() => {
            dispatch({ type: "historyReloaded", streamId });
            if (abortRef.current === controller) abortRef.current = null;
          });
        },
        (errorMessage) => {
          dispatch({ type: "error", streamId, message: errorMessage });
          void refreshHistory(conversationId);
          if (abortRef.current === controller) abortRef.current = null;
        },
        controller.signal,
      );
    },
    [clusterId, dispatch, refreshHistory],
  );

  /** 停止生成：中止连接，已经输出的内容留在界面上 */
  const stopStreaming = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    dispatch({ type: "stopped", streamId: currentStreamIdRef.current });
    if (stream !== null) void refreshHistory(stream.conversationId);
  }, [dispatch, refreshHistory, stream]);

  /** 丢弃已经结束的流的残留（分段、用量统计）。还在输出的流不动 —— 它属于自己的对话，切回去还能看到 */
  const discardFinishedStream = useCallback(() => dispatch({ type: "discardFinished" }), [dispatch]);

  return { stream, sendAndStream, stopStreaming, discardFinishedStream };
}
