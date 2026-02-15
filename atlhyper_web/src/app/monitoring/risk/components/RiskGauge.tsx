"use client";

import { useI18n } from "@/i18n/context";
import { RiskBadge } from "@/components/aiops/RiskBadge";

const LEVEL_COLORS: Record<string, string> = {
  healthy: "#22c55e",
  low: "#3b82f6",
  warning: "#eab308",
  critical: "#ef4444",
};

interface RiskGaugeProps {
  risk: number;
  level: string;
  anomalyCount: number;
  totalEntities: number;
}

export function RiskGauge({ risk, level, anomalyCount, totalEntities }: RiskGaugeProps) {
  const { t } = useI18n();
  const color = LEVEL_COLORS[level] ?? LEVEL_COLORS.healthy;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      <div className="flex items-center gap-2 mb-4">
        <h3 className="text-sm font-semibold text-default">{t.aiops.clusterRisk}</h3>
        <RiskBadge level={level} size="md" />
      </div>

      {/* 大数字 */}
      <div className="flex items-baseline gap-2 mb-4">
        <span className="text-5xl font-bold" style={{ color }}>
          {risk}
        </span>
        <span className="text-lg text-muted">/ 100</span>
      </div>

      {/* 进度条 */}
      <div className="w-full h-2.5 bg-[var(--background)] rounded-full overflow-hidden mb-4">
        <div
          className="h-full rounded-full transition-all duration-500"
          style={{ width: `${Math.min(risk, 100)}%`, backgroundColor: color }}
        />
      </div>

      {/* 统计 */}
      <div className="flex gap-4 text-xs text-muted">
        <span>
          {t.aiops.anomalyCount}: <span className="font-medium text-default">{anomalyCount}</span>
        </span>
        <span>
          {t.aiops.totalEntities}: <span className="font-medium text-default">{totalEntities}</span>
        </span>
      </div>
    </div>
  );
}
