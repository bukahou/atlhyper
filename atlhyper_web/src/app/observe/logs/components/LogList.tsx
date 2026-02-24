"use client";

import React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { LogEntry } from "@/types/model/log";
import type { LogTranslations } from "@/types/i18n";
import { severityColor, shortScopeName } from "@/types/model/log";

interface LogListProps {
  logs: LogEntry[];
  total: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onSelectEntry?: (entry: LogEntry, idx: number) => void;
  selectedIdx?: number | null;
  searchHighlight?: string;
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

/** Level → left bar color */
function severityBarColor(severity: string): string {
  switch (severity.toUpperCase()) {
    case "ERROR": return "#ef4444";
    case "WARN": return "#f59e0b";
    case "INFO": return "#3b82f6";
    case "DEBUG":
    default: return "#6b7280";
  }
}

/** Highlight search keyword in text */
function highlightText(text: string, keyword?: string): React.ReactNode {
  if (!keyword) return text;
  const lower = text.toLowerCase();
  const kw = keyword.toLowerCase();
  const idx = lower.indexOf(kw);
  if (idx < 0) return text;
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-yellow-300/70 dark:bg-yellow-500/40 text-inherit rounded-sm px-0.5">{text.slice(idx, idx + keyword.length)}</mark>
      {text.slice(idx + keyword.length)}
    </>
  );
}

export function LogList({ logs, total, page, pageSize, onPageChange, onSelectEntry, selectedIdx, searchHighlight, t }: LogListProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const hasPrev = page > 1;
  const hasNext = page < totalPages;

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
          const isSelected = selectedIdx === idx;
          return (
            <div key={idx} className="flex">
              {/* Severity color bar */}
              <div
                className="w-1 flex-shrink-0"
                style={{ backgroundColor: severityBarColor(entry.severity) }}
              />

              <button
                onClick={() => onSelectEntry?.(entry, idx)}
                className={`flex-1 min-w-0 flex items-center gap-2 px-3 py-2 text-left hover:bg-[var(--hover-bg)] transition-colors text-xs ${
                  isSelected ? "bg-primary/5 border-l-2 border-l-primary" : ""
                }`}
              >
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

                {/* Body preview with search highlight */}
                <span className="text-default truncate flex-1 min-w-0">
                  {highlightText(entry.body, searchHighlight)}
                </span>
              </button>
            </div>
          );
        })}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3 mt-3">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={!hasPrev}
            className="p-1.5 rounded-md border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ChevronLeft className="w-4 h-4 text-default" />
          </button>
          <span className="text-xs text-muted tabular-nums">
            {page} / {totalPages}
          </span>
          <button
            onClick={() => onPageChange(page + 1)}
            disabled={!hasNext}
            className="p-1.5 rounded-md border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ChevronRight className="w-4 h-4 text-default" />
          </button>
        </div>
      )}
    </div>
  );
}
