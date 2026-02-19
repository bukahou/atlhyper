"use client";

interface ImpactBarProps {
  value: number; // 0-1
  color?: string;
}

export function ImpactBar({ value, color = "#3b82f6" }: ImpactBarProps) {
  const pct = Math.max(0, Math.min(1, value)) * 100;
  return (
    <div className="w-20 h-2 rounded-full bg-[var(--border-color)] overflow-hidden">
      <div
        className="h-full rounded-full transition-all"
        style={{ width: `${pct}%`, backgroundColor: color }}
      />
    </div>
  );
}
