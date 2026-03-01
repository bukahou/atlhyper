"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useSearchParams, useRouter } from "next/navigation";
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
import type { AIConfigStatus } from "./types";
import { MOCK_CONVERSATIONS, MOCK_MESSAGES } from "./mock-data";

export function useAIChat() {
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

  // 发送消息并启动 SSE 流式
  const sendAndStream = useCallback(
    (convId: number, message: string) => {
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

      const controller = new AbortController();
      abortRef.current = controller;

      streamChat(
        { conversation_id: convId, cluster_id: currentClusterId, message },
        (segment) => setStreamSegments((prev) => [...prev, segment]),
        (stats) => {
          setStreaming(false);
          setCurrentStats(stats);
          abortRef.current = null;
          getMessages(convId)
            .then((res) => {
              setMessages(res.data || []);
              setStreamSegments([]);
            })
            .catch(() => {});
          loadConversations();
        },
        (err) => {
          setStreamSegments((prev) => [...prev, { type: "error", content: err }]);
          setStreaming(false);
          abortRef.current = null;
        },
        controller.signal,
      );
    },
    [currentClusterId, loadConversations],
  );

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
    if (isAuthenticated) checkAIConfig();
  }, [isAuthenticated, checkAIConfig]);

  // AI 配置就绪后加载对话列表
  useEffect(() => {
    if (isAuthenticated && aiStatus === "ready") loadConversations();
  }, [isAuthenticated, aiStatus, loadConversations]);

  // 处理从告警页面跳转过来的情况
  useEffect(() => {
    const fromAlerts = searchParams.get("from") === "alerts";
    if (!fromAlerts || alertsProcessed || !isAuthenticated || aiStatus !== "ready") return;

    const stored = sessionStorage.getItem("alertContext");
    if (!stored) return;

    try {
      const alerts: RecentAlert[] = JSON.parse(stored);
      sessionStorage.removeItem("alertContext");
      setAlertsProcessed(true);
      if (alerts.length === 0) return;

      const message = formatAlertsMessage(alerts);
      const alertTitle = t.locale === "zh" ? "告警分析" : "アラート分析";

      (async () => {
        try {
          const res = await createConversation(currentClusterId, alertTitle);
          const newConv = res.data;
          setConversations((prev) => [newConv, ...prev]);
          setCurrentConvId(newConv.id);
          setMessages([]);
          setStreamSegments([]);
          sendAndStream(newConv.id, message);
        } catch {
          // 创建对话失败
        }
      })();
    } catch {
      sessionStorage.removeItem("alertContext");
    }
  }, [searchParams, alertsProcessed, isAuthenticated, aiStatus, currentClusterId, loadConversations, sendAndStream]);

  // 选择对话 -> 加载消息
  const handleSelect = useCallback(async (id: number) => {
    setCurrentConvId(id);
    setStreamSegments([]);
    setStreaming(false);
    setCurrentStats(undefined);
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
    setCurrentStats(undefined);
    sendAndStream(convId, message);
  }, [currentConvId, currentClusterId, sendAndStream]);

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

  // 演示模式下的处理函数
  const handleDemoAction = useCallback(() => {
    openLoginDialog();
  }, [openLoginDialog]);

  return {
    aiStatus,
    isDemo,
    conversations,
    currentConvId,
    messages,
    streaming,
    streamSegments,
    currentStats,
    currentClusterId,
    handleSelect,
    handleNew,
    handleDelete,
    handleSend,
    handleStop,
    goToSettings,
    handleDemoAction,
    openLoginDialog,
  };
}
