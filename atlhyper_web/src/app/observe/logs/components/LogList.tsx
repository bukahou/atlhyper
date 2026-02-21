"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { LogEntry } from "@/types/model/log";
import type { LogTranslations } from "@/types/i18n";
import { severityColor, shortScopeName } from "@/types/model/log";
import { LogDetail } from "./LogDetail";

interface LogListProps {
  logs: LogEntry[];
  total: number;
  displayCount: number;
  onLoadMore: () => void;
  t: LogTranslations;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  const ss = String(d.getSeconds()).padStart(2, "0");
  const ms = String(d.getMilliseconds()).padStart(3, "0");
  return `${hh}:${mm}:${ss}.${ms}`;
}

export function LogList({ logs, total, displayCount, onLoadMore, t }: LogListProps) {
  const [expandedIdx, setExpandedIdx] = useState<number | null>(null);

  const toggleExpand = (idx: number) => {
    setExpandedIdx(expandedIdx === idx ? null : idx);
  };

  if (logs.length === 0) {
    return (
      <div className="flex items-center justify-center h-48 text-muted text-sm">
        {t.noLogs}
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <div className="divide-y divide-[var(--border-color)] border border-[var(--border-color)] rounded-lg overflow-hidden">
        {logs.map((entry, idx) => {
          const isExpanded = expandedIdx === idx;
          return (
            <div key={idx}>
              <button
                onClick={() => toggleExpand(idx)}
                className={`w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-[var(--hover-bg)] transition-colors text-xs ${
                  isExpanded ? "bg-[var(--hover-bg)]" : ""
                }`}
              >
                {isExpanded
                  ? <ChevronDown className="w-3.5 h-3.5 text-muted flex-shrink-0" />
                  : <ChevronRight className="w-3.5 h-3.5 text-muted flex-shrink-0" />
                }

                {/* Time */}
                <span className="font-mono text-muted w-[90px] flex-shrink-0">
                  {formatTime(entry.timestamp)}
                </span>

                {/* Service badge */}
                <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-purple-500/10 text-purple-500 w-[110px] flex-shrink-0 truncate text-center">
                  {entry.serviceName.replace("geass-", "")}
                </span>

                {/* Severity badge */}
                <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium w-[50px] flex-shrink-0 text-center ${severityColor(entry.severity)}`}>
                  {entry.severity}
                </span>

                {/* Scope (short) */}
                <span className="text-muted w-[100px] flex-shrink-0 truncate hidden lg:inline-block" title={entry.scopeName}>
                  {shortScopeName(entry.scopeName)}
                </span>

                {/* Body preview */}
                <span className="text-default truncate flex-1 min-w-0">
                  {entry.body}
                </span>
              </button>

              {/* Expanded detail */}
              {isExpanded && <LogDetail entry={entry} t={t} />}
            </div>
          );
        })}
      </div>

      {/* Load more */}
      {displayCount < total && (
        <div className="flex justify-center mt-4">
          <button
            onClick={onLoadMore}
            className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] text-default transition-colors"
          >
            {t.showMore}
          </button>
        </div>
      )}
    </div>
  );
}
