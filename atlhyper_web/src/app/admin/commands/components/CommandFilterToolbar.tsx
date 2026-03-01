"use client";

import { Filter, X } from "lucide-react";
import { FilterInput, FilterSelect } from ".";

interface CommandFilterToolbarProps {
  sourceFilter: string;
  statusFilter: string;
  actionFilter: string;
  searchTerm: string;
  onSourceChange: (v: string) => void;
  onStatusChange: (v: string) => void;
  onActionChange: (v: string) => void;
  onSearchChange: (v: string) => void;
  onClearAll: () => void;
  actionOptions: { value: string; label: string }[];
  commandCount: number;
  total: number;
  t: {
    common: { filter: string; clearAll: string; items: string };
    commands: {
      allSources: string;
      allStatus: string;
      allActions: string;
      searchPlaceholder: string;
      sources: { web: string; ai: string };
      statuses: { pending: string; running: string; success: string; failed: string; timeout: string };
    };
  };
}

export function CommandFilterToolbar({
  sourceFilter,
  statusFilter,
  actionFilter,
  searchTerm,
  onSourceChange,
  onStatusChange,
  onActionChange,
  onSearchChange,
  onClearAll,
  actionOptions,
  commandCount,
  total,
  t,
}: CommandFilterToolbarProps) {
  const activeFilterCount = [sourceFilter, statusFilter, actionFilter, searchTerm].filter(Boolean).length;
  const hasActiveFilters = activeFilterCount > 0;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Filter className="w-4 h-4 text-muted" />
        <span className="text-sm font-medium text-default">{t.common.filter}</span>
        {activeFilterCount > 0 && (
          <span className="px-1.5 py-0.5 text-xs bg-primary/10 text-primary rounded">
            {activeFilterCount}
          </span>
        )}
        {hasActiveFilters && (
          <button
            onClick={onClearAll}
            className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
          >
            <X className="w-3 h-3" />
            {t.common.clearAll}
          </button>
        )}
      </div>

      <div className="flex flex-wrap gap-3 items-center">
        <FilterSelect
          value={sourceFilter}
          onChange={onSourceChange}
          onClear={() => onSourceChange("")}
          placeholder={t.commands.allSources}
          options={[
            { value: "web", label: t.commands.sources.web },
            { value: "ai", label: t.commands.sources.ai },
          ]}
        />
        <FilterSelect
          value={statusFilter}
          onChange={onStatusChange}
          onClear={() => onStatusChange("")}
          placeholder={t.commands.allStatus}
          options={[
            { value: "pending", label: t.commands.statuses.pending },
            { value: "running", label: t.commands.statuses.running },
            { value: "success", label: t.commands.statuses.success },
            { value: "failed", label: t.commands.statuses.failed },
            { value: "timeout", label: t.commands.statuses.timeout },
          ]}
        />
        <FilterSelect
          value={actionFilter}
          onChange={onActionChange}
          onClear={() => onActionChange("")}
          placeholder={t.commands.allActions}
          options={actionOptions}
        />
        <FilterInput
          value={searchTerm}
          onChange={onSearchChange}
          onClear={() => onSearchChange("")}
          placeholder={t.commands.searchPlaceholder}
        />
        <span className="text-sm text-muted whitespace-nowrap">
          {commandCount} / {total} {t.common.items}
        </span>
      </div>
    </div>
  );
}
