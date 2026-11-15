"use client";

import { ArrowUpRight } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { incidentSubject } from "@/lib/entity-key";
import type { IncidentLinkDetail } from "@/types/incident-grouping";
import { IncidentStateBadge } from "./IncidentBadges";
import { DependencyPath, LinkedSubject } from "./IncidentLinkParts";
import { formatShortTime } from "./incident-format";
import { linkSubjectSource } from "./incident-grouping-format";
import { fillTemplate } from "./incident-view-format";

interface IncidentRootLinkProps {
  root: IncidentLinkDetail;
  /** 切到源头事件的详情 */
  onOpenIncident: (incidentId: string) => void;
}

/** 受影响事件：结论块下方一行「源头：<主体> · 经 <路径>」，点主体切到源头事件的详情 */
export function IncidentRootLink({ root, onOpenIncident }: IncidentRootLinkProps) {
  const { t } = useI18n();
  const texts = t.aiops.grouping;

  return (
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-xl border border-dashed border-[var(--border-color)] px-3 py-2 text-xs">
      <span className="text-muted flex-shrink-0">{texts.rootLabel}</span>
      <button
        type="button"
        onClick={() => onOpenIncident(root.incidentId)}
        title={texts.rootLinkHint}
        className="inline-flex items-center gap-1.5 min-w-0 rounded-lg px-1.5 py-0.5 text-sm text-default hover:bg-[var(--hover-bg)] transition-colors"
      >
        <LinkedSubject subject={incidentSubject(linkSubjectSource(root))} />
        <IncidentStateBadge state={root.state} />
        <ArrowUpRight className="w-3.5 h-3.5 flex-shrink-0 text-blue-600 dark:text-blue-400" />
      </button>
      <span className="text-muted flex-shrink-0">· {texts.via}</span>
      <DependencyPath path={root.path} />
      {root.linkedAt !== "" && (
        <span className="text-muted tabular-nums whitespace-nowrap">
          · {fillTemplate(texts.linkedAt, { time: formatShortTime(root.linkedAt) })}
        </span>
      )}
    </div>
  );
}
