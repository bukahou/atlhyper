"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  labels: {
    showing: string;
    entries: string;
    previousPage: string;
    nextPage: string;
  };
}

export function Pagination({
  page,
  pageSize,
  total,
  onPageChange,
  labels,
}: PaginationProps) {
  if (total <= pageSize) return null;

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="flex items-center justify-between px-6 py-3 border-t border-[var(--border-color)]">
      <span className="text-xs text-muted">
        {labels.showing} {page * pageSize + 1}-
        {Math.min((page + 1) * pageSize, total)} / {total} {labels.entries}
      </span>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page === 0}
          className="p-1 rounded text-muted hover:text-default hover:bg-[var(--bg-secondary)] transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>
        <span className="text-xs text-muted px-2">
          {page + 1} / {totalPages}
        </span>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page + 1 >= totalPages}
          className="p-1 rounded text-muted hover:text-default hover:bg-[var(--bg-secondary)] transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}

/** 对数组进行分页切片 */
export function paginate<T>(items: T[], page: number, pageSize: number): T[] {
  return items.slice(page * pageSize, (page + 1) * pageSize);
}
