import type { Activity } from "lucide-react";

export function SummaryCard({
  icon: Icon,
  label,
  value,
  subValue,
  color,
}: {
  icon: typeof Activity;
  label: string;
  value: string;
  subValue?: string;
  color: string;
}) {
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3">
      <div className="flex items-center gap-2 mb-2">
        <div className={`p-1.5 rounded-lg ${color}`}>
          <Icon className="w-4 h-4 sm:w-5 sm:h-5" />
        </div>
        <span className="text-xs text-muted">{label}</span>
      </div>
      <div className="text-xl font-bold text-default">{value}</div>
      {subValue && <div className="text-[10px] text-muted mt-0.5">{subValue}</div>}
    </div>
  );
}
