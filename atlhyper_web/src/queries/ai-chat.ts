/**
 * AI 对话页的取数描述（对话列表、消息历史、AI 是否可用）
 *
 * 流式对话本身（SSE）不走查询：它是一条持续输出的连接，不是「取一份数据」，
 * 保留在 app/aiops/chat/useChatStream.ts 里直接调接口。
 *
 * 这些接口需要 ai.chat 权限：有没有权限由调用方用 useCan 得到后传进来。
 */
import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { createConversation, deleteConversation, getConversations, getMessages } from "@/api/ai";
import { getAISettings } from "@/api/ai-provider";
import type { Conversation } from "@/components/ai/types";
import { aiChatKeys, aiSettingsKeys } from "./keys";

/** AI 对话是否可用（有没有给 chat 角色分配 Provider）。取不到 = AI 没有配置 */
export function aiChatReadinessQuery(canChat: boolean) {
  return queryOptions({
    queryKey: aiSettingsKeys.chatReadiness(),
    queryFn: async ({ signal }) => (await getAISettings(signal)).data,
    enabled: canChat,
  });
}

/** 对话列表。chatReady 为假（没有权限、或 AI 还不可用）时不取 */
export function conversationListQuery(chatReady: boolean) {
  return queryOptions({
    queryKey: aiChatKeys.conversations(),
    queryFn: async ({ signal }) => (await getConversations(undefined, undefined, signal)).data ?? [],
    enabled: chatReady,
  });
}

/** 某个对话的消息历史。conversationId 为 null 表示没有选中对话 */
export function conversationMessagesQuery(conversationId: number | null, chatReady: boolean) {
  return queryOptions({
    queryKey: aiChatKeys.messages(conversationId ?? 0),
    queryFn: async ({ signal }) => (conversationId === null ? [] : ((await getMessages(conversationId, signal)).data ?? [])),
    enabled: chatReady && conversationId !== null,
  });
}

export interface NewConversation {
  clusterId: string;
  /** 不给标题时由后端 / 接口层取默认标题 */
  title?: string;
}

/**
 * 新建对话。成功后把它放到列表最前面，并把它的消息历史记为空 ——
 * 刚建的对话没有消息，不必为此再请求一次。
 */
export function createConversationMutation() {
  return mutationOptions({
    mutationFn: async ({ clusterId, title }: NewConversation) => (await createConversation(clusterId, title)).data,
    onSuccess: (created, _newConversation, _onMutateResult, context) => {
      context.client.setQueryData<Conversation[]>(aiChatKeys.conversations(), (list) => [created, ...(list ?? [])]);
      context.client.setQueryData(aiChatKeys.messages(created.id), []);
    },
  });
}

/** 删除对话。成功后从列表里拿掉，并丢弃它的消息缓存 */
export function deleteConversationMutation() {
  return mutationOptions({
    mutationFn: (conversationId: number) => deleteConversation(conversationId),
    onSuccess: (_deleted, conversationId, _onMutateResult, context) => {
      context.client.setQueryData<Conversation[]>(aiChatKeys.conversations(), (list) =>
        (list ?? []).filter((conversation) => conversation.id !== conversationId),
      );
      context.client.removeQueries({ queryKey: aiChatKeys.messages(conversationId) });
    },
  });
}
