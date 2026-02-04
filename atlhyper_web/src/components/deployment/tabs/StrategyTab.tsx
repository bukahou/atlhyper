"use client";

import type { DeploymentDetail } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";
import { InfoCard } from "./InfoCard";

interface StrategyTabProps {
  detail: DeploymentDetail;
  t: ReturnType<typeof useI18n>["t"];
}

export function StrategyTab({ detail, t }: StrategyTabProps) {
  const spec = detail.spec;
  return (
    <div className="space-y-6">
      {/* 更新策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.updateStrategy}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label={t.deployment.strategyType} value={spec?.strategyType || detail.strategy || "-"} />
          <InfoCard label="MaxUnavailable" value={spec?.maxUnavailable || "-"} />
          <InfoCard label="MaxSurge" value={spec?.maxSurge || "-"} />
        </div>
      </div>

      {/* 其他配置 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.otherConfig}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label="MinReadySeconds" value={spec?.minReadySeconds ?? 0} />
          <InfoCard label="RevisionHistoryLimit" value={spec?.revisionHistoryLimit ?? 10} />
          <InfoCard label="ProgressDeadlineSeconds" value={spec?.progressDeadlineSeconds ?? 600} />
        </div>
      </div>

      {/* 选择器 */}
      {spec?.matchLabels && Object.keys(spec.matchLabels).length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.selector}</h3>
          <div className="space-y-2">
            {Object.entries(spec.matchLabels).map(([k, v]) => (
              <div key={k} className="bg-[var(--background)] rounded-lg p-3 flex items-start gap-2">
                <span className="text-sm font-mono text-primary">{k}</span>
                <span className="text-muted">=</span>
                <span className="text-sm font-mono text-default">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
