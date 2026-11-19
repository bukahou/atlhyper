import type { Conversation, Message } from "@/components/ai/types";
import type { AIChatPageTranslations } from "@/types/i18n";

// Mock 数据工厂（根据当前语言生成演示数据）

export function getMockConversations(t: AIChatPageTranslations): Conversation[] {
  return [
    {
      id: 1,
      clusterId: "demo-cluster",
      title: t.demo.conv1Title,
      messageCount: 4,
      totalInputTokens: 256,
      totalOutputTokens: 512,
      totalToolCalls: 3,
      createdAt: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
      updatedAt: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    },
    {
      id: 2,
      clusterId: "demo-cluster",
      title: t.demo.conv2Title,
      messageCount: 2,
      totalInputTokens: 128,
      totalOutputTokens: 256,
      totalToolCalls: 1,
      createdAt: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
      updatedAt: new Date(Date.now() - 1000 * 60 * 60).toISOString(),
    },
  ];
}

export function getMockMessages(t: AIChatPageTranslations): Message[] {
  return [
    {
      id: 1,
      conversationId: 1,
      role: "user",
      content: t.demo.msg1User,
      createdAt: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
    },
    // 一轮工具调用（两次并行查询）+ 各自的结果 + 最终回答；形态与后端历史接口一致
    {
      id: 2,
      conversationId: 1,
      role: "assistant",
      content: "",
      toolCalls: [
        {
          id: "demo-1",
          name: "query_cluster",
          params: '{"action":"describe","kind":"Pod","namespace":"default","name":"nginx-deployment-5d4f7b8c9-abc12"}',
        },
        {
          id: "demo-2",
          name: "query_cluster",
          params: '{"action":"get_events","kind":"Event","involved_kind":"Pod","involved_name":"nginx-deployment-5d4f7b8c9-abc12"}',
        },
      ],
      createdAt: new Date(Date.now() - 1000 * 60 * 26).toISOString(),
    },
    {
      id: 3,
      conversationId: 1,
      role: "tool",
      content: "",
      toolResult: { callId: "demo-1", name: "query_cluster", content: "Pod: Status=CrashLoopBackOff, RestartCount=5, LastExitCode=137" },
      createdAt: new Date(Date.now() - 1000 * 60 * 26).toISOString(),
    },
    {
      id: 4,
      conversationId: 1,
      role: "tool",
      content: "",
      toolResult: { callId: "demo-2", name: "query_cluster", content: "Events: OOMKilled - Container exceeded memory limit" },
      createdAt: new Date(Date.now() - 1000 * 60 * 26).toISOString(),
    },
    {
      id: 5,
      conversationId: 1,
      role: "assistant",
      content: t.demo.msg2Assistant,
      createdAt: new Date(Date.now() - 1000 * 60 * 25).toISOString(),
    },
    {
      id: 6,
      conversationId: 1,
      role: "user",
      content: t.demo.msg3User,
      createdAt: new Date(Date.now() - 1000 * 60 * 10).toISOString(),
    },
    {
      id: 7,
      conversationId: 1,
      role: "assistant",
      content: t.demo.msg4Assistant,
      createdAt: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    },
  ];
}
