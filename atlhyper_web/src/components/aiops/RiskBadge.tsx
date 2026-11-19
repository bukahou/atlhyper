"use client";

import { useI18n } from "@/i18n/context";
import { levelBadgeClass } from "@/lib/risk";

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
  const style = levelBadgeClass(level);
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
