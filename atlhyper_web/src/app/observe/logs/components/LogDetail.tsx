"use client";

import Link from "next/link";
import { ExternalLink } from "lucide-react";
import type { LogEntry } from "@/types/model/log";
import type { LogTranslations } from "@/types/i18n";
import { hasTrace, shortScopeName } from "@/types/model/log";

interface LogDetailProps {
  entry: LogEntry;
  t: LogTranslations;
}

export function LogDetail({ entry, t }: LogDetailProps) {
  const attrs = Object.entries(entry.attributes);
  const res = Object.entries(entry.resource);

  return (
    <div className="px-4 py-3 border-t border-[var(--border-color)] bg-[var(--hover-bg)]">
      {/* Body 全文 */}
      <div className="mb-3 p-3 rounded-lg bg-[var(--background)] border border-[var(--border-color)]">
        <p className="font-mono text-xs whitespace-pre-wrap break-all text-default">
          {entry.body}
        </p>
      </div>

      {/* 元信息网格 */}
      <div className="grid grid-cols-2 md:grid-cols-3 gap-x-6 gap-y-2 text-xs mb-3">
        <div>
          <span className="text-muted">{t.timestamp}</span>
          <p className="text-default font-mono">{entry.timestamp}</p>
        </div>
        <div>
          <span className="text-muted">{t.service}</span>
          <p className="text-default">{entry.serviceName}</p>
        </div>
        <div>
          <span className="text-muted">{t.severity}</span>
          <p className="text-default">{entry.severity} ({entry.severityNum})</p>
        </div>
        <div>
          <span className="text-muted">{t.scopeName}</span>
          <p className="text-default" title={entry.scopeName}>
            {shortScopeName(entry.scopeName)}
            <span className="text-muted ml-1 hidden md:inline">({entry.scopeName})</span>
          </p>
        </div>

        {/* TraceId */}
        {hasTrace(entry) && (
          <div>
            <span className="text-muted">{t.traceId}</span>
            <div className="flex items-center gap-1.5">
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
          <div>
            <span className="text-muted">{t.spanId}</span>
            <p className="text-default font-mono">{entry.spanId}</p>
          </div>
        )}
      </div>

      {/* Attributes */}
      {attrs.length > 0 && (
        <div className="mb-3">
          <h4 className="text-xs font-medium text-muted mb-1">{t.attributes}</h4>
          <div className="rounded-lg border border-[var(--border-color)] overflow-hidden">
            <table className="w-full text-xs">
              <tbody>
                {attrs.map(([key, value]) => (
                  <tr key={key} className="border-b border-[var(--border-color)] last:border-b-0">
                    <td className="px-3 py-1.5 font-mono text-muted bg-[var(--background)] w-1/3">{key}</td>
                    <td className="px-3 py-1.5 font-mono text-default">{value}</td>
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
          <h4 className="text-xs font-medium text-muted mb-1">{t.resourceInfo}</h4>
          <div className="rounded-lg border border-[var(--border-color)] overflow-hidden">
            <table className="w-full text-xs">
              <tbody>
                {res.map(([key, value]) => (
                  <tr key={key} className="border-b border-[var(--border-color)] last:border-b-0">
                    <td className="px-3 py-1.5 font-mono text-muted bg-[var(--background)] w-1/3">{key}</td>
                    <td className="px-3 py-1.5 font-mono text-default">{value}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
