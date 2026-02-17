"use client";

import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";
import { EntityLink } from "@/components/aiops/EntityLink";
import { getEntityRiskDetail } from "@/api/aiops";
import type { EntityRiskDetail } from "@/api/aiops";

interface NodeDetailProps {
  entityKey: string;
  clusterId: string;
}

export function NodeDetail({ entityKey, clusterId }: NodeDetailProps) {
  const { t } = useI18n();
  const [detail, setDetail] = useState<EntityRiskDetail | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getEntityRiskDetail(clusterId, entityKey)
      .then(setDetail)
      .catch((err) => console.error("Failed to load entity detail:", err))
      .finally(() => setLoading(false));
  }, [clusterId, entityKey]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
      </div>
    );
  }

  if (!detail) {
    return <div className="text-sm text-muted text-center py-8">{t.aiops.noData}</div>;
  }

  return (
    <div className="space-y-4 overflow-y-auto max-h-full">
      {/* 实体信息 */}
      <div>
        <h3 className="text-sm font-bold text-default mb-2">{t.aiops.nodeDetail}</h3>
        <div className="space-y-1.5 text-xs">
          <div className="flex items-center gap-2">
            <EntityLink entityKey={detail.entityKey} />
          </div>
          <div className="text-muted">
            {t.aiops.entityType}: <span className="text-default">{detail.entityType}</span>
          </div>
          {detail.namespace && (
            <div className="text-muted">
              {t.common.namespace}: <span className="text-default">{detail.namespace}</span>
            </div>
          )}
        </div>
      </div>

      {/* 风险分数 */}
      <div className="bg-[var(--background)] rounded-lg p-3 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">{t.aiops.rLocal}</span>
          <span className="text-sm font-mono font-medium text-default">{detail.rLocal.toFixed(1)}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">{t.aiops.rFinal}</span>
          <span className="text-sm font-mono font-bold text-default">{detail.rFinal.toFixed(1)}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">{t.aiops.riskLevelLabel}</span>
          <RiskBadge level={detail.riskLevel} size="md" />
        </div>
      </div>

      {/* 指标列表 */}
      {detail.metrics?.length > 0 && (
        <div>
          <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.metricName}</h4>
          <div className="space-y-2">
            {detail.metrics.map((m, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-2.5 text-xs space-y-1">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-default">{m.metricName}</span>
                  {m.isAnomaly && (
                    <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/15 text-red-500 font-medium">
                      {t.aiops.isAnomaly}
                    </span>
                  )}
                </div>
                <div className="flex gap-3 text-muted">
                  <span>
                    {t.aiops.currentValue}: <span className="text-default">{m.currentValue.toFixed(2)}</span>
                  </span>
                  <span>
                    {t.aiops.deviation}: <span className="text-default">{m.deviation.toFixed(1)}σ</span>
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 因果链 */}
      {detail.causalChain?.length > 0 && (
        <div>
          <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.causalChain}</h4>
          <div className="space-y-1.5">
            {detail.causalChain.map((c, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <span className="text-muted w-4 text-right">{i + 1}.</span>
                <EntityLink entityKey={c.entityKey} showType={false} />
                <span className="font-mono text-muted">{c.metricName}</span>
                <span className="text-default">{c.deviation.toFixed(1)}σ</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 传播路径 */}
      {detail.propagation?.length > 0 && (
        <div>
          <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{t.aiops.propagation}</h4>
          <div className="space-y-1.5">
            {detail.propagation.map((p, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <EntityLink entityKey={p.from} showType={false} />
                <span className="text-muted">→</span>
                <EntityLink entityKey={p.to} showType={false} />
                <span className="text-muted">({(p.contribution * 100).toFixed(0)}%)</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
