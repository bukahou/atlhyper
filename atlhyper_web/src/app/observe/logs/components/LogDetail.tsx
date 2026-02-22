"use client";

import Link from "next/link";
import { ExternalLink, X } from "lucide-react";
import type { LogEntry } from "@/types/model/log";
import type { LogTranslations } from "@/types/i18n";
import { hasTrace, shortScopeName, severityColor } from "@/types/model/log";

interface LogDetailDrawerProps {
  entry: LogEntry | null;
  onClose: () => void;
  t: LogTranslations;
}

export function LogDetailDrawer({ entry, onClose, t }: LogDetailDrawerProps) {
  if (!entry) return null;

  const attrs = Object.entries(entry.attributes);
  const res = Object.entries(entry.resource);

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/20 dark:bg-black/40"
        onClick={onClose}
      />

      {/* Drawer */}
      <div className="fixed top-0 right-0 z-50 h-full w-[480px] max-w-[90vw] bg-card border-l border-[var(--border-color)] shadow-2xl flex flex-col animate-slide-in-right">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${severityColor(entry.severity)}`}>
              {entry.severity}
            </span>
            <span className="text-sm font-medium text-default truncate">
              {entry.serviceName}
            </span>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-[var(--hover-bg)] transition-colors"
          >
            <X className="w-4 h-4 text-muted" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
          {/* Body 全文 */}
          <div className="p-3 rounded-lg bg-[var(--background)] border border-[var(--border-color)]">
            <p className="font-mono text-xs whitespace-pre-wrap break-all text-default leading-relaxed">
              {entry.body}
            </p>
          </div>

          {/* 元信息 */}
          <div className="grid grid-cols-2 gap-x-6 gap-y-3 text-xs">
            <div>
              <span className="text-muted">{t.timestamp}</span>
              <p className="text-default font-mono mt-0.5">{entry.timestamp}</p>
            </div>
            <div>
              <span className="text-muted">{t.service}</span>
              <p className="text-default mt-0.5">{entry.serviceName}</p>
            </div>
            <div>
              <span className="text-muted">{t.severity}</span>
              <p className="text-default mt-0.5">{entry.severity} ({entry.severityNum})</p>
            </div>
            <div>
              <span className="text-muted">{t.scopeName}</span>
              <p className="text-default mt-0.5 truncate" title={entry.scopeName}>
                {shortScopeName(entry.scopeName)}
              </p>
            </div>

            {/* TraceId */}
            {hasTrace(entry) && (
              <div className="col-span-2">
                <span className="text-muted">{t.traceId}</span>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <p className="text-default font-mono truncate">{entry.traceId}</p>
                  <Link
                    href={`/observe/apm?trace=${entry.traceId}`}
                    className="inline-flex items-center gap-0.5 text-primary hover:text-primary/80 whitespace-nowrap"
                  >
                    {t.viewTrace}
                    <ExternalLink className="w-3 h-3" />
                  </Link>
                </div>
              </div>
            )}

            {/* SpanId */}
            {entry.spanId && (
              <div className="col-span-2">
                <span className="text-muted">{t.spanId}</span>
                <p className="text-default font-mono mt-0.5">{entry.spanId}</p>
              </div>
            )}
          </div>

          {/* Attributes */}
          {attrs.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-muted mb-1.5">{t.attributes}</h4>
              <div className="rounded-lg border border-[var(--border-color)] overflow-hidden">
                <table className="w-full text-xs">
                  <tbody>
                    {attrs.map(([key, value]) => (
                      <tr key={key} className="border-b border-[var(--border-color)] last:border-b-0">
                        <td className="px-3 py-1.5 font-mono text-muted bg-[var(--background)] w-2/5">{key}</td>
                        <td className="px-3 py-1.5 font-mono text-default break-all">{value}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Resource */}
          {res.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-muted mb-1.5">{t.resourceInfo}</h4>
              <div className="rounded-lg border border-[var(--border-color)] overflow-hidden">
                <table className="w-full text-xs">
                  <tbody>
                    {res.map(([key, value]) => (
                      <tr key={key} className="border-b border-[var(--border-color)] last:border-b-0">
                        <td className="px-3 py-1.5 font-mono text-muted bg-[var(--background)] w-2/5">{key}</td>
                        <td className="px-3 py-1.5 font-mono text-default break-all">{value}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      </div>
    </>
  );
}
