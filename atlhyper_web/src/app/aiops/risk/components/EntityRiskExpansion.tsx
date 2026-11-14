"use client";

import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { BaselineCard } from "@/components/aiops/BaselineCard";
import { CausalTreeNodeView } from "@/components/aiops/CausalTreeNodeView";
import { EntityLink } from "@/components/aiops/EntityLink";
import { entityBaselineQuery, entityRiskDetailQuery } from "@/queries/aiops";

interface EntityRiskExpansionProps {
  clusterId: string;
  entityKey: string;
}

/**
 * 风险实体展开后的内容：异常指标、因果树（或因果链）、指标基线。
 *
 * 只在展开时挂载，所以详情与基线只在展开时才取；两份各自成一个查询 ——
 * 基线取不到不挡住风险详情，反之亦然。
 */
export function EntityRiskExpansion({ clusterId, entityKey }: EntityRiskExpansionProps) {
  const { t } = useI18n();
  const detailQuery = useQuery(entityRiskDetailQuery(clusterId, entityKey));
  const baselineQuery = useQuery(entityBaselineQuery(clusterId, entityKey));
  const detail = detailQuery.data;

  return (
    <div className="px-5 pb-4 pt-1 bg-[var(--background)]/50 border-b border-[var(--border-color)]/30">
      {detailQuery.isPending ? (
        <div className="flex items-center justify-center py-4">
          <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
        </div>
      ) : detail ? (
        <div className="space-y-4">
          {detail.metrics.length === 0 &&
          detail.causalChain.length === 0 &&
          (!detail.causalTree || detail.causalTree.length === 0) ? (
            <div className="py-2 text-xs text-muted">{t.aiops.noAnomalyDetail}</div>
          ) : (
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
                          {t.aiops.deviation}: <span className="text-default">{m.deviation.toFixed(1)}σ</span>
                        </span>
                        {m.isAnomaly && (
                          <span className="text-red-500 text-[10px] font-medium">{t.aiops.isAnomaly}</span>
                        )}
                        {m.absoluteBreach && (
                          <span
                            className="text-[10px] px-1 py-0.5 rounded bg-red-500/15 text-red-500 font-medium"
                            title={t.aiops.absoluteBreachHint}
                          >
                            {t.aiops.absoluteBreach}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* 因果树（优先）或因果链（降级） */}
              {detail.causalTree && detail.causalTree.length > 0 ? (
                <div>
                  <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.causalTree}</h4>
                  <div className="space-y-1">
                    {detail.causalTree.map((node, i) => (
                      <CausalTreeNodeView key={i} node={node} depth={0} t={t.aiops} />
                    ))}
                  </div>
                </div>
              ) : detail.causalChain.length > 0 ? (
                <div>
                  <h4 className="text-xs font-medium text-muted mb-2">{t.aiops.causalChain}</h4>
                  <div className="space-y-1">
                    {detail.causalChain.map((c, i) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <span className="text-muted">{i + 1}.</span>
                        <EntityLink entityKey={c.entityKey} showType={false} />
                        <span className="font-mono text-default">{c.metricName}</span>
                        <span className="text-muted">{c.deviation.toFixed(1)}σ</span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
            </div>
          )}

          {/* 基线：无论是否有异常都展示 —— 健康实体此前展开只有一行
              「暂无异常详情」，基线才是这里真正有信息量的内容 */}
          <BaselineCard baseline={baselineQuery.data ?? null} />
        </div>
      ) : (
        // 详情没取到：说出来，而不是留一块空白
        <div className="py-3 text-center text-xs text-muted">{t.aiops.loadFailed}</div>
      )}
    </div>
  );
}
