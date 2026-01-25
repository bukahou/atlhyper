"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { LogIn } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { ChatPanel } from "@/components/ai/ChatPanel";
import { StatusPage } from "@/components/common";
import { Conversation, Message, StreamSegment } from "@/components/ai/types";
import { useClusterStore } from "@/store/clusterStore";
import { useAuthStore } from "@/store/authStore";
import {
  getConversations,
  createConversation,
  deleteConversation,
  getMessages,
  streamChat,
} from "@/api/ai";

export default function AIChatPage() {
  const { currentClusterId } = useClusterStore();
  const { isAuthenticated, openLoginDialog } = useAuthStore();

  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [currentConvId, setCurrentConvId] = useState<number | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [streaming, setStreaming] = useState(false);
  const [streamSegments, setStreamSegments] = useState<StreamSegment[]>([]);

  const abortRef = useRef<AbortController | null>(null);

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

  // 初始化加载（仅登录后）
  useEffect(() => {
    if (isAuthenticated) {
      loadConversations();
    }
  }, [isAuthenticated, loadConversations]);

  // 选择对话 → 加载消息
  const handleSelect = useCallback(async (id: number) => {
    setCurrentConvId(id);
    setStreamSegments([]);
    setStreaming(false);
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
      () => {
        setStreaming(false);
        abortRef.current = null;
        // 重新加载消息（后端已持久化，获取正确 id）
        getMessages(convId!)
          .then((res) => {
            setMessages(res.data || []);
            setStreamSegments([]);
          })
          .catch(() => {});
        // 刷新对话列表（更新 message_count）
        loadConversations();
      },
      // onError
      (err) => {
        setStreaming(false);
        abortRef.current = null;
        setStreamSegments((prev) => [
          ...prev,
          { type: "error", content: err },
        ]);
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
