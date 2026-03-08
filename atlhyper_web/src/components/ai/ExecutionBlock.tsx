"use client";

import { useState } from "react";
import { useI18n } from "@/i18n/context";
import { ChevronDown, CheckCircle2, Loader2, Terminal, Brain } from "lucide-react";
import { ChatStats } from "./types";
import { Round, formatCommandTitle, formatCommandParams } from "./command-utils";

// ==================== RoundBlock ====================

const statusConfig = {
  running: { icon: Loader2, color: "text-yellow-500", animate: true },
  success: { icon: CheckCircle2, color: "text-green-500", animate: false },
  failed: { icon: CheckCircle2, color: "text-red-500", animate: false },
};

function RoundBlock({ round, roundIdx }: { round: Round; roundIdx: number }) {
  const { t } = useI18n();
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
        <span className="text-xs font-medium text-muted flex-shrink-0">{t.aiChatPage.execution.roundLabel.replace("{n}", String(roundIdx + 1))}</span>
        {!expanded && (
          <>
            <Brain className="w-3.5 h-3.5 text-purple-400 flex-shrink-0" />
            <span className="text-xs text-muted flex-1 truncate">{round.thinking || t.aiChatPage.execution.executing}</span>
          </>
        )}
        {expanded && <span className="flex-1" />}
        {cmdCount > 0 && <span className="text-xs text-muted">{t.aiChatPage.execution.commandsUnit.replace("{n}", String(cmdCount))}</span>}
        <span className={`text-xs ${hasRunning ? "text-yellow-400" : allSuccess ? "text-green-400" : "text-red-400"}`}>
          {hasRunning ? t.aiChatPage.execution.executing : allSuccess ? t.aiChatPage.execution.success : t.aiChatPage.execution.failed}
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

export function ExecutionBlock({ rounds, stats, streaming }: ExecutionBlockProps) {
  const { t } = useI18n();
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
            <span>{t.aiChatPage.execution.thinkingRounds.replace("{n}", String(rounds.length))}</span>
            <span className="text-muted/50">·</span>
            <span>{t.aiChatPage.execution.commandsCount.replace("{n}", String(totalCommands))}</span>
            {stats && (
              <>
                <span className="text-muted/50">·</span>
                <span className="text-emerald-500/80">↑{stats.inputTokens.toLocaleString()}</span>
                <span className="text-blue-500/80">↓{stats.outputTokens.toLocaleString()}</span>
              </>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className={`text-xs font-medium ${hasRunning || streaming ? "text-yellow-400" : allSuccess ? "text-green-400" : "text-red-400"}`}>
            {hasRunning || streaming ? t.aiChatPage.execution.executing : allSuccess ? t.aiChatPage.execution.completed : t.aiChatPage.execution.hasFailed}
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
