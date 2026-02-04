"use client";

import {
  Server,
  BarChart3,
  Shield,
  Check,
  X,
  Loader2,
  Brain,
} from "lucide-react";
import { Message, StreamSegment, ChatStats, Conversation } from "./types";
import { useI18n } from "@/i18n/context";

export interface InspectorPanelProps {
  messages: Message[];
  streamSegments: StreamSegment[];
  streaming: boolean;
  clusterId: string;
  currentStats?: ChatStats; // 当前提问的统计信息（streaming 结束后更新）
  currentConversation?: Conversation; // 当前对话（含累计统计）
}

export function InspectorPanel({
  messages,
  streamSegments,
  streaming,
  clusterId,
  currentStats,
  currentConversation,
}: InspectorPanelProps) {
  const { t } = useI18n();
  const inspectorT = t.aiChatPage.inspector;

  // 实时统计 streaming 中的 tool 调用（streaming 时显示）
  let streamingToolCalls = 0;
  for (const seg of streamSegments) {
    if (seg.type === "tool_call") streamingToolCalls++;
  }

  // 计算对话轮次（一问一答算一轮）
  const conversationRounds = messages.filter((m) => m.role === "user").length;

  // 计算对话历史中的总指令数（所有消息）
  let historyToolCalls = 0;
  for (const msg of messages) {
    if (msg.role === "assistant" && msg.tool_calls) {
      try {
        const calls = JSON.parse(msg.tool_calls);
        historyToolCalls += calls.length;
      } catch { /* ignore */ }
    }
  }

  return (
    <div className="flex flex-col overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)]/50">
        <h3 className="text-sm font-medium text-default">{inspectorT.title}</h3>
      </div>

      <div className="flex-1 overflow-y-auto">
        {/* 集群上下文 */}
        <Section icon={Server} title={inspectorT.clusterContext}>
          <div className="px-4 space-y-1.5">
            <InfoRow label={inspectorT.clusterId} value={clusterId || inspectorT.notSelected} />
            <InfoRow
              label={inspectorT.status}
              value={streaming ? inspectorT.querying : inspectorT.connected}
              highlight={streaming}
            />
          </div>
        </Section>

        {/* 本次提问统计 */}
        <Section icon={Brain} title={inspectorT.currentQuestion}>
          <div className="px-4 space-y-1.5">
            {streaming ? (
              <>
                <InfoRow
                  label={inspectorT.thinkingRounds}
                  value={inspectorT.inProgress}
                  highlight
                />
                <InfoRow
                  label={inspectorT.executedCommands}
                  value={`${streamingToolCalls} ${inspectorT.commandsUnit}`}
                  highlight={streamingToolCalls > 0}
                />
              </>
            ) : currentStats ? (
              <>
                <InfoRow
                  label={inspectorT.thinkingRounds}
                  value={`${currentStats.rounds} ${inspectorT.roundsUnit}`}
                />
                <InfoRow
                  label={inspectorT.executedCommands}
                  value={`${currentStats.total_tool_calls} ${inspectorT.commandsUnit}`}
                />
                <InfoRow
                  label={inspectorT.inputTokens}
                  value={currentStats.input_tokens.toLocaleString()}
                />
                <InfoRow
                  label={inspectorT.outputTokens}
                  value={currentStats.output_tokens.toLocaleString()}
                />
                <InfoRow
                  label={inspectorT.totalTokens}
                  value={(currentStats.input_tokens + currentStats.output_tokens).toLocaleString()}
                />
              </>
            ) : (
              <InfoRow label={inspectorT.status} value={inspectorT.waitingQuestion} />
            )}
          </div>
        </Section>

        {/* 对话统计 */}
        <Section icon={BarChart3} title={inspectorT.conversationOverview}>
          <div className="px-4 space-y-1.5">
            <InfoRow label={inspectorT.conversationRounds} value={`${conversationRounds} ${inspectorT.roundsUnit}`} />
            <InfoRow label={inspectorT.totalCommands} value={`${currentConversation?.total_tool_calls ?? historyToolCalls} ${inspectorT.commandsUnit}`} />
            <InfoRow
              label={inspectorT.totalTokensLabel}
              value={currentConversation
                ? `↑${currentConversation.total_input_tokens.toLocaleString()} ↓${currentConversation.total_output_tokens.toLocaleString()}`
                : "-"
              }
            />
          </div>
        </Section>

        {/* AI 能力边界 */}
        <Section icon={Shield} title={inspectorT.aiCapabilities}>
          <div className="px-4 space-y-2">
            <div className="space-y-1">
              <p className="text-[11px] font-medium text-secondary mb-1">{inspectorT.canDo}</p>
              <CapRow allowed text={inspectorT.capQueryResources} />
              <CapRow allowed text={inspectorT.capViewLogs} />
              <CapRow allowed text={inspectorT.capViewEvents} />
              <CapRow allowed text={inspectorT.capFilterByLabel} />
            </div>
            <div className="space-y-1 pt-1 border-t border-[var(--border-color)]">
              <p className="text-[11px] font-medium text-secondary mb-1">{inspectorT.cannotDo}</p>
              <CapRow allowed={false} text={inspectorT.capNoWrite} />
              <CapRow allowed={false} text={inspectorT.capNoSecrets} />
              <CapRow allowed={false} text={inspectorT.capNoSystemNs} />
              <CapRow allowed={false} text={inspectorT.capNoSensitive} />
            </div>
          </div>
        </Section>
      </div>
    </div>
  );
}

// Section 区块
function Section({ icon: Icon, title, children }: {
  icon: typeof Server;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="border-b border-[var(--border-color)] py-3">
      <div className="flex items-center gap-2 px-4 mb-2">
        <Icon className="w-3.5 h-3.5 text-muted" />
        <span className="text-xs font-medium text-secondary uppercase tracking-wide">{title}</span>
      </div>
      {children}
    </div>
  );
}

// 信息行
function InfoRow({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="flex items-center justify-between text-xs">
      <span className="text-muted">{label}</span>
      <span className={highlight ? "text-amber-500 font-medium flex items-center gap-1" : "text-default"}>
        {highlight && <Loader2 className="w-3 h-3 animate-spin" />}
        {value}
      </span>
    </div>
  );
}

// 能力行
function CapRow({ allowed, text }: { allowed: boolean; text: string }) {
  return (
    <div className="flex items-start gap-1.5 text-[11px]">
      {allowed ? (
        <Check className="w-3 h-3 text-emerald-500 flex-shrink-0 mt-0.5" />
      ) : (
        <X className="w-3 h-3 text-red-400 flex-shrink-0 mt-0.5" />
      )}
      <span className="text-muted">{text}</span>
    </div>
  );
}
