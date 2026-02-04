"use client";

import { useState } from "react";
import { Bot, User, ChevronDown, CheckCircle2, Loader2, Terminal, Brain } from "lucide-react";
import { Message, StreamSegment, ChatStats } from "./types";

// 检测是否是 ToolResult JSON
function isToolResultJSON(content: string): boolean {
  if (!content) return false;
  const trimmed = content.trim();
  return trimmed.startsWith('{"CallID":') || trimmed.startsWith('{"callid":');
}

// 格式化指令参数为易读形式
function formatCommandParams(name: string, paramsJson: string): string {
  try {
    const p = JSON.parse(paramsJson);

    // query_cluster 指令
    if (name === "query_cluster") {
      const action = p.action || "";
      const kind = p.kind || "";
      const ns = p.namespace || "";
      const resourceName = p.name || "";

      // 构建资源路径
      let resource = kind;
      if (ns && resourceName) {
        resource = `${kind} ${ns}/${resourceName}`;
      } else if (resourceName) {
        resource = `${kind} ${resourceName}`;
      } else if (ns) {
        resource = `${kind} -n ${ns}`;
      }

      switch (action) {
        case "describe":
          return `kubectl describe ${resource}`;
        case "get_logs":
          const container = p.container ? ` -c ${p.container}` : "";
          const tail = p.tail ? ` --tail=${p.tail}` : "";
          return `kubectl logs ${resource}${container}${tail}`;
        case "get_events":
          if (ns) {
            return `kubectl get events -n ${ns}${resourceName ? ` --field-selector involvedObject.name=${resourceName}` : ""}`;
          }
          return `kubectl get events`;
        case "list":
          return `kubectl get ${kind}${ns ? ` -n ${ns}` : " -A"}`;
        default:
          return `${action} ${resource}`.trim();
      }
    }

    // 其他指令：简化显示关键字段
    const parts: string[] = [];
    for (const [key, value] of Object.entries(p)) {
      if (value && typeof value === "string" && value.length < 50) {
        parts.push(`${key}=${value}`);
      }
    }
    return parts.join(" ") || paramsJson;
  } catch {
    return paramsJson;
  }
}

// 根据指令生成友好的标题
function formatCommandTitle(name: string, paramsJson: string): string {
  try {
    const p = JSON.parse(paramsJson);

    if (name === "query_cluster") {
      const action = p.action || "";
      const kind = p.kind || "";
      const resourceName = p.name || "";

      switch (action) {
        case "describe":
          return `查看 ${kind} 详情${resourceName ? `: ${resourceName}` : ""}`;
        case "get_logs":
          return `获取 ${kind} 日志${resourceName ? `: ${resourceName}` : ""}`;
        case "get_events":
          return `查询事件${p.namespace ? ` (${p.namespace})` : ""}`;
        case "list":
          return `列出 ${kind}${p.namespace ? ` (${p.namespace})` : ""}`;
        default:
          return `${action} ${kind}`;
      }
    }

    return name;
  } catch {
    return name;
  }
}

// ==================== 类型定义 ====================

type CommandStatus = "running" | "success" | "failed";

interface Command {
  id: string;
  name: string;
  params: string;
  status: CommandStatus;
  result?: string;
}

interface Round {
  thinking: string;
  commands: Command[];
}

// ==================== 从 StreamSegments 解析 Rounds ====================

function parseRoundsFromSegments(segments: StreamSegment[]): { rounds: Round[]; finalText: string } {
  const rounds: Round[] = [];
  let currentRound: Round | null = null;
  let finalText = "";
  let pendingToolCall: { name: string; params: string; id: string } | null = null;

  for (const seg of segments) {
    if (seg.type === "text" && !isToolResultJSON(seg.content)) {
      // 文本内容
      if (currentRound) {
        // 如果当前轮有指令，说明这是下一轮的思考
        if (currentRound.commands.length > 0) {
          rounds.push(currentRound);
          currentRound = { thinking: seg.content, commands: [] };
        } else {
          // 追加到当前轮的思考
          currentRound.thinking += seg.content;
        }
      } else {
        // 开始新的一轮
        currentRound = { thinking: seg.content, commands: [] };
      }
      finalText += seg.content;
    } else if (seg.type === "tool_call") {
      // 工具调用开始
      if (!currentRound) {
        currentRound = { thinking: "", commands: [] };
      }
      pendingToolCall = {
        name: seg.tool || "unknown",
        params: seg.params || "{}",
        id: `${seg.tool}-${Date.now()}-${Math.random()}`,
      };
      currentRound.commands.push({
        id: pendingToolCall.id,
        name: pendingToolCall.name,
        params: pendingToolCall.params,
        status: "running",
      });
    } else if (seg.type === "tool_result") {
      // 工具调用结果
      if (currentRound && currentRound.commands.length > 0) {
        // 找到对应的 running 命令并更新状态
        const lastCmd = currentRound.commands.find(
          (c) => c.status === "running" && c.name === seg.tool
        );
        if (lastCmd) {
          lastCmd.status = "success";
          lastCmd.result = seg.content;
        }
      }
      pendingToolCall = null;
    }
  }

  // 保存最后一轮
  if (currentRound && (currentRound.thinking || currentRound.commands.length > 0)) {
    rounds.push(currentRound);
  }

  return { rounds, finalText };
}

// ==================== RoundBlock ====================

const statusConfig = {
  running: { icon: Loader2, color: "text-yellow-500", animate: true },
  success: { icon: CheckCircle2, color: "text-green-500", animate: false },
  failed: { icon: CheckCircle2, color: "text-red-500", animate: false },
};

