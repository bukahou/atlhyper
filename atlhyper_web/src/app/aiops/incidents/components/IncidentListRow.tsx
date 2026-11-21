"use client";

import type { KeyboardEvent, MouseEvent } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import { BADGE_CLASS, IncidentSeverityBadge, IncidentStateBadge } from "./IncidentBadges";
import { formatDuration, formatShortTime } from "./incident-format";
import { rootTagLabel, type IncidentListRowView } from "./incident-grouping-format";
import { fillTemplate } from "./incident-view-format";
import { SubjectTitle } from "./SubjectTitle";

/** ID | 状态 | 严重度 | 主体（+ 风险最高的成员） | 峰值 | 开始 | 持续 | 复发 | 升级时间 */
export const GRID_COLUMNS = "grid grid-cols-[72px_88px_64px_minmax(0,1fr)_64px_92px_76px_52px_92px] gap-2";

/**
 * 行本身是按钮（点开详情），行里的小标签也要能点：用 role="button" 的 span（同 EntityLink），
 * 阻止冒泡，免得同时打开这一行的详情
 */
function inlineAction(action: () => void) {
  return {
    role: "button" as const,
    tabIndex: 0,
    onClick: (event: MouseEvent) => {
      event.stopPropagation();
      action();
    },
    onKeyDown: (event: KeyboardEvent) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        event.stopPropagation();
        action();
      }
    },
  };
}

interface IncidentListRowProps {
  row: IncidentListRowView;
  onSelect: (id: string) => void;
  /** 按组折叠时源头的 impacted 可展开；平铺时「关联 N 个」只是计数 */
  expandable: boolean;
  expanded: boolean;
  onToggleExpanded: (id: string) => void;
}

/** 事件列表的一行：源头带「关联 N 个」，平铺时受影响事件带「源头：<主体>」小标签 */
export function IncidentListRow({ row, onSelect, expandable, expanded, onToggleExpanded }: IncidentListRowProps) {
  const { t } = useI18n();
  const texts = t.aiops.grouping;
  const typeLabels: Record<string, string> = t.aiops.subjectTypeLabel;
  const { incident: inc, rootTag } = row;

  return (
    <button
      onClick={() => onSelect(inc.id)}
      className={`w-full ${GRID_COLUMNS} px-5 py-2.5 text-sm text-left hover:bg-[var(--hover-bg)] transition-colors items-center border-b border-[var(--border-color)]/20 last:border-b-0`}
    >
      <span className="font-mono text-xs text-default">{inc.id}</span>
      <span className="flex flex-col items-start gap-1">
        <IncidentStateBadge state={inc.state} />
        {/* 失联关闭不是真的恢复了，要一眼看得出来 */}
        {inc.closeReason === "stale" && (
          <span
            title={t.aiops.closeReasonStaleHint}
            className={`${BADGE_CLASS} bg-orange-500/15 text-orange-600 dark:text-orange-400`}
          >
            {t.aiops.closeReasonLabel.stale}
          </span>
        )}
      </span>
      <IncidentSeverityBadge severity={inc.severity} />
      <div className="min-w-0 space-y-0.5">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 min-w-0">
          <SubjectTitle incident={inc} />
          {/* 源头：关联 N 个（按组折叠时可展开） */}
          {inc.linkedCount > 0 &&
            (expandable ? (
              <span
                {...inlineAction(() => onToggleExpanded(inc.id))}
                aria-expanded={expanded}
                title={texts.linkedCountHint}
                className={`${BADGE_CLASS} gap-0.5 cursor-pointer bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-500/20`}
              >
                {expanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                {fillTemplate(texts.linkedCount, { count: inc.linkedCount })}
              </span>
            ) : (
              <span title={texts.linkedCountFlatHint} className={`${BADGE_CLASS} bg-indigo-500/10 text-indigo-600 dark:text-indigo-400`}>
                {fillTemplate(texts.linkedCount, { count: inc.linkedCount })}
              </span>
            ))}
          {/* 平铺时的受影响事件：源头小标签，点击打开源头详情 */}
          {rootTag && (
            <span
              {...inlineAction(() => onSelect(rootTag.incidentId))}
              title={texts.rootTagHint}
              className={`${BADGE_CLASS} cursor-pointer border border-dashed border-[var(--border-color)] text-muted hover:text-default max-w-full truncate`}
            >
              {rootTagLabel(rootTag, texts, typeLabels)}
            </span>
          )}
        </div>
        {/* 领头实体：主体的成员里风险最高的那个 */}
        <div className="flex items-center gap-1.5 text-xs text-muted min-w-0">
          <span className="flex-shrink-0">{t.aiops.leadMember}:</span>
          <EntityLink entityKey={inc.rootCause} />
        </div>
      </div>
      <span className="justify-self-start">
        <RiskBadge level={inc.peakRiskLevel} />
      </span>
      <span className="text-xs text-muted">{formatShortTime(inc.startedAt)}</span>
      <span className="text-xs text-muted">{formatDuration(inc.durationS, t.aiops.minutes)}</span>
      <span className="text-xs text-muted text-center tabular-nums">{inc.recurrence}</span>
      <span className="text-xs text-muted">
        {inc.escalatedAt ? formatShortTime(inc.escalatedAt) : t.aiops.notEscalated}
      </span>
    </button>
  );
}
