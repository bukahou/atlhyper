"use client";

import type { StatefulSetDetail } from "@/api/workload";
import type { useI18n } from "@/i18n/context";

type Translations = ReturnType<typeof useI18n>["t"];

interface StrategyTabProps {
  detail: StatefulSetDetail;
  t: Translations;
}

export function StrategyTab({ detail, t }: StrategyTabProps) {
  const strategy = detail.spec.updateStrategy;

  return (
    <div className="space-y-6">
      {/* 更新策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.statefulset.updateStrategy}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.statefulset.strategyType}</div>
            <div className="text-sm font-medium text-default">
              {strategy?.type || "RollingUpdate"}
            </div>
          </div>
          {strategy?.partition !== undefined && (
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">Partition</div>
              <div className="text-sm font-medium text-default">{strategy.partition}</div>
            </div>
          )}
          {strategy?.maxUnavailable && (
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">Max Unavailable</div>
              <div className="text-sm font-medium text-default">{strategy.maxUnavailable}</div>
            </div>
          )}
        </div>
      </div>

      {/* Pod 管理策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.statefulset.podManagement}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.statefulset.podManagementPolicy}</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.podManagementPolicy || "OrderedReady"}
            </div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Min Ready Seconds</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.minReadySeconds || 0}
            </div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Revision History Limit</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.revisionHistoryLimit ?? 10}
            </div>
          </div>
        </div>
      </div>

      {/* PVC 保留策略 */}
      {detail.spec.persistentVolumeClaimRetentionPolicy && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.statefulset.pvcRetentionPolicy}</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{t.statefulset.whenDeleted}</div>
              <div className="text-sm font-medium text-default">
                {detail.spec.persistentVolumeClaimRetentionPolicy.whenDeleted || "Retain"}
              </div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{t.statefulset.whenScaled}</div>
              <div className="text-sm font-medium text-default">
                {detail.spec.persistentVolumeClaimRetentionPolicy.whenScaled || "Retain"}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 调度信息 */}
      {(detail.template?.nodeSelector || detail.template?.tolerations?.length) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.statefulset.scheduling}</h3>
          <div className="space-y-3">
            {detail.template?.nodeSelector && Object.keys(detail.template.nodeSelector).length > 0 && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-2">Node Selector</div>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(detail.template.nodeSelector).map(([k, v]) => (
                    <span key={k} className="px-2 py-1 bg-[var(--card-background)] rounded text-xs font-mono">
                      {k}={v}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {detail.template?.tolerations && detail.template.tolerations.length > 0 && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-2">Tolerations ({detail.template.tolerations.length})</div>
                <div className="space-y-1">
                  {detail.template.tolerations.slice(0, 5).map((tol, i) => (
                    <div key={i} className="text-xs font-mono text-default">
                      {tol.key || "*"} {tol.operator || "Equal"} {tol.value || ""} : {tol.effect || "All"}
                    </div>
                  ))}
                  {detail.template.tolerations.length > 5 && (
                    <div className="text-xs text-muted">... {t.statefulset.moreItems.replace("{count}", String(detail.template.tolerations.length - 5))}</div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
