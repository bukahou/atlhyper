"use client";

import { CornerDownRight } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { incidentSubject } from "@/lib/entity-key";
import type { IncidentLinkSummary } from "@/types/incident-grouping";
import { BADGE_CLASS, IncidentSeverityBadge, IncidentStateBadge } from "./IncidentBadges";
import { LinkedSubject } from "./IncidentLinkParts";
import { GRID_COLUMNS } from "./IncidentListRow";
import { linkSubjectSource } from "./incident-grouping-format";

interface IncidentImpactedRowsProps {
  impacted: IncidentLinkSummary[];
  onSelect: (id: string) => void;
}

/**
 * 按组折叠时展开的受影响事件：折叠在源头行下面，每行可点开详情。
 * 后端只给了概要（事件号、主体、状态、严重度），其余列留空，详情里看。
 */
export function IncidentImpactedRows({ impacted, onSelect }: IncidentImpactedRowsProps) {
  const { t } = useI18n();
  const texts = t.aiops.grouping;

  return (
    <div className="bg-[var(--background)]/60 border-b border-[var(--border-color)]/20">
      {impacted.map((item) => (
        <button
          key={item.incidentId}
          onClick={() => onSelect(item.incidentId)}
          className={`w-full ${GRID_COLUMNS} px-5 py-2 text-sm text-left hover:bg-[var(--hover-bg)] transition-colors items-center`}
        >
          <span className="inline-flex items-center gap-1 font-mono text-xs text-muted">
            <CornerDownRight className="w-3 h-3 flex-shrink-0" />
            {item.incidentId}
          </span>
          <span className="flex flex-col items-start">
            <IncidentStateBadge state={item.state} />
          </span>
          <IncidentSeverityBadge severity={item.severity} />
          <span className="col-span-6 flex items-center gap-2 min-w-0">
            <span className={`${BADGE_CLASS} flex-shrink-0 border border-[var(--border-color)] text-muted`}>
              {texts.impactedBadge}
            </span>
            <LinkedSubject subject={incidentSubject(linkSubjectSource(item))} />
          </span>
        </button>
      ))}
    </div>
  );
}
