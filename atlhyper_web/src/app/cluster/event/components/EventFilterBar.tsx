"use client";

import { Filter, X } from "lucide-react";
import { useI18n } from "@/i18n/context";

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

export interface EventFilters {
  search: string;
  severity: string;
  kind: string;
  namespace: string;
}

interface EventFilterBarProps {
  filters: EventFilters;
  onFilterChange: (key: string, value: string) => void;
  onClearAll: () => void;
  kinds: string[];
  namespaces: string[];
}

export function EventFilterBar({
  filters,
  onFilterChange,
  onClearAll,
  kinds,
  namespaces,
}: EventFilterBarProps) {
  const { t } = useI18n();

  const hasFilters = filters.search || filters.severity || filters.kind || filters.namespace;
  const activeCount = [filters.search, filters.severity, filters.kind, filters.namespace].filter(Boolean).length;

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
          placeholder={t.event.searchPlaceholder}
        />
        <FilterSelect
          value={filters.severity}
          onChange={(v) => onFilterChange("severity", v)}
          onClear={() => onFilterChange("severity", "")}
          placeholder={t.event.allSeverities}
          options={[
            { value: "Normal", label: "Normal" },
            { value: "Warning", label: "Warning" },
            { value: "Critical", label: "Critical" },
          ]}
        />
        <FilterSelect
          value={filters.kind}
          onChange={(v) => onFilterChange("kind", v)}
          onClear={() => onFilterChange("kind", "")}
          placeholder={t.event.allKinds}
          options={kinds.map((k) => ({ value: k, label: k }))}
        />
        <FilterSelect
          value={filters.namespace}
          onChange={(v) => onFilterChange("namespace", v)}
          onClear={() => onFilterChange("namespace", "")}
          placeholder={t.event.allNamespaces}
          options={namespaces.map((ns) => ({ value: ns, label: ns }))}
        />
      </div>
    </div>
  );
}
