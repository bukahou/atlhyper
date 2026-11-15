"use client";

interface SegmentedFilterProps<T extends string> {
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
}

/** 列表上方的状态筛选：几个互斥的选项排成一行 */
export function SegmentedFilter<T extends string>({ options, value, onChange }: SegmentedFilterProps<T>) {
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          aria-pressed={value === option.value}
          onClick={() => onChange(option.value)}
          className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${
            value === option.value
              ? "bg-primary text-white border-primary"
              : "bg-card text-muted border-[var(--border-color)] hover:text-default"
          }`}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
