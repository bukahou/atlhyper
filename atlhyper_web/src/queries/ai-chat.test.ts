import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const chatApi = vi.hoisted(() => ({
  getConversations: vi.fn(),
  getMessages: vi.fn(),
  createConversation: vi.fn(),
  deleteConversation: vi.fn(),
}));
const providerApi = vi.hoisted(() => ({ getAISettings: vi.fn() }));
vi.mock("@/api/ai", () => chatApi);
vi.mock("@/api/ai-provider", () => providerApi);

import {
  aiChatReadinessQuery,
  conversationListQuery,
  conversationMessagesQuery,
  createConversationMutation,
  deleteConversationMutation,
} from "./ai-chat";
import { aiChatKeys } from "./keys";

beforeEach(() => {
  for (const call of [...Object.values(chatApi), ...Object.values(providerApi)]) call.mockReset();
});

describe("aiChatReadinessQuery", () => {
  it("没有 ai.chat 权限时不发请求", () => {
    expect(aiChatReadinessQuery(false).enabled).toBe(false);
    expect(aiChatReadinessQuery(true).enabled).toBe(true);
  });

  it("返回解包后的设置，带取消信号", async () => {
    providerApi.getAISettings.mockResolvedValue({ data: { toolTimeout: 30, chatReady: true } });

    expect(await new QueryClient().fetchQuery(aiChatReadinessQuery(true))).toEqual({ toolTimeout: 30, chatReady: true });
    expect(providerApi.getAISettings.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });
});

describe("conversationListQuery / conversationMessagesQuery", () => {
  it("AI 不可用或没有权限时不取；没有选中对话时不取消息", () => {
    expect(conversationListQuery(false).enabled).toBe(false);
    expect(conversationListQuery(true).enabled).toBe(true);
    expect(conversationMessagesQuery(3, false).enabled).toBe(false);
    expect(conversationMessagesQuery(null, true).enabled).toBe(false);
    expect(conversationMessagesQuery(3, true).enabled).toBe(true);
  });

  it("消息的键带对话 ID：切换对话时各用各的缓存", () => {
    expect(conversationMessagesQuery(3, true).queryKey).toEqual(["ai-chat", "messages", { conversationId: 3 }]);
    expect(conversationMessagesQuery(3, true).queryKey).not.toEqual(conversationMessagesQuery(4, true).queryKey);
  });

  it("返回解包后的数据，带取消信号", async () => {
    chatApi.getConversations.mockResolvedValue({ data: [{ id: 1 }] });
    chatApi.getMessages.mockResolvedValue({ data: [{ id: 10 }] });
    const client = new QueryClient();

    expect(await client.fetchQuery(conversationListQuery(true))).toEqual([{ id: 1 }]);
    expect(await client.fetchQuery(conversationMessagesQuery(3, true))).toEqual([{ id: 10 }]);

    expect(chatApi.getConversations.mock.calls[0][2]).toBeInstanceOf(AbortSignal);
    expect(chatApi.getMessages.mock.calls[0][0]).toBe(3);
    expect(chatApi.getMessages.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
  });
});

describe("createConversationMutation / deleteConversationMutation", () => {
  it("新建成功：放到列表最前面，消息历史记为空", async () => {
    chatApi.createConversation.mockResolvedValue({ data: { id: 9, title: "t" } });
    const client = new QueryClient();
    client.setQueryData(aiChatKeys.conversations(), [{ id: 1 }]);

    const created = await client
      .getMutationCache()
      .build(client, createConversationMutation())
      .execute({ clusterId: "c1", title: "t" });

    expect(chatApi.createConversation).toHaveBeenCalledWith("c1", "t");
    expect(created).toEqual({ id: 9, title: "t" });
    expect(client.getQueryData(aiChatKeys.conversations())).toEqual([{ id: 9, title: "t" }, { id: 1 }]);
    expect(client.getQueryData(aiChatKeys.messages(9))).toEqual([]);
  });

  it("新建失败不动列表", async () => {
    chatApi.createConversation.mockRejectedValue(new Error("forbidden"));
    const client = new QueryClient();
    client.setQueryData(aiChatKeys.conversations(), [{ id: 1 }]);

    await expect(
      client.getMutationCache().build(client, createConversationMutation()).execute({ clusterId: "c1" }),
    ).rejects.toThrow("forbidden");
    expect(client.getQueryData(aiChatKeys.conversations())).toEqual([{ id: 1 }]);
  });

  it("删除成功：从列表里拿掉，丢弃它的消息缓存", async () => {
    chatApi.deleteConversation.mockResolvedValue({ data: { status: "ok" } });
    const client = new QueryClient();
    client.setQueryData(aiChatKeys.conversations(), [{ id: 1 }, { id: 2 }]);
    client.setQueryData(aiChatKeys.messages(2), [{ id: 10 }]);

    await client.getMutationCache().build(client, deleteConversationMutation()).execute(2);

    expect(chatApi.deleteConversation).toHaveBeenCalledWith(2);
    expect(client.getQueryData(aiChatKeys.conversations())).toEqual([{ id: 1 }]);
    expect(client.getQueryData(aiChatKeys.messages(2))).toBeUndefined();
  });
});
