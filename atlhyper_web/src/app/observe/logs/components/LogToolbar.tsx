"use client";

import { Search } from "lucide-react";
import type { LogTranslations } from "@/types/i18n";

interface LogToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  displayCount: number;
  total: number;
  t: LogTranslations;
}

export function LogToolbar({ search, onSearchChange, displayCount, total, t }: LogToolbarProps) {
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
        {t.showing.replace("{count}", String(Math.min(displayCount, total))).replace("{total}", String(total))}
      </span>
    </div>
  );
}
