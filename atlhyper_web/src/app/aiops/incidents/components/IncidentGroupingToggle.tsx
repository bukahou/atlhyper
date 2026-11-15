"use client";

import { useI18n } from "@/i18n/context";

interface IncidentGroupingToggleProps {
  grouped: boolean;
  onChange: (grouped: boolean) => void;
}

/** 列表视图切换：按组折叠（grouped=true，默认）/ 全部平铺（grouped=false） */
export function IncidentGroupingToggle({ grouped, onChange }: IncidentGroupingToggleProps) {
  const { t } = useI18n();
  const texts = t.aiops.grouping;
  const options = [
    { value: true, label: texts.groupedView },
    { value: false, label: texts.flatView },
  ];

  return (
    <div
      role="group"
      title={texts.viewToggleHint}
      className="inline-flex rounded-lg border border-[var(--border-color)] overflow-hidden"
    >
      {options.map((option) => (
        <button
          key={String(option.value)}
          type="button"
          aria-pressed={grouped === option.value}
          onClick={() => onChange(option.value)}
          className={`px-3 py-1.5 text-xs transition-colors ${
            grouped === option.value ? "bg-indigo-500 text-white" : "bg-card text-muted hover:text-default"
          }`}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
