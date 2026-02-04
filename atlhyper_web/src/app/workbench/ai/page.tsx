"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { LogIn, Settings, AlertTriangle, Loader2 } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { ChatPanel } from "@/components/ai/ChatPanel";
import { StatusPage } from "@/components/common";
import { Conversation, Message, StreamSegment, ChatStats } from "@/components/ai/types";
import { useClusterStore } from "@/store/clusterStore";
import { useAuthStore } from "@/store/authStore";
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

// AI 配置状态类型
type AIConfigStatus = "loading" | "not_enabled" | "not_configured" | "ready";

export default function AIChatPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { currentClusterId } = useClusterStore();
  const { isAuthenticated, openLoginDialog } = useAuthStore();

  // AI 配置状态
  const [aiStatus, setAiStatus] = useState<AIConfigStatus>("loading");

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

  // 未登录时弹出登录框
  useEffect(() => {
    if (!isAuthenticated) {
      openLoginDialog();
    }
  }, [isAuthenticated, openLoginDialog]);

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
      (async () => {
        try {
          const res = await createConversation(currentClusterId, "告警分析");
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
    router.push("/system/settings/ai");
  }, [router]);

  // ==================== 条件渲染 ====================

  // 未登录
  if (!isAuthenticated) {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={LogIn}
            title="需要登录"
            description="请先登录后使用 AI 助手"
            action={{ label: "登录", onClick: () => openLoginDialog() }}
          />
        </div>
      </Layout>
    );
  }

  // 加载中
  if (aiStatus === "loading") {
    return (
      <Layout>
        <div className="-m-6 h-[calc(100vh-3.5rem)]">
          <StatusPage
            icon={Loader2}
            title="加载中"
            description="正在检查 AI 配置..."
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
            title="AI 功能未启用"
            description="管理员已关闭 AI 功能，请联系管理员或在系统设置中启用"
            action={{ label: "前往设置", onClick: goToSettings }}
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
            title="AI 功能未配置"
            description="请先在系统设置中添加 AI 提供商（如 Gemini、OpenAI、Claude）并选择激活"
            action={{ label: "前往设置", onClick: goToSettings }}
          />
        </div>
      </Layout>
    );
  }

  // AI 配置就绪，显示聊天界面
  return (
    <Layout>
      {/* 负边距抵消 Layout 的 p-6，全屏聊天布局 */}
      <div className="-m-6 h-[calc(100vh-3.5rem)] flex relative">
        <ChatPanel
          messages={messages}
          streaming={streaming}
          streamSegments={streamSegments}
          conversations={conversations}
          currentConvId={currentConvId}
          clusterId={currentClusterId}
          currentStats={currentStats}
          onSelectConv={handleSelect}
          onNewConv={handleNew}
          onDeleteConv={handleDelete}
          onSend={handleSend}
          onStop={handleStop}
        />
      </div>
    </Layout>
  );
}
