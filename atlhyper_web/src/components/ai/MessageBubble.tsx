"use client";

import { Bot, User } from "lucide-react";
import { Message, StreamSegment, ChatStats } from "./types";
import { CommandStatus, Round, isToolResultJSON, parseRoundsFromSegments } from "./command-utils";
import { ExecutionBlock } from "./ExecutionBlock";

// ==================== MessageBubble ====================

interface MessageBubbleProps {
  message: Message;
  stats?: ChatStats; // 当前提问的统计信息（只有最后一条 assistant 消息需要）
}

export function MessageBubble({ message, stats }: MessageBubbleProps) {
  const isUser = message.role === "user";

  if (isUser) {
    return (
      <div className="flex justify-end gap-3">
        <div className="max-w-[85%] bg-primary text-white rounded-2xl rounded-br-sm px-4 py-2.5 text-sm whitespace-pre-wrap break-words">
          {message.content}
        </div>
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
          <User className="w-4 h-4 text-primary" />
        </div>
      </div>
    );
  }

  // Assistant message - 解析 toolCalls 显示 ExecutionBlock
  const content = message.content && !isToolResultJSON(message.content) ? message.content : "";

  // 从 toolCalls 解析 rounds（每3个 tool_call 为一轮，模拟多轮思考）
  let rounds: Round[] = [];
  if (message.toolCalls) {
    try {
      const toolCalls = JSON.parse(message.toolCalls);
      if (Array.isArray(toolCalls) && toolCalls.length > 0) {
        // 将 toolCalls 按轮次分组（启发式：每轮最多3个指令）
        const commands = toolCalls.map((tc: { ID?: string; Name?: string; Params?: string }, idx: number) => ({
          id: tc.ID || `cmd-${idx}`,
          name: tc.Name || "unknown",
          params: tc.Params || "{}",
          status: "success" as CommandStatus,
        }));

        // 按每3个指令分组为一轮
        for (let i = 0; i < commands.length; i += 3) {
          rounds.push({
            thinking: "",
            commands: commands.slice(i, i + 3),
          });
        }
      }
    } catch {
      // 忽略解析错误
    }
  }

  if (!content && rounds.length === 0) {
    return null;
  }

  return (
    <div className="flex gap-3">
      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center">
        <Bot className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
      </div>
      <div className="flex-1 min-w-0">
        {/* ExecutionBlock - 如果有 tool calls */}
        {rounds.length > 0 && <ExecutionBlock rounds={rounds} stats={stats} />}

        {/* 文本内容 */}
        {content && (
          <div className="bg-card border border-[var(--border-color)] rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-default whitespace-pre-wrap">
            {content}
          </div>
        )}
      </div>
    </div>
  );
}

// ==================== StreamingBubble ====================

interface StreamingBubbleProps {
  segments: StreamSegment[];
  stats?: ChatStats;
  streaming?: boolean; // 是否正在流式传输
}

export function StreamingBubble({ segments, stats, streaming = true }: StreamingBubbleProps) {
  // 解析 segments 为 rounds 结构
  const { rounds, finalText } = parseRoundsFromSegments(segments);

  // 错误信息
  const errorSegment = segments.find((seg) => seg.type === "error");

  // 检查是否有 tool 调用（用于显示 ExecutionBlock）
  const hasToolCalls = rounds.length > 0 && rounds.some(r => r.commands.length > 0);

  // streaming=false 时不显示（stats 已传递给 MessageBubble）
  // 但如果有错误，需要显示错误信息
  if (!streaming && !errorSegment) {
    return null;
  }

  // 完全没有内容时显示加载动画
  if (!finalText && !errorSegment && rounds.length === 0) {
    return (
      <div className="flex gap-3">
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
    <div className="flex gap-3">
      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center">
        <Bot className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
      </div>
      <div className="flex-1 min-w-0 space-y-2">
        {/* 错误信息 */}
        {errorSegment && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-red-700 dark:text-red-300">
            {errorSegment.content}
          </div>
        )}

        {/* ExecutionBlock - 显示思考轮次和指令 */}
        {hasToolCalls && (
          <ExecutionBlock rounds={rounds} stats={stats} streaming={true} />
        )}

        {/* 文本内容 */}
        {finalText && (
          <div className="bg-card border border-[var(--border-color)] rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-default whitespace-pre-wrap">
            {finalText}
            <span className="inline-block w-1 h-4 bg-primary animate-pulse ml-0.5 align-text-bottom" />
          </div>
        )}
      </div>
    </div>
  );
}
