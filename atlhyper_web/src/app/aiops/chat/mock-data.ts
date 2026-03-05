import type { Conversation, Message } from "@/components/ai/types";
import type { AIChatPageTranslations } from "@/types/i18n";

// Mock 数据工厂（根据当前语言生成演示数据）

export function getMockConversations(t: AIChatPageTranslations): Conversation[] {
  return [
    {
      id: 1,
      cluster_id: "demo-cluster",
      title: t.demo.conv1Title,
      message_count: 4,
      total_input_tokens: 256,
      total_output_tokens: 512,
      total_tool_calls: 3,
      created_at: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
      updated_at: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    },
    {
      id: 2,
      cluster_id: "demo-cluster",
      title: t.demo.conv2Title,
      message_count: 2,
      total_input_tokens: 128,
      total_output_tokens: 256,
      total_tool_calls: 1,
      created_at: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
      updated_at: new Date(Date.now() - 1000 * 60 * 60).toISOString(),
    },
  ];
}

export function getMockMessages(t: AIChatPageTranslations): Message[] {
  return [
    {
      id: 1,
      conversation_id: 1,
      role: "user",
      content: t.demo.msg1User,
      created_at: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
    },
    {
      id: 2,
      conversation_id: 1,
      role: "assistant",
      content: t.demo.msg2Assistant,
      tool_calls: JSON.stringify([
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
      created_at: new Date(Date.now() - 1000 * 60 * 25).toISOString(),
    },
    {
      id: 3,
      conversation_id: 1,
      role: "user",
      content: t.demo.msg3User,
      created_at: new Date(Date.now() - 1000 * 60 * 10).toISOString(),
    },
    {
      id: 4,
      conversation_id: 1,
      role: "assistant",
      content: t.demo.msg4Assistant,
      tool_calls: JSON.stringify([
        {
          tool: "execute_command",
          params: '{"action":"patch","kind":"Deployment","namespace":"default","name":"nginx-deployment","patch":"..."}',
          result: "deployment.apps/nginx-deployment patched",
        },
      ]),
      created_at: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    },
  ];
}
