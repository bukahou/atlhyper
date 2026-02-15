"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { LogIn, Settings, AlertTriangle, Loader2, Eye } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { ChatPanel } from "@/components/ai/ChatPanel";
import { StatusPage } from "@/components/common";
import { Conversation, Message, StreamSegment, ChatStats } from "@/components/ai/types";
import { useClusterStore } from "@/store/clusterStore";
import { useAuthStore } from "@/store/authStore";
import { useI18n } from "@/i18n/context";
import {
  getConversations,
  createConversation,
  deleteConversation,
  getMessages,
  streamChat,
} from "@/api/ai";
import { getActiveConfig, type ActiveConfig } from "@/api/ai-provider";
import { formatAlertsMessage } from "@/lib/alertFormat";
import type { RecentAlert } from "@/types/overview";

// Mock 数据（演示用）
const MOCK_CONVERSATIONS: Conversation[] = [
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

const MOCK_MESSAGES: Message[] = [
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

// AI 配置状态类型
type AIConfigStatus = "loading" | "not_enabled" | "not_configured" | "ready";

export default function AIChatPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const { isAuthenticated, openLoginDialog } = useAuthStore();

  // 演示模式（未登录）
  const isDemo = !isAuthenticated;

  // AI 配置状态
  const [aiStatus, setAiStatus] = useState<AIConfigStatus>(isDemo ? "ready" : "loading");

  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [currentConvId, setCurrentConvId] = useState<number | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [streaming, setStreaming] = useState(false);
  const [streamSegments, setStreamSegments] = useState<StreamSegment[]>([]);
  const [currentStats, setCurrentStats] = useState<ChatStats | undefined>();
  const [alertsProcessed, setAlertsProcessed] = useState(false);

  const abortRef = useRef<AbortController | null>(null);

  // 检查 AI 配置状态
  const checkAIConfig = useCallback(async () => {
    try {
      const res = await getActiveConfig();
      const config: ActiveConfig = res.data;

      if (!config.enabled) {
        setAiStatus("not_enabled");
      } else if (config.provider_id === null) {
        setAiStatus("not_configured");
      } else {
        setAiStatus("ready");
      }
    } catch {
      // API 调用失败，可能是未登录或网络问题
      setAiStatus("not_configured");
    }
  }, []);

  // 加载对话列表
  const loadConversations = useCallback(async () => {
    try {
      const res = await getConversations();
      setConversations(res.data || []);
    } catch {
      // 忽略（401 由全局拦截器处理）
    }
  }, []);

  // 演示模式：加载 mock 数据
  useEffect(() => {
    if (isDemo) {
      setConversations(MOCK_CONVERSATIONS);
      setCurrentConvId(1);
      setMessages(MOCK_MESSAGES);
      setAiStatus("ready");
    }
  }, [isDemo]);

  // 登录后检查 AI 配置状态
  useEffect(() => {
    if (isAuthenticated) {
      checkAIConfig();
    }
  }, [isAuthenticated, checkAIConfig]);

  // AI 配置就绪后加载对话列表
  useEffect(() => {
    if (isAuthenticated && aiStatus === "ready") {
      loadConversations();
    }
  }, [isAuthenticated, aiStatus, loadConversations]);

  // 处理从告警页面跳转过来的情况
  useEffect(() => {
    const fromAlerts = searchParams.get("from") === "alerts";
    if (!fromAlerts || alertsProcessed || !isAuthenticated || aiStatus !== "ready") return;

    const stored = sessionStorage.getItem("alertContext");
    if (!stored) return;

    try {
      const alerts: RecentAlert[] = JSON.parse(stored);
      sessionStorage.removeItem("alertContext"); // 用完即删
      setAlertsProcessed(true);

      if (alerts.length === 0) return;

      // 格式化告警为消息并自动发送
      const message = formatAlertsMessage(alerts);

      // 创建新对话并发送消息
      const alertTitle = t.locale === "zh" ? "告警分析" : "アラート分析";
      (async () => {
        try {
          const res = await createConversation(currentClusterId, alertTitle);
          const newConv = res.data;
          setConversations((prev) => [newConv, ...prev]);
          setCurrentConvId(newConv.id);
          setMessages([]);
          setStreamSegments([]);

          // 追加 user 消息到 UI
          const userMsg: Message = {
            id: Date.now(),
            conversation_id: newConv.id,
            role: "user",
            content: message,
            created_at: new Date().toISOString(),
          };
          setMessages([userMsg]);
          setStreaming(true);
          setStreamSegments([]);

          // 创建 AbortController 并发送
          const controller = new AbortController();
          abortRef.current = controller;

          streamChat(
            {
              conversation_id: newConv.id,
              cluster_id: currentClusterId,
              message,
            },
            (segment) => {
              setStreamSegments((prev) => [...prev, segment]);
            },
            (stats) => {
              setStreaming(false);
              setCurrentStats(stats);
              abortRef.current = null;
              getMessages(newConv.id)
                .then((res) => {
                  setMessages(res.data || []);
                  setStreamSegments([]); // 清空 streamSegments
                })
                .catch(() => {});
              loadConversations();
            },
            (err) => {
              // 先添加 error segment，再停止 streaming
              setStreamSegments((prev) => [
                ...prev,
                { type: "error", content: err },
              ]);
              setStreaming(false);
              abortRef.current = null;
            },
            controller.signal,
          );
        } catch {
          // 创建对话失败
        }
      })();
    } catch {
      // JSON 解析失败
      sessionStorage.removeItem("alertContext");
    }
  }, [searchParams, alertsProcessed, isAuthenticated, aiStatus, currentClusterId, loadConversations]);

  // 选择对话 → 加载消息
  const handleSelect = useCallback(async (id: number) => {
    setCurrentConvId(id);
    setStreamSegments([]);
    setStreaming(false);
    setCurrentStats(undefined); // 切换对话时重置统计
    try {
      const res = await getMessages(id);
      setMessages(res.data || []);
    } catch {
      setMessages([]);
    }
  }, []);

  // 新建对话
  const handleNew = useCallback(async () => {
    try {
      const res = await createConversation(currentClusterId);
      const newConv = res.data;
      setConversations((prev) => [newConv, ...prev]);
      setCurrentConvId(newConv.id);
      setMessages([]);
      setStreamSegments([]);
      setCurrentStats(undefined);
    } catch {
      // 创建失败忽略
    }
  }, [currentClusterId]);

  // 删除对话
  const handleDelete = useCallback(async (id: number) => {
    try {
      await deleteConversation(id);
      setConversations((prev) => prev.filter((c) => c.id !== id));
      if (currentConvId === id) {
        setCurrentConvId(null);
        setMessages([]);
      }
    } catch {
      // 删除失败忽略
    }
  }, [currentConvId]);

  // 发送消息 (SSE 流式)
  const handleSend = useCallback(async (message: string) => {
    // 如果没有当前对话，自动创建一个
    let convId = currentConvId;
    if (!convId) {
      try {
        const res = await createConversation(currentClusterId, message.slice(0, 20));
        const newConv = res.data;
        convId = newConv.id;
        setConversations((prev) => [newConv, ...prev]);
        setCurrentConvId(convId);
      } catch {
        return;
      }
    }

    // 追加 user 消息到 UI（乐观更新）
    const userMsg: Message = {
      id: Date.now(),
      conversation_id: convId,
      role: "user",
      content: message,
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, userMsg]);
    setStreaming(true);
    setStreamSegments([]);
    setCurrentStats(undefined); // 开始新提问时重置统计

    // 创建 AbortController
    const controller = new AbortController();
    abortRef.current = controller;

    streamChat(
      {
        conversation_id: convId,
        cluster_id: currentClusterId,
        message,
      },
      // onChunk
      (segment) => {
        setStreamSegments((prev) => [...prev, segment]);
      },
      // onDone
      (stats) => {
        setStreaming(false);
        setCurrentStats(stats);
        abortRef.current = null;
        // 重新加载消息和对话列表（后端已持久化统计信息）
        getMessages(convId!)
          .then((res) => {
            setMessages(res.data || []);
            setStreamSegments([]); // 清空 streamSegments
          })
          .catch(() => {});
        // 刷新对话列表（更新 message_count）
        loadConversations();
      },
      // onError
      (err) => {
        // 先添加 error segment，再停止 streaming
        setStreamSegments((prev) => [
          ...prev,
          { type: "error", content: err },
        ]);
        setStreaming(false);
        abortRef.current = null;
      },
      controller.signal,
    );
  }, [currentConvId, currentClusterId, loadConversations]);

  // 停止生成
  const handleStop = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setStreaming(false);
  }, []);

  // 跳转到设置页
  const goToSettings = useCallback(() => {
    router.push("/settings/ai");
  }, [router]);

  // 演示模式下的处理函数（禁用操作，显示登录提示）
  const handleDemoAction = useCallback(() => {
    openLoginDialog();
  }, [openLoginDialog]);

  // ==================== 条件渲染 ====================
  const aiChatT = t.aiChatPage;

  // 加载中（仅登录后）
  if (aiStatus === "loading") {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={Loader2}
            title={aiChatT.loading}
            description={aiChatT.checkingConfig}
          />
        </div>
      </Layout>
    );
  }

  // AI 功能未启用
  if (aiStatus === "not_enabled") {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={AlertTriangle}
            title={aiChatT.notEnabled}
            description={aiChatT.notEnabledDesc}
            action={{ label: aiChatT.goToSettings, onClick: goToSettings }}
          />
        </div>
      </Layout>
    );
  }

  // 未配置 AI 提供商
  if (aiStatus === "not_configured") {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={Settings}
            title={aiChatT.notConfigured}
            description={aiChatT.notConfiguredDesc}
            action={{ label: aiChatT.goToSettings, onClick: goToSettings }}
          />
        </div>
      </Layout>
    );
  }

  // AI 配置就绪，显示聊天界面
  return (
    <Layout>
      {/* 负边距抵消 Layout 的 p-6，全屏聊天布局 */}
      <div className="-m-6 h-[calc(100vh-3.5rem)] flex flex-col relative">
        {/* 演示模式横幅 */}
        {isDemo && (
          <div className="flex items-center justify-between gap-3 px-4 py-2 bg-amber-50 dark:bg-amber-900/20 border-b border-amber-200 dark:border-amber-800">
            <div className="flex items-center gap-2">
              <Eye className="w-4 h-4 text-amber-600 dark:text-amber-400" />
              <span className="text-sm text-amber-800 dark:text-amber-300">
                {aiChatT.demoMode} - {aiChatT.demoModeDesc}
              </span>
            </div>
            <button
              onClick={() => openLoginDialog()}
              className="px-3 py-1 text-xs font-medium rounded-lg bg-amber-600 text-white hover:bg-amber-700 transition-colors"
            >
              {aiChatT.login}
            </button>
          </div>
        )}
        <div className="flex-1 flex">
          <ChatPanel
            messages={messages}
            streaming={streaming}
            streamSegments={streamSegments}
            conversations={conversations}
            currentConvId={currentConvId}
            clusterId={currentClusterId}
            currentStats={currentStats}
            onSelectConv={isDemo ? () => {} : handleSelect}
            onNewConv={isDemo ? handleDemoAction : handleNew}
            onDeleteConv={isDemo ? handleDemoAction : handleDelete}
            onSend={isDemo ? handleDemoAction : handleSend}
            onStop={handleStop}
            readOnly={isDemo}
          />
        </div>
      </div>
    </Layout>
  );
}
