"use client";

import { useI18n } from "@/i18n/context";
import { Filter, X } from "lucide-react";

export interface NodeFilters {
  search: string;
  status: string;
  architecture: string;
  schedulable: string;
}

/** 没有任何筛选条件 */
export const NO_NODE_FILTERS: NodeFilters = { search: "", status: "", architecture: "", schedulable: "" };

// 带清除按钮的筛选输入框
function FilterInput({
  value,
  onChange,
  onClear,
  placeholder,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
}) {
  return (
    <div className="relative">
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2.5 sm:py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary"
      />
      {value && (
        <button
          onClick={onClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-muted hover:text-default transition-colors"
        >
          <X className="w-4 h-4 sm:w-3 sm:h-3" />
        </button>
      )}
    </div>
  );
}

// 带清除按钮的筛选下拉框
function FilterSelect({
  value,
  onChange,
  onClear,
  placeholder,
  options,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
  options: { value: string; label: string }[];
}) {
  return (
    <div className="relative">
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2.5 sm:py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary appearance-none"
      >
        <option value="">{placeholder}</option>
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {value ? (
        <button
          onClick={(e) => {
            e.preventDefault();
            onClear();
          }}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-muted hover:text-default transition-colors z-10"
        >
          <X className="w-4 h-4 sm:w-3 sm:h-3" />
        </button>
      ) : (
        <div className="absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none">
          <svg className="w-4 h-4 text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      )}
    </div>
  );
}

// 筛选栏组件
export function NodeFilterBar({
  filters,
  architectureOptions,
  onFilterChange,
  onClearAll,
}: {
  filters: NodeFilters;
  /** 架构选项从当前节点列表里动态提取 */
  architectureOptions: { value: string; label: string }[];
  onFilterChange: (key: string, value: string) => void;
  onClearAll: () => void;
}) {
  const { t } = useI18n();
  const hasFilters = filters.search || filters.status || filters.architecture || filters.schedulable;
  const activeCount = [filters.search, filters.status, filters.architecture, filters.schedulable].filter(Boolean).length;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Filter className="w-4 h-4 text-muted" />
        <span className="text-sm font-medium text-default">{t.common.filter}</span>
        {activeCount > 0 && (
          <span className="px-1.5 py-0.5 text-xs bg-primary/10 text-primary rounded">
            {activeCount}
          </span>
        )}
        {hasFilters && (
          <button
            onClick={onClearAll}
            className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
          >
            <X className="w-3 h-3" />
            {t.common.clearAll}
          </button>
        )}
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <FilterInput
          value={filters.search}
          onChange={(v) => onFilterChange("search", v)}
          onClear={() => onFilterChange("search", "")}
          placeholder={t.node.searchPlaceholder}
        />
        <FilterSelect
          value={filters.status}
          onChange={(v) => onFilterChange("status", v)}
          onClear={() => onFilterChange("status", "")}
          placeholder={t.node.allStatus}
          options={[
            { value: "Ready", label: "Ready" },
            { value: "NotReady", label: "NotReady" },
          ]}
        />
        <FilterSelect
          value={filters.architecture}
          onChange={(v) => onFilterChange("architecture", v)}
          onClear={() => onFilterChange("architecture", "")}
          placeholder={t.node.allArchitectures}
          options={architectureOptions}
        />
        <FilterSelect
          value={filters.schedulable}
          onChange={(v) => onFilterChange("schedulable", v)}
          onClear={() => onFilterChange("schedulable", "")}
          placeholder={t.node.allSchedulable}
          options={[
            { value: "Schedulable", label: t.node.schedulable },
            { value: "Unschedulable", label: t.node.unschedulable },
          ]}
        />
      </div>
    </div>
  );
}
