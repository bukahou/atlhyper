"use client";

import { useI18n } from "@/i18n/context";

const LEVEL_STYLES: Record<string, string> = {
  healthy: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
  low: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  medium: "bg-yellow-500/15 text-yellow-700 dark:text-yellow-400",
  warning: "bg-yellow-500/15 text-yellow-700 dark:text-yellow-400",
  high: "bg-orange-500/15 text-orange-600 dark:text-orange-400",
  critical: "bg-red-500/15 text-red-600 dark:text-red-400",
};

const SIZE_CLASSES = {
  sm: "text-[10px] px-1.5 py-0.5",
  md: "text-xs px-2 py-1",
};

interface RiskBadgeProps {
  level: string;
  size?: "sm" | "md";
}

export function RiskBadge({ level, size = "sm" }: RiskBadgeProps) {
  const { t } = useI18n();
  const style = LEVEL_STYLES[level] ?? LEVEL_STYLES.healthy;
  const label =
    t.aiops.riskLevel[level as keyof typeof t.aiops.riskLevel] ?? level;

  return (
    <span
      className={`inline-flex items-center rounded-full font-medium whitespace-nowrap ${style} ${SIZE_CLASSES[size]}`}
    >
      {label}
    </span>
  );
}
