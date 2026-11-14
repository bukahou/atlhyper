"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import type { EntityRisk } from "@/api/aiops";
import { EntityRiskExpansion } from "./EntityRiskExpansion";

const LIMIT_OPTIONS = [20, 50, 100] as const;

interface TopEntitiesProps {
  entities: EntityRisk[];
  clusterId: string;
  limit: number;
  onLimitChange: (limit: number) => void;
}

function formatTimeAgo(ts: number, t: { minutesAgo: string; hoursAgo: string; justNow: string; noAnomaly: string }): string {
  if (!ts) return t.noAnomaly;
  const diffMs = Date.now() - ts * 1000; // ts 是 Unix 秒，转换为毫秒
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return t.justNow;
  if (diffMin < 60) return `${diffMin} ${t.minutesAgo}`;
  return `${Math.floor(diffMin / 60)} ${t.hoursAgo}`;
}

export function TopEntities({ entities, clusterId, limit, onLimitChange }: TopEntitiesProps) {
  const { t } = useI18n();
  const [expandedKey, setExpandedKey] = useState<string | null>(null);

  // 展开的那一行自己取详情与基线（见 EntityRiskExpansion）；这里只记展开的是哪一行
  const handleToggle = (key: string) => setExpandedKey((current) => (current === key ? null : key));

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-5 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <h3 className="text-sm font-semibold text-default">{t.aiops.topRiskEntities}</h3>
        <div className="flex items-center gap-1">
          {LIMIT_OPTIONS.map((n) => (
            <button
              key={n}
              onClick={() => onLimitChange(n)}
              className={`px-2 py-0.5 text-xs rounded transition-colors ${
                limit === n
                  ? "bg-blue-500/15 text-blue-500 font-medium"
                  : "text-muted hover:text-default"
              }`}
            >
              {n}
            </button>
          ))}
        </div>
      </div>

      {/* 表头 */}
      <div className="grid grid-cols-[2fr_80px_80px_80px_80px_100px] gap-2 px-5 py-2 text-[10px] text-muted uppercase tracking-wider border-b border-[var(--border-color)]/50">
        <span>{t.aiops.entityKey}</span>
        <span>{t.aiops.entityType}</span>
        <span>{t.aiops.rLocal}</span>
        <span>{t.aiops.rFinal}</span>
        <span>{t.aiops.riskLevelLabel}</span>
        <span>{t.aiops.firstAnomaly}</span>
      </div>

      {/* 行 */}
      {entities.length === 0 ? (
        <div className="px-5 py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
      ) : (
        entities.map((entity) => {
          const isExpanded = expandedKey === entity.entityKey;

          return (
            <div key={entity.entityKey}>
              <button
                onClick={() => handleToggle(entity.entityKey)}
                className="w-full grid grid-cols-[2fr_80px_80px_80px_80px_100px] gap-2 px-5 py-2.5 text-sm hover:bg-[var(--hover-bg)] transition-colors items-center"
              >
                <div className="flex items-center gap-2 min-w-0">
                  {isExpanded ? (
                    <ChevronDown className="w-3.5 h-3.5 text-muted flex-shrink-0" />
                  ) : (
                    <ChevronRight className="w-3.5 h-3.5 text-muted flex-shrink-0" />
                  )}
                  <EntityLink entityKey={entity.entityKey} />
                </div>
                <span className="text-xs text-muted">{entity.entityType}</span>
                <span className="text-xs font-mono text-default">{Math.round(entity.rLocal)}</span>
                <span className="text-xs font-mono font-semibold text-default">{Math.round(entity.rFinal)}</span>
                <RiskBadge level={entity.riskLevel} />
                <span className="text-xs text-muted">{formatTimeAgo(entity.firstAnomaly, t.aiops)}</span>
              </button>

              {/* 展开详情 */}
              {isExpanded && <EntityRiskExpansion clusterId={clusterId} entityKey={entity.entityKey} />}
            </div>
          );
        })
      )}
    </div>
  );
}
