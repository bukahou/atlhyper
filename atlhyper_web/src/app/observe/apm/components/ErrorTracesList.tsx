"use client";

import { useState, useMemo } from "react";
import type { TraceSummary } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { AlertTriangle } from "lucide-react";

interface ErrorTracesListProps {
  t: ApmTranslations;
  traces: TraceSummary[];
  onSelectTrace?: (traceId: string) => void;
}

export function ErrorTracesList({ t, traces, onSelectTrace }: ErrorTracesListProps) {
  const [page, setPage] = useState(0);
  const pageSize = 10;
  const totalPages = Math.max(1, Math.ceil(traces.length / pageSize));
  const paged = useMemo(() => traces.slice(page * pageSize, (page + 1) * pageSize), [traces, page]);

  if (traces.length === 0) {
    return (
      <div className="border border-[var(--border-color)] rounded-xl p-8 bg-card text-center">
        <AlertTriangle className="w-8 h-8 mx-auto mb-2 text-muted" />
        <p className="text-sm text-muted">{t.noErrors}</p>
      </div>
    );
  }

  return (
    <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
      <h3 className="text-sm font-medium text-default mb-3">{t.errorTraces} ({traces.length})</h3>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-muted text-xs border-b border-[var(--border-color)]">
              <th className="pb-2 pr-4">{t.traceId}</th>
              <th className="pb-2 pr-4">{t.rootOperation}</th>
              <th className="pb-2 pr-4">{t.exceptionType}</th>
              <th className="pb-2 pr-4">{t.exceptionMessage}</th>
              <th className="pb-2 pr-4 text-right">{t.duration}</th>
              <th className="pb-2 pr-4 text-right">{t.spans}</th>
              <th className="pb-2 text-right">{t.startTime}</th>
            </tr>
          </thead>
          <tbody>
            {paged.map((tr) => (
              <tr
                key={tr.traceId}
                onClick={() => onSelectTrace?.(tr.traceId)}
                className="border-b border-[var(--border-color)] last:border-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                <td className="py-2 pr-4 font-mono text-xs text-red-500">{tr.traceId.slice(0, 16)}...</td>
                <td className="py-2 pr-4 text-default truncate max-w-[200px]">{tr.rootOperation}</td>
                <td className="py-2 pr-4 text-red-400 text-xs">{tr.errorType ? tr.errorType.split('.').pop() : '-'}</td>
                <td className="py-2 pr-4 text-default text-xs truncate max-w-[240px]">{tr.errorMessage || '-'}</td>
                <td className="py-2 pr-4 text-right text-default">{formatDurationMs(tr.durationMs)}</td>
                <td className="py-2 pr-4 text-right text-muted">{tr.spanCount}</td>
                <td className="py-2 text-right text-muted text-xs">{new Date(tr.timestamp).toLocaleTimeString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {totalPages > 1 && (
        <div className="flex items-center justify-end gap-2 mt-3 text-xs text-muted">
          <button disabled={page === 0} onClick={() => setPage(page - 1)} className="px-2 py-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30">←</button>
          <span>{page + 1} / {totalPages}</span>
          <button disabled={page >= totalPages - 1} onClick={() => setPage(page + 1)} className="px-2 py-1 rounded hover:bg-[var(--hover-bg)] disabled:opacity-30">→</button>
        </div>
      )}
    </div>
  );
}
