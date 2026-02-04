"use client";

import { memo } from "react";
import type { Server } from "lucide-react";

interface StatCardProps {
  title: string;
  value: string | number;
  subText?: string;
  icon: typeof Server;
  percent?: number;
  accentColor?: string;
}

export const StatCard = memo(function StatCard({
  title,
  value,
  subText,
  icon: Icon,
  percent,
  accentColor = "#14b8a6",
}: StatCardProps) {
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-full">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-[var(--background)] rounded-lg">
            <Icon className="w-4 h-4" style={{ color: accentColor }} />
          </div>
          <span className="text-sm font-semibold text-default">{title}</span>
        </div>
        {subText && (
          <span className="text-xs text-muted bg-[var(--background)] px-2 py-1 rounded-full">
            {subText}
          </span>
        )}
      </div>
      <div className="text-2xl font-bold text-default mb-2 transition-all duration-300">
        {typeof value === "number" && percent !== undefined
          ? `${value.toFixed(1)}%`
          : value}
      </div>
      {percent !== undefined && (
        <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-300"
            style={{ width: `${Math.min(100, percent)}%`, backgroundColor: accentColor }}
          />
        </div>
      )}
    </div>
  );
});
