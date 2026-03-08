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
    {
      id: 2,
      conversationId: 1,
      role: "assistant",
      content: t.demo.msg2Assistant,
      toolCalls: JSON.stringify([
        {
          tool: "query_cluster",
          params: '{"action":"describe","kind":"Pod","namespace":"default","name":"nginx-deployment-5d4f7b8c9-abc12"}',
          result: "Pod: Status=CrashLoopBackOff, RestartCount=5, LastExitCode=137",
        },
        {
          tool: "query_cluster",
          params: '{"action":"get_events","involved_kind":"Pod","involved_name":"nginx-deployment-5d4f7b8c9-abc12"}',
          result: "Events: OOMKilled - Container exceeded memory limit",
        },
      ]),
      createdAt: new Date(Date.now() - 1000 * 60 * 25).toISOString(),
    },
    {
      id: 3,
      conversationId: 1,
      role: "user",
      content: t.demo.msg3User,
      createdAt: new Date(Date.now() - 1000 * 60 * 10).toISOString(),
    },
    {
      id: 4,
      conversationId: 1,
      role: "assistant",
      content: t.demo.msg4Assistant,
      toolCalls: JSON.stringify([
        {
          tool: "execute_command",
          params: '{"action":"patch","kind":"Deployment","namespace":"default","name":"nginx-deployment","patch":"..."}',
          result: "deployment.apps/nginx-deployment patched",
        },
      ]),
      createdAt: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    },
  ];
}
