"use client";

import { Search } from "lucide-react";
import type { LogTranslations } from "@/types/i18n";

interface LogToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  page: number;
  pageSize: number;
  total: number;
  t: LogTranslations;
}

export function LogToolbar({ search, onSearchChange, page, pageSize, total, t }: LogToolbarProps) {
  const start = Math.min((page - 1) * pageSize + 1, total);
  const end = Math.min(page * pageSize, total);
  const rangeText = total > 0
    ? t.showing.replace("{count}", `${start}-${end}`).replace("{total}", String(total))
    : t.showing.replace("{count}", "0").replace("{total}", "0");

  return (
    <div className="flex items-center gap-3">
      <div className="flex-1 relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted" />
        <input
          type="text"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={t.searchPlaceholder}
          className="w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-card text-default placeholder:text-muted focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors"
        />
      </div>
      <span className="text-xs text-muted whitespace-nowrap tabular-nums">
        {rangeText}
      </span>
    </div>
  );
}
