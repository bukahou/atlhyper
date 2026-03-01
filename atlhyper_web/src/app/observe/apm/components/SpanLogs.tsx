"use client";

import { useState, useEffect } from "react";
import type { ApmTranslations } from "@/types/i18n";
import type { LogEntry } from "@/types/model/log";
import { useClusterStore } from "@/store/clusterStore";
import { queryLogs } from "@/datasource/logs";

interface SpanLogsProps {
  t: ApmTranslations;
  traceId: string;
  serviceName?: string;
  compact?: boolean;
}

export function SpanLogs({ t, traceId, serviceName, compact }: SpanLogsProps) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const clusterId = useClusterStore((s) => s.currentClusterId);

  useEffect(() => {
    if (!clusterId) { setLoading(false); return; }
    setLoading(true);
    queryLogs({
      clusterId,
      traceId,
      services: serviceName ? [serviceName] : undefined,
      limit: compact ? 20 : 100,
    })
      .then((result) => setLogs(result.logs))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [clusterId, traceId, serviceName, compact]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-muted">
        <div className="w-4 h-4 border-2 border-primary/30 border-t-primary rounded-full animate-spin mr-2" />
        {t.loading}
      </div>
    );
  }

  if (logs.length === 0) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-muted">
        {t.noCorrelatedLogs}
      </div>
    );
  }

  return (
    <div className={`overflow-auto ${compact ? "max-h-[300px]" : "max-h-[500px]"}`}>
      {logs.map((log, i) => {
        const severityClass =
          log.severity === "ERROR" ? "bg-red-500/10 text-red-500" :
          log.severity === "WARN" ? "bg-amber-500/10 text-amber-500" :
          log.severity === "DEBUG" ? "bg-gray-500/10 text-gray-500" :
          "bg-blue-500/10 text-blue-500";
        return (
          <div key={i} className="flex items-start gap-2 px-4 py-2 border-b border-[var(--border-color)]/20 hover:bg-[var(--hover-bg)] text-xs">
            <span className="text-[10px] text-muted flex-shrink-0 w-[70px] pt-0.5">
              {new Date(log.timestamp).toLocaleTimeString()}
            </span>
            <span className={`px-1.5 py-0.5 rounded text-[10px] font-semibold flex-shrink-0 ${severityClass}`}>
              {log.severity}
            </span>
            <span className="text-muted flex-shrink-0 max-w-[120px] truncate">{log.serviceName}</span>
            <span className="text-default break-all">{log.body}</span>
          </div>
        );
      })}
    </div>
  );
}
