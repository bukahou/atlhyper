"use client";

import { useI18n } from "@/i18n/context";

interface ServerPagerProps {
  /** 从 0 起算的页码 */
  page: number;
  pageSize: number;
  /** 符合条件的总条数（由接口返回） */
  total: number;
  onPageChange: (page: number) => void;
}

/** 后端分页的翻页条：显示当前范围与总数。只有一页时不显示翻页按钮，范围与总数照常显示 */
export function ServerPager({ page, pageSize, total, onPageChange }: ServerPagerProps) {
  const { t } = useI18n();
  const firstShown = total === 0 ? 0 : page * pageSize + 1;
  const lastShown = Math.min((page + 1) * pageSize, total);
  const hasNextPage = (page + 1) * pageSize < total;

  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 border-t border-[var(--border-color)] bg-[var(--background)]">
      <span className="text-sm text-muted">
        {t.table.showing} {firstShown}-{lastShown} / {t.common.total} {total} {t.table.entries}
      </span>
      {(page > 0 || hasNextPage) && (
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => onPageChange(Math.max(0, page - 1))}
            disabled={page === 0}
            className="px-3 py-1 text-sm border border-[var(--border-color)] rounded-lg hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t.table.previousPage}
          </button>
          <button
            type="button"
            onClick={() => onPageChange(page + 1)}
            disabled={!hasNextPage}
            className="px-3 py-1 text-sm border border-[var(--border-color)] rounded-lg hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t.table.nextPage}
          </button>
        </div>
      )}
    </div>
  );
}
