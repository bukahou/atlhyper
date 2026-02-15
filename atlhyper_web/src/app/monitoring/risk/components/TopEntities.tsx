"use client";

import { useState, useCallback } from "react";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import { getEntityRiskDetail } from "@/api/aiops";
import type { EntityRisk, EntityRiskDetail } from "@/api/aiops";

interface TopEntitiesProps {
  entities: EntityRisk[];
  clusterId: string;
}

function formatTimeAgo(ts: number, t: { minutesAgo: string; hoursAgo: string; justNow: string; noAnomaly: string }): string {
  if (!ts) return t.noAnomaly;
  const diffMs = Date.now() - ts;
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return t.justNow;
  if (diffMin < 60) return `${diffMin} ${t.minutesAgo}`;
  return `${Math.floor(diffMin / 60)} ${t.hoursAgo}`;
}

export function TopEntities({ entities, clusterId }: TopEntitiesProps) {
  const { t } = useI18n();
  const [expandedKey, setExpandedKey] = useState<string | null>(null);
  const [detailCache, setDetailCache] = useState<Record<string, EntityRiskDetail>>({});
  const [loadingKey, setLoadingKey] = useState<string | null>(null);

  const handleToggle = useCallback(
    async (key: string) => {
      if (expandedKey === key) {
        setExpandedKey(null);
        return;
      }
      setExpandedKey(key);

      if (!detailCache[key]) {
        setLoadingKey(key);
        try {
          const detail = await getEntityRiskDetail(clusterId, key);
          setDetailCache((prev) => ({ ...prev, [key]: detail }));
        } catch (err) {
          console.error("Failed to load entity detail:", err);
        } finally {
          setLoadingKey(null);
        }
      }
    },
    [expandedKey, clusterId, detailCache]
  );

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-5 py-3 border-b border-[var(--border-color)]">
        <h3 className="text-sm font-semibold text-default">{t.aiops.topRiskEntities}</h3>
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
          const detail = detailCache[entity.entityKey];
          const isLoading = loadingKey === entity.entityKey;

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
                <span className="text-xs font-mono text-default">{entity.rLocal.toFixed(1)}</span>
                <span className="text-xs font-mono font-semibold text-default">{entity.rFinal.toFixed(1)}</span>
                <RiskBadge level={entity.riskLevel} />
                <span className="text-xs text-muted">{formatTimeAgo(entity.firstAnomaly, t.aiops)}</span>
              </button>

              {/* 展开详情 */}
              {isExpanded && (
                <div className="px-5 pb-4 pt-1 bg-[var(--background)]/50 border-b border-[var(--border-color)]/30">
                  {isLoading ? (
                    <div className="flex items-center justify-center py-4">
                      <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
                    </div>
                  ) : detail ? (
                    <div className="space-y-3">
                      {/* 异常指标 */}
                      {detail.metrics.length > 0 && (
                        <div>
                          <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.metricName}</h4>
                          <div className="space-y-1">
                            {detail.metrics.map((m, i) => (
                              <div key={i} className="flex items-center gap-3 text-xs">
                                <span className="font-mono text-default w-40 truncate">{m.metricName}</span>
                                <span className="text-muted">
                                  {t.aiops.currentValue}: <span className="text-default">{m.currentValue.toFixed(2)}</span>
                                </span>
                                <span className="text-muted">
                                  {t.aiops.baseline}: <span className="text-default">{m.baseline.toFixed(2)}</span>
                                </span>
                                <span className="text-muted">
                                  {t.aiops.deviation}: <span className="text-default">{m.deviation.toFixed(1)}\u03c3</span>
                                </span>
                                {m.isAnomaly && (
                                  <span className="text-red-500 text-[10px] font-medium">{t.aiops.isAnomaly}</span>
                                )}
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {/* 因果链 */}
                      {detail.causalChain.length > 0 && (
                        <div>
                          <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.causalChain}</h4>
                          <div className="space-y-1">
                            {detail.causalChain.map((c, i) => (
                              <div key={i} className="flex items-center gap-2 text-xs">
                                <span className="text-muted">{i + 1}.</span>
                                <EntityLink entityKey={c.entityKey} showType={false} />
                                <span className="font-mono text-default">{c.metricName}</span>
                                <span className="text-muted">{c.deviation.toFixed(1)}\u03c3</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ) : null}
                </div>
              )}
            </div>
          );
        })
      )}
    </div>
  );
}
