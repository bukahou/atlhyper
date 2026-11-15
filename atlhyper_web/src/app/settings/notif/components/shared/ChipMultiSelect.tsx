"use client";

export interface ChipOption<T extends string | number> {
  value: T;
  label: string;
}

interface ChipMultiSelectProps<T extends string | number> {
  options: ChipOption<T>[];
  selected: T[];
  onChange: (selected: T[]) => void;
  disabled?: boolean;
  /** 一个选项都没有时显示的说明 */
  emptyText: string;
}

/** 多选：每个选项一个可点的标签，选中的高亮。选项少（集群、来源、接收人）时比下拉多选直观 */
export function ChipMultiSelect<T extends string | number>({
  options,
  selected,
  onChange,
  disabled = false,
  emptyText,
}: ChipMultiSelectProps<T>) {
  if (options.length === 0) {
    return <p className="text-sm text-muted">{emptyText}</p>;
  }

  const toggle = (value: T) => {
    onChange(selected.includes(value) ? selected.filter((held) => held !== value) : [...selected, value]);
  };

  return (
    <div className="flex flex-wrap gap-2">
      {options.map((option) => {
        const isSelected = selected.includes(option.value);
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={isSelected}
            onClick={() => toggle(option.value)}
            disabled={disabled}
            className={`px-2.5 py-1 rounded-full text-sm border transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${
              isSelected
                ? "bg-primary text-white border-primary"
                : "bg-[var(--background)] text-muted border-[var(--border-color)] hover:text-default"
            }`}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
