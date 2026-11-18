"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import type { Message } from "@/components/ai/types";
import { useAuthStore } from "@/store/authStore";
import { useI18n } from "@/i18n/context";
import { formatAlertsMessage } from "@/lib/alertFormat";
import type { RecentAlert } from "@/types/overview";
import type { AIConfigStatus } from "./types";
import { getMockConversations, getMockMessages } from "./mock-data";
import { useCan } from "@/hooks/useCan";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useSessionReady } from "@/hooks/useSessionReady";
import { toast } from "@/components/common/Toast";
import {
  aiChatReadinessQuery,
  conversationListQuery,
  conversationMessagesQuery,
  createConversationMutation,
  deleteConversationMutation,
} from "@/queries/ai-chat";
import { useChatStream } from "./useChatStream";

/** 演示对话里默认选中的那一个（mock-data.ts 的第一条） */
const DEMO_CONVERSATION_ID = 1;
const NO_MESSAGES: Message[] = [];

export function useAIChat() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useI18n();
  const { clusterId: currentClusterId } = useCurrentCluster();
  const { openLoginDialog, isAuthenticated } = useAuthStore();
  const sessionReady = useSessionReady();
  const canChat = useCan()("ai.chat");

  // 演示模式（无 ai.chat 权限：访客与成员）。权限加载完才下结论，之前显示加载中
  const isDemo = sessionReady && !canChat;

  // AI 是否可用：没有配置 / 没有给 chat 角色分配 Provider 时各有一个说明页
  const readiness = useQuery(aiChatReadinessQuery(canChat));
  const chatReady = canChat && readiness.data?.chatReady === true;
  const aiStatus: AIConfigStatus = !sessionReady
    ? "loading"
    : isDemo
      ? "ready"
      : readiness.isPending
        ? "loading"
        : readiness.isError
          ? "not_configured"
          : readiness.data.chatReady
            ? "ready"
            : "chat_not_assigned";

  const [selectedConvId, setSelectedConvId] = useState<number | null>(null);
  const currentConvId = isDemo ? DEMO_CONVERSATION_ID : selectedConvId;

  const conversationList = useQuery(conversationListQuery(chatReady));
  const messageHistory = useQuery(conversationMessagesQuery(selectedConvId, chatReady));
  const createConversation = useMutation(createConversationMutation());
  const deleteConversation = useMutation(deleteConversationMutation());
  const { stream, sendAndStream, stopStreaming, discardFinishedStream } = useChatStream(currentClusterId);

  // 演示数据随语言切换重新生成
  const aiChatT = t.aiChatPage;
  const demoConversations = useMemo(() => getMockConversations(aiChatT), [aiChatT]);
  const demoMessages = useMemo(() => getMockMessages(aiChatT), [aiChatT]);

  // 流式回答属于它自己的对话：看别的对话时不显示它的分段、临时消息与用量
  const streamOfCurrent = stream !== null && stream.conversationId === currentConvId ? stream : null;
  const conversations = isDemo ? demoConversations : (conversationList.data ?? []);
  const messages = isDemo
    ? demoMessages
    : (streamOfCurrent?.messagesWhileStreaming ?? messageHistory.data ?? NO_MESSAGES);

  // 处理从告警页面跳转过来的情况：新建一个对话，把告警内容作为第一条消息发出去。只做一次
  const alertsHandledRef = useRef(false);
  const { mutate: startConversation } = createConversation;
  const alertAnalysisTitle = aiChatT.alertAnalysis;
  const fromAlerts = searchParams.get("from") === "alerts";
  useEffect(() => {
    if (!fromAlerts || !chatReady || alertsHandledRef.current) return;
    const stored = sessionStorage.getItem("alertContext");
    if (!stored) return;
    alertsHandledRef.current = true;
    sessionStorage.removeItem("alertContext");

    let alerts: RecentAlert[] = [];
    try {
      alerts = JSON.parse(stored);
    } catch {
      // 存的内容解析不了：已经清掉，当作没有
    }
    if (alerts.length === 0) return;

    startConversation(
      { clusterId: currentClusterId, title: alertAnalysisTitle },
      {
        onSuccess: (created) => {
          setSelectedConvId(created.id);
          sendAndStream(created.id, formatAlertsMessage(alerts), NO_MESSAGES);
        },
      },
    );
  }, [fromAlerts, chatReady, currentClusterId, alertAnalysisTitle, startConversation, sendAndStream]);

  // 选择对话：消息历史由查询按对话 ID 取
  const handleSelect = (id: number) => {
    setSelectedConvId(id);
    discardFinishedStream();
  };

  // 新建对话
  const handleNew = () => {
    createConversation.mutate(
      { clusterId: currentClusterId },
      {
        onSuccess: (created) => {
          setSelectedConvId(created.id);
          discardFinishedStream();
        },
      },
    );
  };

  // 删除对话
  const handleDelete = (id: number) => {
    deleteConversation.mutate(id, {
      onSuccess: () => setSelectedConvId((current) => (current === id ? null : current)),
    });
  };

  // 发送消息 (SSE 流式)：还没有对话时先建一个，标题取消息开头
  const handleSend = async (message: string) => {
    if (selectedConvId !== null) {
      sendAndStream(selectedConvId, message, messages);
      return;
    }
    try {
      const created = await createConversation.mutateAsync({
        clusterId: currentClusterId,
        title: message.slice(0, 20),
      });
      setSelectedConvId(created.id);
      sendAndStream(created.id, message, NO_MESSAGES);
    } catch {
      // 创建对话失败：不发送
    }
  };

  // 跳转到设置页
  const goToSettings = useCallback(() => {
    router.push("/settings/ai");
  }, [router]);

  // 演示模式下的处理函数：访客弹登录框；已登录但无 ai.chat → 提示联系管理员
  const handleDemoAction = useCallback(() => {
    if (isAuthenticated) {
      toast.error(t.permission.forbidden);
      return;
    }
    openLoginDialog();
  }, [isAuthenticated, openLoginDialog, t]);

  return {
    aiStatus,
    isDemo,
    conversations,
    currentConvId,
    messages,
    streaming: streamOfCurrent?.streaming ?? false,
    streamSegments: streamOfCurrent?.segments ?? [],
    currentStats: streamOfCurrent?.stats,
    currentClusterId,
    handleSelect,
    handleNew,
    handleDelete,
    handleSend,
    handleStop: stopStreaming,
    goToSettings,
    handleDemoAction,
    openLoginDialog,
    isAuthenticated,
  };
}
