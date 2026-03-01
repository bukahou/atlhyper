"use client";

interface CommandPaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  t: {
    table: { showing: string; entries: string; previousPage: string; nextPage: string };
  };
}

export function CommandPagination({
  page,
  pageSize,
  total,
  onPageChange,
  t,
}: CommandPaginationProps) {
  if (total <= pageSize) return null;

  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-muted">
        {t.table.showing} {page * pageSize + 1}-{Math.min((page + 1) * pageSize, total)} / {total} {t.table.entries}
      </span>
      <div className="flex gap-2">
        <button
          onClick={() => onPageChange(Math.max(0, page - 1))}
          disabled={page === 0}
          className="px-3 py-1 text-sm border border-[var(--border-color)] rounded-lg hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {t.table.previousPage}
        </button>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={(page + 1) * pageSize >= total}
          className="px-3 py-1 text-sm border border-[var(--border-color)] rounded-lg hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {t.table.nextPage}
        </button>
      </div>
    </div>
  );
}
