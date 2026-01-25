"use client";

import {
  Server,
  BarChart3,
  Shield,
  Check,
  X,
  Loader2,
} from "lucide-react";
import { Message, StreamSegment } from "./types";

export interface InspectorPanelProps {
  messages: Message[];
  streamSegments: StreamSegment[];
  streaming: boolean;
  clusterId: string;
}

export function InspectorPanel({
  messages,
  streamSegments,
  streaming,
  clusterId,
}: InspectorPanelProps) {
  // 统计 tool 调用次数
  let toolCallCount = 0;
  for (const msg of messages) {
    if (msg.role === "assistant" && msg.tool_calls) {
      try {
        const calls = JSON.parse(msg.tool_calls);
        toolCallCount += calls.length;
      } catch { /* ignore */ }
    }
  }
  for (const seg of streamSegments) {
    if (seg.type === "tool_call") toolCallCount++;
  }

  // 估算 token 总量
  const totalChars = messages.reduce(
    (sum, m) => sum + (m.content?.length || 0) + (m.tool_calls?.length || 0),
    0
  );
  const estimatedTokens = Math.ceil(totalChars * 0.7) + 550; // +550 system prompt overhead

  return (
    <div className="flex flex-col overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)]/50">
        <h3 className="text-sm font-medium text-default">对话详情</h3>
      </div>

      <div className="flex-1 overflow-y-auto">
        {/* 集群上下文 */}
        <Section icon={Server} title="集群上下文">
          <div className="px-4 space-y-1.5">
            <InfoRow label="集群 ID" value={clusterId || "未选择"} />
            <InfoRow
              label="状态"
              value={streaming ? "查询中" : "已连接"}
              highlight={streaming}
            />
          </div>
        </Section>

        {/* 对话统计 */}
        <Section icon={BarChart3} title="对话统计">
          <div className="px-4 space-y-1.5">
            <InfoRow label="消息数" value={`${messages.length}`} />
            <InfoRow label="Tool 调用" value={`${toolCallCount} 次`} />
            <InfoRow label="估算 Token" value={`~${estimatedTokens.toLocaleString()}`} />
            <InfoRow
              label="当前状态"
              value={streaming ? "生成中..." : "空闲"}
              highlight={streaming}
            />
          </div>
        </Section>

        {/* AI 能力边界 */}
        <Section icon={Shield} title="AI 能力边界">
          <div className="px-4 space-y-2">
            <div className="space-y-1">
              <p className="text-[11px] font-medium text-secondary mb-1">可执行</p>
              <CapRow allowed text="查询所有资源类型 (Pod, Node, Deployment...)" />
              <CapRow allowed text="查看 Pod 日志 (最近 200 行)" />
              <CapRow allowed text="查看 Event 和 ConfigMap" />
              <CapRow allowed text="按标签过滤资源" />
            </div>
            <div className="space-y-1 pt-1 border-t border-[var(--border-color)]">
              <p className="text-[11px] font-medium text-secondary mb-1">不可执行</p>
              <CapRow allowed={false} text="任何写操作 (创建/删除/重启/扩缩容)" />
              <CapRow allowed={false} text="查询 Secret 资源" />
              <CapRow allowed={false} text="访问 kube-system 等系统命名空间" />
              <CapRow allowed={false} text="输出密码/Token/API Key" />
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
