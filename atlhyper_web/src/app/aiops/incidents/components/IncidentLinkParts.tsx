"use client";

import { useState } from "react";
import { useI18n } from "@/i18n/context";
import type { IncidentSubject } from "@/lib/entity-key";
import type { IncidentPathStep } from "@/types/incident-grouping";
import { gapText, isPathCollapsible, pathSegments, pathText, subjectShortName } from "./incident-grouping-format";

/**
 * 关联另一端事件的主体：类型 + 命名空间 / 名称，纯文字。
 *
 * 与 SubjectTitle 不同，名称不是资源链接：它所在的按钮点了是切到那个事件的详情，
 * 里面再放一个「打开资源详情」的链接会让点击目标含混。
 */
export function LinkedSubject({ subject }: { subject: IncidentSubject }) {
  const { t } = useI18n();
  const typeLabels: Record<string, string> = t.aiops.subjectTypeLabel;
  return (
    <span className="inline-flex items-center gap-1.5 min-w-0 max-w-full" title={subject.key}>
      <span className="flex-shrink-0 rounded font-medium bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 text-[10px] px-1.5 py-0.5">
        {typeLabels[subject.type] ?? subject.type}
      </span>
      <span className="truncate font-medium">{subjectShortName(subject)}</span>
    </span>
  );
}

/**
 * 依赖路径（归组依据）：实体名（类型）→ 边 → 实体名（类型）…
 * 实体多于 3 个时默认折叠成「起点 → …（N 个）… → 边 → 终点」，可展开；悬停显示完整路径。
 */
export function DependencyPath({ path }: { path: IncidentPathStep[] }) {
  const { t } = useI18n();
  const texts = t.aiops.grouping;
  const typeLabels: Record<string, string> = t.aiops.subjectTypeLabel;
  const [collapsed, setCollapsed] = useState(true);

  if (path.length === 0) return <span className="text-xs text-muted">{texts.pathEmpty}</span>;

  const collapsible = isPathCollapsible(path);
  const segments = pathSegments(path, texts, typeLabels, collapsed);
  const fullText = pathText(pathSegments(path, texts, typeLabels, false), texts);

  return (
    <span className="inline-flex flex-wrap items-center gap-x-1 gap-y-0.5 text-xs min-w-0" title={fullText}>
      {segments.map((segment, index) => (
        <span key={index} className="inline-flex items-center gap-1">
          {index > 0 && <span className="text-muted">→</span>}
          {segment.kind === "entity" && (
            <span className="text-default">
              <span className="font-mono">{segment.name}</span>
              <span className="text-muted"> ({segment.typeLabel})</span>
            </span>
          )}
          {segment.kind === "edge" && (
            <span className="px-1 rounded bg-[var(--hover-bg)] text-muted whitespace-nowrap">{segment.label}</span>
          )}
          {segment.kind === "gap" && <span className="text-muted whitespace-nowrap">{gapText(segment.hidden, texts)}</span>}
        </span>
      ))}
      {collapsible && (
        <button
          type="button"
          onClick={() => setCollapsed((value) => !value)}
          className="ml-1 text-blue-600 dark:text-blue-400 hover:underline whitespace-nowrap"
        >
          {collapsed ? texts.pathExpand : texts.pathCollapse}
        </button>
      )}
    </span>
  );
}
