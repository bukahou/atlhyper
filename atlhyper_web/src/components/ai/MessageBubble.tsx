"use client";

import { Bot, User } from "lucide-react";
import { Message, StreamSegment } from "./types";
import { ToolCallBlock } from "./ToolCallBlock";

interface MessageBubbleProps {
  message: Message;
  callTokens?: number; // 本次 API 调用总 token (含完整上下文)
}

// 解析 assistant 消息的 tool_calls 为渲染段
function parseAssistantSegments(msg: Message): StreamSegment[] {
  const segments: StreamSegment[] = [];

  // 如果有 tool_calls，构建 tool 段
  if (msg.tool_calls) {
    try {
      const calls = JSON.parse(msg.tool_calls) as Array<{
        id: string;
        name: string;
        params: string;
      }>;
      for (const call of calls) {
        segments.push({
          type: "tool_call",
          tool: call.name,
          params: call.params,
          content: "",
        });
      }
    } catch {
      // ignore parse error
    }
  }

  // 文本内容
  if (msg.content) {
    segments.push({ type: "text", content: msg.content });
  }

  return segments;
}

export function MessageBubble({ message, callTokens }: MessageBubbleProps) {
  const isUser = message.role === "user";

  if (isUser) {
    return (
      <div className="flex justify-end gap-3 px-4">
        <div className="max-w-[70%] bg-primary text-white rounded-2xl rounded-br-sm px-4 py-2.5 text-sm whitespace-pre-wrap">
          {message.content}
        </div>
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
          <User className="w-4 h-4 text-primary" />
        </div>
      </div>
    );
  }

  // Assistant message
  const segments = parseAssistantSegments(message);

  return (
    <div className="flex gap-3 px-4">
      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center">
        <Bot className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
      </div>
      <div className="max-w-[75%] space-y-1">
        {segments.map((seg, i) => {
          if (seg.type === "tool_call") {
            return <ToolCallBlock key={i} segment={seg} tokens={callTokens} />;
          }
          if (seg.type === "tool_result") {
            return null;
          }
          // text
          return (
            <div
              key={i}
              className="bg-card border border-[var(--border-color)] rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-default whitespace-pre-wrap"
            >
              {seg.content}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// 流式消息 (正在生成中)
interface StreamingBubbleProps {
  segments: StreamSegment[];
}

export function StreamingBubble({ segments }: StreamingBubbleProps) {
  if (segments.length === 0) {
    return (
      <div className="flex gap-3 px-4">
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center">
          <Bot className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div className="bg-card border border-[var(--border-color)] rounded-2xl rounded-bl-sm px-4 py-2.5">
          <div className="flex gap-1">
            <span className="w-2 h-2 bg-muted rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
            <span className="w-2 h-2 bg-muted rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
            <span className="w-2 h-2 bg-muted rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex gap-3 px-4">
      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center">
        <Bot className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
      </div>
      <div className="max-w-[75%] space-y-1">
        {segments.map((seg, i) => {
          if (seg.type === "tool_call") {
            return <ToolCallBlock key={i} segment={seg} />;
          }
          if (seg.type === "tool_result") {
            return null;
          }
          if (seg.type === "error") {
            return (
              <div key={i} className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-red-700 dark:text-red-300">
                {seg.content}
              </div>
            );
          }
          return (
            <div
              key={i}
              className="bg-card border border-[var(--border-color)] rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-default whitespace-pre-wrap"
            >
              {seg.content}
              <span className="inline-block w-1 h-4 bg-primary animate-pulse ml-0.5 align-text-bottom" />
            </div>
          );
        })}
      </div>
    </div>
  );
}
