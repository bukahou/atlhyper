"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { Bot, History, Settings2 } from "lucide-react";
import { ConversationPanel } from "./ConversationSidebar";
import { InspectorPanel } from "./InspectorPanel";
import { Conversation, ChatStats } from "./types";
import { Message, StreamSegment } from "./types";
import { MessageBubble, StreamingBubble } from "./MessageBubble";
import { ChatInput } from "./ChatInput";

// 合并 tool_calls JSON 字符串
function mergeToolCalls(tc1?: string, tc2?: string): string | undefined {
  if (!tc1 && !tc2) return undefined;
  if (!tc1) return tc2;
  if (!tc2) return tc1;
  try {
    const arr1 = JSON.parse(tc1);
    const arr2 = JSON.parse(tc2);
    return JSON.stringify([...arr1, ...arr2]);
  } catch {
    return tc1 || tc2;
  }
}

// 消息分组（过滤 tool 消息，合并连续的 assistant 消息）
function groupMessages(messages: Message[]): Message[] {
  const result: Message[] = [];
  let pendingAssistant: Message | null = null;

  for (const msg of messages) {
    // tool 消息不显示（仅用于 API 上下文）
    if (msg.role === "tool") continue;

    if (msg.role === "user") {
      // user 消息前，保存 pending assistant
      if (pendingAssistant) {
        result.push(pendingAssistant);
        pendingAssistant = null;
      }
      result.push(msg);
    } else if (msg.role === "assistant") {
      if (pendingAssistant) {
        // 合并连续的 assistant 消息
        const merged: Message = {
          id: pendingAssistant.id,
          conversation_id: pendingAssistant.conversation_id,
          role: "assistant",
          content: [pendingAssistant.content, msg.content].filter(Boolean).join("\n\n"),
          tool_calls: mergeToolCalls(pendingAssistant.tool_calls, msg.tool_calls),
          created_at: pendingAssistant.created_at,
        };
        pendingAssistant = merged;
      } else {
        pendingAssistant = { ...msg };
      }
    }
  }

  // 保存最后的 pending assistant
  if (pendingAssistant) {
    result.push(pendingAssistant);
  }

  return result;
}

type PanelType = "history" | "inspector" | null;

interface ChatPanelProps {
  messages: Message[];
  streaming: boolean;
  streamSegments: StreamSegment[];
  conversations: Conversation[];
  currentConvId: number | null;
  clusterId: string;
  currentStats?: ChatStats; // 当前提问的统计信息
  onSelectConv: (id: number) => void;
  onNewConv: () => void;
  onDeleteConv: (id: number) => void;
  onSend: (message: string) => void;
  onStop: () => void;
  onQuickQuestion?: (question: string) => void; // 快捷问题点击回调
}

export function ChatPanel({
  messages,
  streaming,
  streamSegments,
  conversations,
  currentConvId,
  clusterId,
  currentStats,
  onSelectConv,
  onNewConv,
  onDeleteConv,
  onSend,
  onStop,
  onQuickQuestion,
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
    <div className="flex-1 flex flex-col h-full min-w-0 relative">
      {/* 右侧垂直浮动按钮栏 */}
      <div className="absolute right-3 top-1/2 -translate-y-1/2 z-20 flex flex-col gap-2">
        <button
          onClick={() => togglePanel("history")}
          className={`group relative w-10 h-10 rounded-xl flex items-center justify-center transition-all ${
            activePanel === "history"
              ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shadow-sm"
              : "bg-card text-muted hover:bg-[var(--hover-bg)] hover:text-default shadow-sm border border-[var(--border-color)]"
          }`}
        >
          <History className="w-5 h-5" />
          {/* Tooltip */}
          <span className="absolute right-12 px-2 py-1 rounded-md bg-gray-900 dark:bg-gray-700 text-white text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity">
            对话历史
          </span>
        </button>
        <button
          onClick={() => togglePanel("inspector")}
          className={`group relative w-10 h-10 rounded-xl flex items-center justify-center transition-all ${
            activePanel === "inspector"
              ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shadow-sm"
              : "bg-card text-muted hover:bg-[var(--hover-bg)] hover:text-default shadow-sm border border-[var(--border-color)]"
          }`}
        >
          <Settings2 className="w-5 h-5" />
          {/* Tooltip */}
          <span className="absolute right-12 px-2 py-1 rounded-md bg-gray-900 dark:bg-gray-700 text-white text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity">
            执行详情
          </span>
        </button>
      </div>

      {/* 浮动面板 */}
      {activePanel && (
        <div
          ref={panelRef}
          className="absolute right-16 top-1/2 -translate-y-1/2 z-30 w-[320px] max-h-[70vh] flex flex-col rounded-xl border border-[var(--border-color)] bg-card shadow-xl overflow-hidden"
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
              currentStats={currentStats}
              currentConversation={conversations.find(c => c.id === currentConvId)}
            />
          )}
        </div>
      )}

      {/* 消息区域 */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto py-6">
        <div className="w-full px-4 sm:px-6 md:w-[85%] lg:w-[75%] mx-auto space-y-6">
          {messages.length === 0 && !streaming ? (
            <EmptyState onQuickQuestion={onQuickQuestion || onSend} />
          ) : (
            <>
              {(() => {
                const grouped = groupMessages(messages);
                return grouped.map((msg, idx) => {
                  // 最后一条 assistant 消息传递 stats（用于显示 token 统计）
                  const isLastAssistant =
                    idx === grouped.length - 1 &&
                    msg.role === "assistant" &&
                    !streaming; // 只有完成后才显示 stats
                  return (
                    <MessageBubble
                      key={msg.id}
                      message={msg}
                      stats={isLastAssistant ? currentStats : undefined}
                    />
                  );
                });
              })()}
              {/* 流式渲染：streaming 时显示，或有错误时显示 */}
              {(streaming || streamSegments.some(s => s.type === "error")) && (
                <StreamingBubble segments={streamSegments} stats={currentStats} streaming={streaming} />
              )}
            </>
          )}
        </div>
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

// 快捷问题列表
const QUICK_QUESTIONS = [
  "为什么 Pod 一直重启？",
  "查看节点资源使用率",
  "分析最近的告警事件",
  "检查 Deployment 状态",
];

// 空状态引导
function EmptyState({ onQuickQuestion }: { onQuickQuestion: (q: string) => void }) {
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
        {QUICK_QUESTIONS.map((q) => (
          <button
            key={q}
            onClick={() => onQuickQuestion(q)}
            className="text-left px-3 py-2 rounded-lg border border-[var(--border-color)] text-sm text-muted hover:bg-[var(--hover-bg)] hover:text-default hover:border-emerald-500/50 transition-colors cursor-pointer"
          >
            {q}
          </button>
        ))}
      </div>
    </div>
  );
}
