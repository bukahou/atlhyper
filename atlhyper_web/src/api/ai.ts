/**
 * AI Chat API
 *
 * CRUD 操作使用 Axios (走统一拦截器)
 * SSE 流式对话使用原生 fetch + ReadableStream (Axios 不支持 SSE)
 */

import { get, post, del, authErrorManager } from "./request";
import { env } from "@/config/env";
import { Conversation, Message, StreamSegment, ChatStats } from "@/components/ai/types";
import { readChatStream, type ChatStreamFailure } from "@/lib/chat-sse";

// ============================================================
// CRUD 接口
// ============================================================

/** 获取对话列表 */
export function getConversations(limit = 20, offset = 0, signal?: AbortSignal) {
  return get<Conversation[]>("/api/v2/ai/conversations", { limit, offset }, signal);
}

/** 创建对话 */
export function createConversation(clusterId: string, title?: string) {
  return post<Conversation>("/api/v2/ai/conversations", {
    clusterId: clusterId,
    title: title || "新对话",
  });
}

/** 删除对话 */
export function deleteConversation(id: number) {
  return del<{ status: string }>(`/api/v2/ai/conversations/${id}`);
}

/** 获取对话消息历史 */
export function getMessages(conversationId: number, signal?: AbortSignal) {
  return get<Message[]>(`/api/v2/ai/conversations/${conversationId}/messages`, undefined, signal);
}

// ============================================================
// SSE 流式对话
// ============================================================

/** 交给 AI 分析的一条告警（只传结构化字段，提示词由后端按 language 拼） */
export interface ChatAlert {
  kind: string;
  namespace: string;
  name: string;
  reason: string;
  message: string;
  timestamp: string;
}

export interface StreamChatParams {
  conversationId: number;
  clusterId: string;
  /** 用户消息；告警分析时不传，改传 alerts */
  message?: string;
  alerts?: ChatAlert[];
  /** 界面语言：后端按它拼告警分析的提示词 */
  language?: string;
}

/**
 * SSE 流式对话
 *
 * 使用原生 fetch + ReadableStream 实现，因为：
 * - Axios 不支持 ReadableStream
 * - EventSource 只支持 GET，无法发送 POST body 和 Auth header
 * 事件分帧与结束判定见 lib/chat-sse。
 *
 * @param params - 请求参数
 * @param onChunk - 收到 text/tool_call/tool_result/notice 时回调
 * @param onDone - 流正常结束回调，接收可选的统计信息
 * @param onError - 没有正常结束时回调（原因的文案由调用方按界面语言出）
 * @param signal - AbortSignal，用于取消请求（取消不算错误）
 */
export function streamChat(
  params: StreamChatParams,
  onChunk: (segment: StreamSegment) => void,
  onDone: (stats?: ChatStats) => void,
  onError: (failure: ChatStreamFailure) => void,
  signal?: AbortSignal,
) {
  const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;

  fetch(`${env.apiUrl}/api/v2/ai/chat`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(params),
    signal,
  })
    .then(async (response) => {
      if (!response.ok) {
        if (response.status === 401) {
          authErrorManager.emit({ type: "unauthorized", message: "登录已过期，请重新登录" });
        }
        onError({ kind: "http", status: response.status });
        return;
      }
      if (!response.body) {
        onError({ kind: "network", message: "ReadableStream unavailable" });
        return;
      }
      const failure = await readChatStream(response.body, { onSegment: onChunk, onDone });
      if (failure) onError(failure);
    })
    .catch((err) => {
      // 用户主动中断，不视为错误
      if (err?.name === "AbortError") return;
      onError({ kind: "network", message: err?.message ?? "" });
    });
}
