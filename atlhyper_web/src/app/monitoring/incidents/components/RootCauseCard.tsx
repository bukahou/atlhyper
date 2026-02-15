"use client";

import { Target } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { EntityLink } from "@/components/aiops/EntityLink";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import type { IncidentEntity } from "@/api/aiops";

interface RootCauseCardProps {
  entity: IncidentEntity | undefined;
}

export function RootCauseCard({ entity }: RootCauseCardProps) {
  const { t } = useI18n();

  if (!entity) return null;

  return (
    <div className="bg-red-500/5 border border-red-500/20 rounded-xl p-4">
      <div className="flex items-center gap-2 mb-3">
        <Target className="w-4 h-4 text-red-500" />
        <span className="text-sm font-semibold text-default">{t.aiops.rootCause}</span>
      </div>
      <div className="flex items-center gap-4">
        <EntityLink entityKey={entity.entityKey} />
        <div className="flex items-center gap-2 text-xs text-muted">
          <span>{t.aiops.rFinal}:</span>
          <span className="font-mono font-semibold text-default">{entity.rFinal.toFixed(1)}</span>
        </div>
        <RiskBadge level={entity.rFinal >= 80 ? "critical" : entity.rFinal >= 50 ? "warning" : "low"} size="md" />
      </div>
    </div>
  );
}
