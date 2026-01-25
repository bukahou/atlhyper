"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { Bot, History, Settings2 } from "lucide-react";
import { ConversationPanel } from "./ConversationSidebar";
import { InspectorPanel } from "./InspectorPanel";
import { Conversation } from "./types";
import { Message, StreamSegment } from "./types";
import { MessageBubble, StreamingBubble } from "./MessageBubble";
import { ChatInput } from "./ChatInput";

// Token 估算: 每次 API 调用发送完整上下文
const OVERHEAD_TOKENS = 550; // system prompt + tool definitions

function estimateTokens(text: string): number {
  if (!text) return 0;
  return Math.ceil(text.length * 0.7);
}

function getCallTokens(messages: Message[], index: number): number | undefined {
  const msg = messages[index];
  if (msg.role !== "assistant" || !msg.tool_calls) return undefined;

  // input = overhead + 此消息之前所有消息
  const contextTokens = messages
    .slice(0, index)
    .reduce((sum, m) => sum + estimateTokens(m.content) + estimateTokens(m.tool_calls || ""), 0);
  const inputTokens = OVERHEAD_TOKENS + contextTokens;

  // output = tool_calls + content
  const outputTokens = estimateTokens(msg.tool_calls) + estimateTokens(msg.content);

  return inputTokens + outputTokens;
}

type PanelType = "history" | "inspector" | null;

interface ChatPanelProps {
  messages: Message[];
  streaming: boolean;
  streamSegments: StreamSegment[];
  conversations: Conversation[];
  currentConvId: number | null;
  clusterId: string;
  onSelectConv: (id: number) => void;
  onNewConv: () => void;
  onDeleteConv: (id: number) => void;
  onSend: (message: string) => void;
  onStop: () => void;
}

export function ChatPanel({
  messages,
  streaming,
  streamSegments,
  conversations,
  currentConvId,
  clusterId,
  onSelectConv,
  onNewConv,
  onDeleteConv,
  onSend,
  onStop,
}: ChatPanelProps) {
  const [activePanel, setActivePanel] = useState<PanelType>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  const togglePanel = useCallback((panel: PanelType) => {
    setActivePanel((prev) => (prev === panel ? null : panel));
  }, []);

  // 点击外部关闭面板
  useEffect(() => {
    if (!activePanel) return;
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setActivePanel(null);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [activePanel]);

  // 自动滚动到底部
  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [messages, streamSegments]);

  return (
    <div className="flex-1 flex flex-col h-full min-w-0">
      {/* 顶部: 右侧功能按钮 */}
      <div className="flex items-center justify-end gap-1 px-3 py-2">
        <button
          onClick={() => togglePanel("history")}
          className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs transition-colors ${
            activePanel === "history"
              ? "bg-[var(--hover-bg)] text-default"
              : "text-muted hover:bg-[var(--hover-bg)] hover:text-default"
          }`}
          title="对话记录"
        >
          <History className="w-4 h-4" />
          <span>历史</span>
        </button>
        <button
          onClick={() => togglePanel("inspector")}
          className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs transition-colors ${
            activePanel === "inspector"
              ? "bg-[var(--hover-bg)] text-default"
              : "text-muted hover:bg-[var(--hover-bg)] hover:text-default"
          }`}
          title="AI 行为详情"
        >
          <Settings2 className="w-4 h-4" />
          <span>详情</span>
        </button>
      </div>

      {/* 浮动面板 */}
      {activePanel && (
        <div
          ref={panelRef}
          className="absolute right-3 top-12 z-30 w-[320px] max-h-[75vh] flex flex-col rounded-xl border border-[var(--border-color)] bg-card shadow-xl overflow-hidden"
        >
          {activePanel === "history" && (
            <ConversationPanel
              open={true}
              onClose={() => setActivePanel(null)}
              conversations={conversations}
              currentId={currentConvId}
              onSelect={onSelectConv}
              onNew={onNewConv}
              onDelete={onDeleteConv}
            />
          )}
          {activePanel === "inspector" && (
            <InspectorPanel
              messages={messages}
              streamSegments={streamSegments}
              streaming={streaming}
              clusterId={clusterId}
            />
          )}
        </div>
      )}

      {/* 消息区域 */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto py-6 space-y-6">
        {messages.length === 0 && !streaming ? (
          <EmptyState />
        ) : (
          <>
            {messages.map((msg, idx) => (
              <MessageBubble
                key={msg.id}
                message={msg}
                callTokens={getCallTokens(messages, idx)}
              />
            ))}
            {streaming && <StreamingBubble segments={streamSegments} />}
          </>
        )}
      </div>

      {/* 输入区域 */}
      <ChatInput
        onSend={onSend}
        onStop={onStop}
        streaming={streaming}
      />
    </div>
  );
}

// 空状态引导
function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-center px-8">
      <div className="w-16 h-16 rounded-2xl bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center mb-4">
        <Bot className="w-8 h-8 text-emerald-600 dark:text-emerald-400" />
      </div>
      <h3 className="text-lg font-semibold text-default mb-2">AI 集群助手</h3>
      <p className="text-sm text-muted max-w-sm mb-6">
        我可以帮你分析 Kubernetes 集群状态、诊断 Pod 问题、查看日志和事件。
      </p>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 w-full max-w-md">
        {[
          "为什么 Pod 一直重启？",
          "查看节点资源使用率",
          "分析最近的告警事件",
          "检查 Deployment 状态",
        ].map((q) => (
          <button
            key={q}
            className="text-left px-3 py-2 rounded-lg border border-[var(--border-color)] text-sm text-muted hover:bg-[var(--hover-bg)] hover:text-default transition-colors"
          >
            {q}
          </button>
        ))}
      </div>
    </div>
  );
}
