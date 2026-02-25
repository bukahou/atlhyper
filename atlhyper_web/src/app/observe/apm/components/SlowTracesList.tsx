"use client";

import type { TraceSummary } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";

interface SlowTracesListProps {
  t: ApmTranslations;
  traces: TraceSummary[];
  onSelectTrace?: (traceId: string) => void;
}

export function SlowTracesList({ t, traces, onSelectTrace }: SlowTracesListProps) {
  if (traces.length === 0) {
    return (
      <div>
        <h3 className="text-sm font-medium text-default mb-3">{t.slowTraces}</h3>
        <div className="py-8 text-center text-sm text-muted">{t.noTraces}</div>
      </div>
    );
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-default mb-3">{t.slowTraces}</h3>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-muted text-xs border-b border-[var(--border-color)]">
              <th className="pb-2 pr-4">#</th>
              <th className="pb-2 pr-4">{t.traceId}</th>
              <th className="pb-2 pr-4">{t.rootOperation}</th>
              <th className="pb-2 pr-4 text-right">{t.duration}</th>
              <th className="pb-2 pr-4 text-right">{t.spans}</th>
              <th className="pb-2 text-right">{t.startTime}</th>
            </tr>
          </thead>
          <tbody>
            {traces.map((tr, i) => (
              <tr
                key={tr.traceId}
                onClick={() => onSelectTrace?.(tr.traceId)}
                className="border-b border-[var(--border-color)] last:border-0 hover:bg-[var(--hover-bg)] cursor-pointer transition-colors"
              >
                <td className="py-2 pr-4 text-muted text-xs">{i + 1}</td>
                <td className="py-2 pr-4 font-mono text-xs text-primary">{tr.traceId.slice(0, 16)}...</td>
                <td className="py-2 pr-4 text-default truncate max-w-[240px]">{tr.rootOperation}</td>
                <td className="py-2 pr-4 text-right">
                  <span className={tr.hasError ? "text-red-500 font-medium" : "text-default"}>
                    {formatDurationMs(tr.durationMs)}
                  </span>
                </td>
                <td className="py-2 pr-4 text-right text-muted">{tr.spanCount}</td>
                <td className="py-2 text-right text-muted text-xs">{new Date(tr.timestamp).toLocaleTimeString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
