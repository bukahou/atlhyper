"use client";

import { ArrowUpRight } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { incidentSubject } from "@/lib/entity-key";
import type { IncidentLinkDetail } from "@/types/incident-grouping";
import { IncidentSeverityBadge, IncidentStateBadge } from "./IncidentBadges";
import { DependencyPath, LinkedSubject } from "./IncidentLinkParts";
import { linkSubjectSource } from "./incident-grouping-format";
import { fillTemplate } from "./incident-view-format";

interface IncidentImpactScopeProps {
  impacted: IncidentLinkDetail[];
  /** 切到某个受影响事件的详情 */
  onOpenIncident: (incidentId: string) => void;
}

/**
 * 源头事件的「影响范围」：归到它下面的受影响事件——主体、状态、严重度、依赖路径（归组依据）。
 * 点主体切到那个事件的详情。没有受影响事件时由调用方不渲染（这是条件信息，不是独立的数据视图）。
 */
export function IncidentImpactScope({ impacted, onOpenIncident }: IncidentImpactScopeProps) {
  const { t } = useI18n();
  const texts = t.aiops.grouping;

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted tracking-wider mb-1">
        {fillTemplate(texts.impactTitle, { count: impacted.length })}
      </h4>
      <p className="text-xs text-muted mb-2">{texts.impactHint}</p>
      <div className="space-y-1.5">
        {impacted.map((item) => (
          <div key={item.incidentId} className="rounded-xl border border-[var(--border-color)] px-3 py-2 space-y-1">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <button
                type="button"
                onClick={() => onOpenIncident(item.incidentId)}
                title={texts.impactOpenHint}
                className="inline-flex items-center gap-1.5 min-w-0 rounded-lg -mx-1.5 px-1.5 py-0.5 text-sm text-default hover:bg-[var(--hover-bg)] transition-colors"
              >
                <LinkedSubject subject={incidentSubject(linkSubjectSource(item))} />
                <ArrowUpRight className="w-3.5 h-3.5 flex-shrink-0 text-blue-600 dark:text-blue-400" />
              </button>
              <IncidentStateBadge state={item.state} />
              <IncidentSeverityBadge severity={item.severity} />
              <span className="font-mono text-[10px] text-muted">{item.incidentId}</span>
            </div>
            <DependencyPath path={item.path} />
          </div>
        ))}
      </div>
    </div>
  );
}
