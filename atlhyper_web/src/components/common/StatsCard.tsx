"use client";

import type { LucideIcon } from "lucide-react";

interface StatsCardProps {
  label: string;
  value: string | number;
  icon?: LucideIcon;
  iconColor?: string;
  trend?: string;
  subtitle?: string;
}

export function StatsCard({ label, value, icon: Icon, iconColor = "text-primary", trend, subtitle }: StatsCardProps) {
  return (
    <div className="bg-card rounded-xl p-3 sm:p-4 border border-[var(--border-color)]">
      <div className="flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <p className="text-xs sm:text-sm text-gray-500 truncate">{label}</p>
          <p className={`text-xl sm:text-2xl font-bold mt-0.5 sm:mt-1 ${iconColor.replace("text-", "text-") || "text-gray-900 dark:text-gray-100"}`}>
            {value}
          </p>
          {subtitle && (
            <p className="text-xs text-muted mt-0.5 truncate max-w-[100px] sm:max-w-[120px]" title={subtitle}>
              {subtitle}
            </p>
          )}
        </div>
        <div className="flex flex-col items-end gap-1 flex-shrink-0 ml-2">
          {Icon && <Icon className={`w-6 h-6 sm:w-8 sm:h-8 ${iconColor}`} />}
          {trend && (
            <span className={`text-xs ${trend.startsWith("+") ? "text-green-500" : trend.startsWith("-") ? "text-red-500" : "text-gray-500"}`}>
              {trend}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