function RoundBlock({ round, roundIdx }: { round: Round; roundIdx: number }) {
  const [expanded, setExpanded] = useState(false);
  const cmdCount = round.commands.length;
  const allSuccess = round.commands.every((c) => c.status === "success");
  const hasRunning = round.commands.some((c) => c.status === "running");

  return (
    <div className={roundIdx > 0 ? "border-t border-[var(--border-color)]" : ""}>
      {/* 轮次标题行 */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full px-3 py-2 flex items-center gap-2 hover:bg-[var(--hover-bg)] transition-colors text-left"
      >
        <span className="text-xs font-medium text-muted flex-shrink-0">第 {roundIdx + 1} 轮</span>
        {!expanded && (
          <>
            <Brain className="w-3.5 h-3.5 text-purple-400 flex-shrink-0" />
            <span className="text-xs text-muted flex-1 truncate">{round.thinking || "执行中..."}</span>
          </>
        )}
        {expanded && <span className="flex-1" />}
        {cmdCount > 0 && <span className="text-xs text-muted">{cmdCount} 条指令</span>}
        <span className={`text-xs ${hasRunning ? "text-yellow-400" : allSuccess ? "text-green-400" : "text-red-400"}`}>
          {hasRunning ? "执行中" : allSuccess ? "成功" : "失败"}
        </span>
        <ChevronDown className={`w-4 h-4 text-muted flex-shrink-0 transition-transform ${expanded ? "" : "-rotate-90"}`} />
      </button>

      {/* 展开后显示完整思考 + 指令列表 */}
      {expanded && (
        <div className="border-t border-[var(--border-color)]/50 bg-[var(--hover-bg)]/30">
          {/* 完整思考内容 */}
          {round.thinking && (
            <div className="px-3 py-2 flex items-start gap-2">
              <Brain className="w-3.5 h-3.5 text-purple-400 mt-0.5 flex-shrink-0" />
              <p className="text-xs text-muted leading-relaxed whitespace-pre-wrap">{round.thinking}</p>
            </div>
          )}

          {/* 该轮执行的指令 */}
          {round.commands.length > 0 && (
            <div className="border-t border-[var(--border-color)]/50">
              {round.commands.map((cmd, cmdIdx) => {
                const config = statusConfig[cmd.status];
                const StatusIcon = config.icon;

                return (
                  <div
                    key={`${cmd.id}-${cmdIdx}`}
                    className={`flex items-start gap-2 px-3 py-2 ${cmdIdx !== round.commands.length - 1 ? "border-b border-[var(--border-color)]/30" : ""}`}
                  >
                    <StatusIcon className={`w-4 h-4 flex-shrink-0 mt-0.5 ${config.color} ${config.animate ? "animate-spin" : ""}`} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-0.5">
                        <span className="text-sm text-default font-medium">{formatCommandTitle(cmd.name, cmd.params)}</span>
                      </div>
                      <code className="text-xs text-muted font-mono block">
                        $ {formatCommandParams(cmd.name, cmd.params)}
                      </code>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ==================== ExecutionBlock ====================

interface ExecutionBlockProps {
  rounds: Round[];
  stats?: ChatStats;
  streaming?: boolean;
}

function ExecutionBlock({ rounds, stats, streaming }: ExecutionBlockProps) {
  const [expanded, setExpanded] = useState(false);

  const totalCommands = rounds.reduce((sum, r) => sum + r.commands.length, 0);
  const allSuccess = rounds.every((r) => r.commands.every((c) => c.status === "success"));
  const hasRunning = rounds.some((r) => r.commands.some((c) => c.status === "running"));

  // 如果没有指令，不显示 ExecutionBlock
  if (rounds.length === 0 || totalCommands === 0) {
    return null;
  }

  return (
    <div className="my-2 rounded-lg border border-[var(--border-color)] overflow-hidden bg-card">
      {/* 标题栏 */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-3 py-2 bg-[var(--hover-bg)] hover:bg-[var(--hover-bg)]/80 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-muted" />
          <div className="flex items-center gap-2 text-xs text-muted">
            <span>思考 {rounds.length} 轮</span>
            <span className="text-muted/50">·</span>
            <span>指令 {totalCommands} 条</span>
            {stats && (
              <>
                <span className="text-muted/50">·</span>
                <span className="text-emerald-500/80">↑{stats.input_tokens.toLocaleString()}</span>
                <span className="text-blue-500/80">↓{stats.output_tokens.toLocaleString()}</span>
              </>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className={`text-xs font-medium ${hasRunning || streaming ? "text-yellow-400" : allSuccess ? "text-green-400" : "text-red-400"}`}>
            {hasRunning || streaming ? "执行中..." : allSuccess ? "完成" : "有失败"}
          </span>
          <ChevronDown className={`w-4 h-4 text-muted transition-transform ${expanded ? "" : "-rotate-90"}`} />
        </div>
      </button>

      {/* 展开后按轮次显示 */}
      {expanded && (
        <div className="border-t border-[var(--border-color)]">
          {rounds.map((round, roundIdx) => (
            <RoundBlock key={roundIdx} round={round} roundIdx={roundIdx} />
          ))}
        </div>
      )}
    </div>
  );
}

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

  // Assistant message - 解析 tool_calls 显示 ExecutionBlock
  const content = message.content && !isToolResultJSON(message.content) ? message.content : "";

  // 从 tool_calls 解析 rounds（每3个 tool_call 为一轮，模拟多轮思考）
  let rounds: Round[] = [];
  if (message.tool_calls) {
    try {
      const toolCalls = JSON.parse(message.tool_calls);
      if (Array.isArray(toolCalls) && toolCalls.length > 0) {
        // 将 tool_calls 按轮次分组（启发式：每轮最多3个指令）
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
