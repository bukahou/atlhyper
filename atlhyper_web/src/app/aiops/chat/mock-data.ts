import { Conversation, Message } from "@/components/ai/types";

// Mock 数据（演示用）
export const MOCK_CONVERSATIONS: Conversation[] = [
  {
    id: 1,
    cluster_id: "demo-cluster",
    title: "Pod 故障诊断",
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
    title: "集群资源分析",
    message_count: 2,
    total_input_tokens: 128,
    total_output_tokens: 256,
    total_tool_calls: 1,
    created_at: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
    updated_at: new Date(Date.now() - 1000 * 60 * 60).toISOString(),
  },
];

export const MOCK_MESSAGES: Message[] = [
  {
    id: 1,
    conversation_id: 1,
    role: "user",
    content: "nginx-deployment 的 Pod 一直在重启，帮我分析一下原因",
    created_at: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
  },
  {
    id: 2,
    conversation_id: 1,
    role: "assistant",
    content: `我来帮你分析 nginx-deployment 的 Pod 重启问题。

## 诊断结果

通过查询集群状态，发现以下问题：

### Pod 状态
- **Pod 名称**: nginx-deployment-5d4f7b8c9-abc12
- **状态**: CrashLoopBackOff
- **重启次数**: 5 次
- **最后退出码**: 137 (OOMKilled)

### 根因分析
Pod 因为内存不足 (OOM) 被系统终止。当前配置：
- 内存限制: 128Mi
- 实际使用峰值: ~150Mi

### 修复建议
1. **临时方案**: 增加内存限制到 256Mi
\`\`\`yaml
resources:
  limits:
    memory: "256Mi"
  requests:
    memory: "128Mi"
\`\`\`

2. **长期方案**: 检查应用是否有内存泄漏

需要我帮你执行修复操作吗？`,
    tool_calls: JSON.stringify([
      {
        tool: "query_cluster",
        params: '{"action":"describe","kind":"Pod","namespace":"default","name":"nginx-deployment-5d4f7b8c9-abc12"}',
        result: "Pod 详情: Status=CrashLoopBackOff, RestartCount=5, LastExitCode=137",
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
    content: "好的，请帮我把内存限制调整到 256Mi",
    created_at: new Date(Date.now() - 1000 * 60 * 10).toISOString(),
  },
  {
    id: 4,
    conversation_id: 1,
    role: "assistant",
    content: `已成功更新 nginx-deployment 的内存配置！

## 执行结果
✅ Deployment 已更新，新的 Pod 正在启动中

### 变更详情
- **资源**: Deployment/nginx-deployment
- **命名空间**: default
- **变更内容**: memory limit 128Mi → 256Mi

### 当前状态
- 新 Pod 已创建: nginx-deployment-5d4f7b8c9-xyz99
- 状态: Running
- 重启次数: 0

建议持续观察 10-15 分钟确认问题已解决。`,
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
