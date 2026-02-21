"use client";

import { X, CheckCircle2, XCircle, AlertCircle, Loader2, Clock } from "lucide-react";
import { StatusBadge } from "@/components/common";
import type { CommandHistory } from "@/api/commands";
import type { useI18n } from "@/i18n/context";

const statusConfig: Record<
  string,
  { icon: typeof CheckCircle2; color: string; badgeType: "success" | "error" | "warning" | "info" | "default" }
> = {
  success: { icon: CheckCircle2, color: "text-green-500", badgeType: "success" },
  failed: { icon: XCircle, color: "text-red-500", badgeType: "error" },
  timeout: { icon: AlertCircle, color: "text-orange-500", badgeType: "warning" },
  running: { icon: Loader2, color: "text-blue-500", badgeType: "info" },
  pending: { icon: Clock, color: "text-gray-500", badgeType: "default" },
};

interface CommandDetailModalProps {
  command: CommandHistory;
  onClose: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

export function CommandDetailModal({ command, onClose, t }: CommandDetailModalProps) {
  const config = statusConfig[command.status] || statusConfig.pending;
  const Icon = config.icon;
  const statusLabel = t.commands.statuses[command.status as keyof typeof t.commands.statuses] || command.status;
  const actionLabel = t.commands.actions[command.action as keyof typeof t.commands.actions] || command.action;
  const sourceLabel = t.commands.sources[command.source as keyof typeof t.commands.sources] || command.source;

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const details = [
    { label: t.commands.commandId, value: command.commandId },
    { label: t.commands.source, value: sourceLabel },
    { label: t.common.action, value: actionLabel },
    { label: t.common.type, value: command.targetKind },
    { label: t.common.namespace, value: command.targetNamespace },
    { label: t.common.name, value: command.targetName },
    { label: t.common.status, value: statusLabel },
    { label: t.commands.duration, value: command.durationMs > 0 ? formatDuration(command.durationMs) : "-" },
    { label: t.commands.createdAt, value: command.createdAt ? new Date(command.createdAt).toLocaleString() : "-" },
    { label: t.commands.startedAt, value: command.startedAt ? new Date(command.startedAt).toLocaleString() : "-" },
    { label: t.commands.finishedAt, value: command.finishedAt ? new Date(command.finishedAt).toLocaleString() : "-" },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-card rounded-xl border border-[var(--border-color)] shadow-xl w-full max-w-2xl mx-4 max-h-[80vh] overflow-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-3">
            <Icon className={`w-6 h-6 ${config.color}`} />
            <div>
              <h2 className="text-lg font-semibold text-default">{t.common.details}</h2>
              <StatusBadge status={statusLabel} type={config.badgeType} />
            </div>
          </div>
          <button onClick={onClose} className="p-2 hover-bg rounded-lg">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* 详细信息 */}
          <div>
            <h3 className="text-sm font-semibold text-default mb-3">{t.common.details}</h3>
            <div className="grid grid-cols-2 gap-4">
              {details.map((item, i) => (
                <div key={i} className="bg-[var(--background)] rounded-lg p-3">
                  <div className="text-xs text-muted mb-1">{item.label}</div>
                  <div className="text-sm text-default font-medium break-all">{item.value || "-"}</div>
                </div>
              ))}
            </div>
          </div>

          {/* 参数 */}
          {command.params && (
            <div>
              <h3 className="text-sm font-semibold text-default mb-2">{t.commands.params}</h3>
              <div className="bg-[var(--background)] rounded-lg p-4">
                <pre className="text-sm text-default whitespace-pre-wrap font-mono">
                  {(() => {
                    try {
                      return JSON.stringify(JSON.parse(command.params), null, 2);
                    } catch {
                      return command.params;
                    }
                  })()}
                </pre>
              </div>
            </div>
          )}

          {/* 结果 */}
          {command.result && (
            <div>
              <h3 className="text-sm font-semibold text-default mb-2">{t.commands.result}</h3>
              <div className="bg-[var(--background)] rounded-lg p-4">
                <pre className="text-sm text-default whitespace-pre-wrap font-mono">
                  {(() => {
                    try {
                      return JSON.stringify(JSON.parse(command.result), null, 2);
                    } catch {
                      return command.result;
                    }
                  })()}
                </pre>
              </div>
            </div>
          )}

          {/* 错误信息 */}
          {command.errorMessage && (
            <div>
              <h3 className="text-sm font-semibold text-red-500 mb-2">{t.commands.errorMessage}</h3>
              <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4 border border-red-200 dark:border-red-800">
                <p className="text-sm text-red-600 dark:text-red-400 whitespace-pre-wrap">
                  {command.errorMessage}
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
