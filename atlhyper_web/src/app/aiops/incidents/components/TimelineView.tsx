"use client";

import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { IncidentTimelineItem } from "@/types/incident-view";
import { formatShortTime } from "./incident-format";
import { timelineLine, timelineSummary } from "./incident-timeline-format";
import { fillTemplate } from "./incident-view-format";

interface TimelineViewProps {
  items: IncidentTimelineItem[];
}

/** 时间线：默认折叠，摘要写「N 个变化，合并了 M 条检测」；展开后只有状态与严重度的变化，检测记录已合并成一行 */
export function TimelineView({ items }: TimelineViewProps) {
  const { t } = useI18n();
  const texts = t.aiops.detail;
  const severityLabels: Record<string, string> = t.aiops.severityLevel;
  const [expanded, setExpanded] = useState(false);
  const { changes, merged } = timelineSummary(items);

  return (
    <div className="rounded-xl border border-[var(--border-color)]">
      <button
        type="button"
        onClick={() => setExpanded((open) => !open)}
        aria-expanded={expanded}
        className="w-full flex items-center justify-between gap-3 px-3 py-2.5 text-sm text-muted hover:text-default transition-colors"
      >
        <span className="text-left">
          <span className="font-semibold text-default">{texts.timelineTitle}</span>
          {items.length > 0 && <> · {fillTemplate(texts.timelineSummary, { changes, merged })}</>}
        </span>
        <span className="inline-flex items-center gap-1 text-xs flex-shrink-0">
          {expanded ? texts.timelineCollapse : texts.timelineExpand}
          <ChevronDown className={`w-3.5 h-3.5 transition-transform ${expanded ? "rotate-180" : ""}`} />
        </span>
      </button>

      {expanded && (
        <div className="px-3 pb-3">
          {items.length === 0 ? (
            <p className="text-xs text-muted py-1">{texts.timelineEmpty}</p>
          ) : (
            <ol className="space-y-2">
              {items.map((item, index) => {
                const line = timelineLine(item, texts, severityLabels);
                return (
                  <li key={`${item.kind}-${item.at}-${index}`} className="flex gap-3 text-sm">
                    <time className="w-20 flex-shrink-0 text-xs text-muted font-mono pt-0.5">
                      {formatShortTime(item.at)}
                    </time>
                    <div className={`min-w-0 ${line.merged ? "text-muted text-xs pt-0.5" : "text-default"}`}>
                      <span className={line.merged ? "" : "font-medium"}>{line.label}</span>
                      {line.notes.length > 0 && (
                        <span className={line.merged ? "" : "text-xs text-muted"}>
                          {line.merged ? " " : " · "}
                          {line.notes.join(" · ")}
                        </span>
                      )}
                    </div>
                  </li>
                );
              })}
            </ol>
          )}
        </div>
      )}
    </div>
  );
}
