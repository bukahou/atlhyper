"use client";

import { EntityLink } from "@/components/aiops/EntityLink";
import { useI18n } from "@/i18n/context";
import type { IncidentEntity } from "@/api/aiops";
import { formatRiskScore } from "@/lib/risk";

const ROLE_COLORS: Record<string, string> = {
  root_cause: "bg-red-500/10 text-red-600 dark:text-red-400",
  affected: "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400",
};

interface IncidentMembersProps {
  /** 事件的全部实体：领头实体（root_cause）与主体的其余成员（affected） */
  entities: IncidentEntity[];
}

/** 成员实体：这个主体下涉及哪几个实体。领头实体排在最前 */
export function IncidentMembers({ entities }: IncidentMembersProps) {
  const { t } = useI18n();
  const roleLabels: Record<string, string> = t.aiops.entityRole;
  const leadFirst = [
    ...entities.filter((entity) => entity.role === "root_cause"),
    ...entities.filter((entity) => entity.role !== "root_cause"),
  ];

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
        {t.aiops.memberEntities}
        <span className="ml-1.5 font-normal tabular-nums">({entities.length})</span>
      </h4>
      {leadFirst.length === 0 ? (
        <p className="text-xs text-muted">{t.aiops.noMemberEntities}</p>
      ) : (
        <div className="space-y-1.5">
          {leadFirst.map((entity) => (
            <div key={entity.entityKey} className="flex items-center gap-3 text-sm">
              <span
                className={`flex-shrink-0 text-[10px] px-1.5 py-0.5 rounded font-medium ${
                  ROLE_COLORS[entity.role] ?? "bg-gray-500/10 text-muted"
                }`}
              >
                {roleLabels[entity.role] ?? entity.role}
              </span>
              <EntityLink entityKey={entity.entityKey} />
              <span className="text-xs text-muted tabular-nums">R={formatRiskScore(entity.rFinal)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
