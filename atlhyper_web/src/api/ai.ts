/**
 * AI Chat API
 *
 * CRUD 操作使用 Axios (走统一拦截器)
 * SSE 流式对话使用原生 fetch + ReadableStream (Axios 不支持 SSE)
 */

import { get, post, del } from "./request";
import { env } from "@/config/env";
import { Conversation, Message, StreamSegment } from "@/components/ai/types";

// ============================================================
// CRUD 接口
// ============================================================

/** 获取对话列表 */
export function getConversations(limit = 20, offset = 0) {
  return get<Conversation[]>("/api/v2/ai/conversations", { limit, offset });
}

/** 创建对话 */
export function createConversation(clusterId: string, title?: string) {
  return post<Conversation>("/api/v2/ai/conversations", {
    cluster_id: clusterId,
    title: title || "新对话",
  });
}

/** 删除对话 */
export function deleteConversation(id: number) {
  return del<{ status: string }>(`/api/v2/ai/conversations/${id}`);
}

/** 获取对话消息历史 */
export function getMessages(conversationId: number) {
  return get<Message[]>(`/api/v2/ai/conversations/${conversationId}/messages`);
}

// ============================================================
// SSE 流式对话
// ============================================================

export interface StreamChatParams {
  conversation_id: number;
  cluster_id: string;
  message: string;
}

/**
 * SSE 流式对话
 *
 * 使用原生 fetch + ReadableStream 实现，因为：
 * - Axios 不支持 ReadableStream
 * - EventSource 只支持 GET，无法发送 POST body 和 Auth header
 *
 * @param params - 请求参数
 * @param onChunk - 收到 text/tool_call/tool_result 时回调
 * @param onDone - 流结束回调
 * @param onError - 错误回调
 * @param signal - AbortSignal，用于取消请求
 */
export function streamChat(
  params: StreamChatParams,
  onChunk: (segment: StreamSegment) => void,
  onDone: () => void,
  onError: (err: string) => void,
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
        const text = await response.text().catch(() => "");
        onError(`HTTP ${response.status}: ${text}`);
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        onError("浏览器不支持 ReadableStream");
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";
      let doneReceived = false;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        // 按换行分割 SSE 事件
        const lines = buffer.split("\n");
        buffer = lines.pop() || ""; // 最后不完整行留在 buffer

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const jsonStr = line.slice(6).trim();
            if (!jsonStr) continue;
            try {
              const chunk = JSON.parse(jsonStr);
              if (chunk.type === "done") {
                doneReceived = true;
                onDone();
              } else if (chunk.type === "error") {
                onError(chunk.content || "未知错误");
              } else {
                onChunk(chunk as StreamSegment);
              }
            } catch {
              // 忽略 JSON 解析错误（可能是不完整片段）
            }
          }
        }
      }

      // 流结束但未收到 done 事件（连接意外关闭）
      if (!doneReceived) {
        onDone();
      }
    })
    .catch((err) => {
      if (err.name === "AbortError") {
        // 用户主动中断，不视为错误
        return;
      }
      onError(err.message || "网络错误");
    });
}
