"use client";

import type { StatefulSetDetail } from "@/api/workload";

interface StrategyTabProps {
  detail: StatefulSetDetail;
}

export function StrategyTab({ detail }: StrategyTabProps) {
  const strategy = detail.spec.updateStrategy;

  return (
    <div className="space-y-6">
      {/* 更新策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">更新策略</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">策略类型</div>
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
        <h3 className="text-sm font-semibold text-default mb-3">Pod 管理</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Pod 管理策略</div>
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
      {detail.spec.pvcRetentionPolicy && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">PVC 保留策略</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">删除时</div>
              <div className="text-sm font-medium text-default">
                {detail.spec.pvcRetentionPolicy.whenDeleted || "Retain"}
              </div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">缩容时</div>
              <div className="text-sm font-medium text-default">
                {detail.spec.pvcRetentionPolicy.whenScaled || "Retain"}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 调度信息 */}
      {(detail.template?.nodeSelector || detail.template?.tolerations?.length) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">调度</h3>
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
                  {detail.template.tolerations.slice(0, 5).map((t, i) => (
                    <div key={i} className="text-xs font-mono text-default">
                      {t.key || "*"} {t.operator || "Equal"} {t.value || ""} : {t.effect || "All"}
                    </div>
                  ))}
                  {detail.template.tolerations.length > 5 && (
                    <div className="text-xs text-muted">... 还有 {detail.template.tolerations.length - 5} 个</div>
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
