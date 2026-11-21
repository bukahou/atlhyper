"use client";

import { Bot, User } from "lucide-react";
import type { Message, StreamSegment, ChatStats } from "./types";
import { type ParsedAnswer, parseRoundsFromSegments } from "./command-utils";
import { ExecutionBlock } from "./ExecutionBlock";
import { MarkdownContent } from "./MarkdownContent";
import { useI18n } from "@/i18n/context";

// ==================== UserBubble ====================

export function UserBubble({ message }: { message: Message }) {
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

// ==================== AnswerBubble ====================

interface AnswerBubbleProps {
  answer: ParsedAnswer;
  stats?: ChatStats; // 当前提问的统计信息（只有最后一次回答需要）
}

/** 历史里的一次完整回答：各轮（执行块）+ 最终回答 */
export function AnswerBubble({ answer, stats }: AnswerBubbleProps) {
  const { rounds, finalText } = answer;
  if (!finalText && rounds.length === 0) return null;

  return (
    <div className="flex gap-3">
      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-emerald-100 dark:bg-emerald-900/30 flex items-center justify-center">
        <Bot className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
      </div>
      <div className="flex-1 min-w-0">
        {rounds.length > 0 && <ExecutionBlock rounds={rounds} stats={stats} />}
        {finalText && (
          <div className="bg-card border border-[var(--border-color)] rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-default">
            <MarkdownContent content={finalText} />
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
  stopped?: boolean; // 用户停止了生成
}

export function StreamingBubble({ segments, stats, streaming = true, stopped = false }: StreamingBubbleProps) {
  const { t } = useI18n();
  // 按分段自带的轮次与调用 ID 解析
  const { rounds, finalText, notices } = parseRoundsFromSegments(segments);

  // 错误信息
  const errorSegment = segments.find((seg) => seg.type === "error");

  // 检查是否有 tool 调用（用于显示 ExecutionBlock）
  const hasToolCalls = rounds.length > 0 && rounds.some(r => r.commands.length > 0);

  // 正常结束后不显示（回答已在服务端历史里）；停止或出错时继续显示已输出的内容
  if (!streaming && !errorSegment && !stopped) {
    return null;
  }

  // 完全没有内容：还在输出时显示加载动画，停止了就什么都不显示
  if (!finalText && !errorSegment && rounds.length === 0) {
    if (!streaming) return null;
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

        {/* 流里的提示（历史已裁剪、调用超上限）：文案按语言出 */}
        {notices.map((notice, i) => (
          <p key={`${notice}-${i}`} className="text-xs text-muted px-1">{t.aiChatPage.notice[notice]}</p>
        ))}

        {/* ExecutionBlock - 显示思考轮次和指令 */}
        {hasToolCalls && (
          <ExecutionBlock rounds={rounds} stats={stats} streaming={streaming} />
        )}

        {/* 文本内容 */}
        {finalText && (
          <div className="bg-card border border-[var(--border-color)] rounded-2xl rounded-bl-sm px-4 py-2.5 text-sm text-default">
            <MarkdownContent content={finalText} />
            {streaming && (
              <span className="inline-block w-1 h-4 bg-primary animate-pulse ml-0.5 align-text-bottom" />
            )}
          </div>
        )}

        {stopped && <p className="text-xs text-muted px-1">{t.aiChatPage.chatInput.generationStopped}</p>}
      </div>
    </div>
  );
}
