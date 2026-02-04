"use client";

import { useState, useMemo } from "react";
import { useI18n } from "@/i18n/context";
import { LoadingSpinner } from "./LoadingSpinner";
import { ChevronLeft, ChevronRight } from "lucide-react";

export interface TableColumn<T> {
  key: string;
  header: string;
  render?: (item: T) => React.ReactNode;
  className?: string;
  /** 移动端卡片视图中是否显示，默认 true */
  mobileVisible?: boolean;
  /** 移动端作为卡片主标题（只能有一列设置） */
  mobileTitle?: boolean;
}

interface DataTableProps<T> {
  columns: TableColumn<T>[];
  data: T[];
  loading?: boolean;
  error?: string;
  keyExtractor: (item: T, index: number) => string;
  onRowClick?: (item: T) => void;
  /** 每页显示条数，默认 10 */
  pageSize?: number;
  /** 可选的页大小选项 */
  pageSizeOptions?: number[];
  /** 是否显示分页控制，默认 true */
  showPagination?: boolean;
}

export function DataTable<T>({
  columns,
  data,
  loading,
  error,
  keyExtractor,
  onRowClick,
  pageSize: initialPageSize = 10,
  pageSizeOptions = [10, 20, 50, 100],
  showPagination = true,
}: DataTableProps<T>) {
  const { t } = useI18n();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(initialPageSize);

  // 计算分页数据
  const { paginatedData, totalPages, startIndex, endIndex } = useMemo(() => {
    const total = data.length;
    const pages = Math.ceil(total / pageSize);
    const start = (currentPage - 1) * pageSize;
    const end = Math.min(start + pageSize, total);
    const items = data.slice(start, end);

    return {
      paginatedData: items,
      totalPages: pages,
      startIndex: start + 1,
      endIndex: end,
    };
  }, [data, currentPage, pageSize]);

  // 当数据变化或页大小变化时，重置到第一页
  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setCurrentPage(1);
  };

  // 确保当前页不超过总页数
  useMemo(() => {
    if (currentPage > totalPages && totalPages > 0) {
      setCurrentPage(totalPages);
    }
  }, [currentPage, totalPages]);

  // 移动端列过滤
  const mobileColumns = useMemo(() => {
    const titleCol = columns.find((c) => c.mobileTitle);
    const otherCols = columns.filter(
      (c) => c.mobileVisible !== false && !c.mobileTitle
    );
    return { titleCol, otherCols };
  }, [columns]);

  if (loading) {
    return <LoadingSpinner />;
  }

  if (error) {
    return <div className="text-center py-12 text-red-500">{error}</div>;
  }

  if (!data.length) {
    return <div className="text-center py-12 text-gray-500">{t.common.noData}</div>;
  }

  // 渲染单元格内容
  const renderCell = (item: T, col: TableColumn<T>) => {
    if (col.render) {
      return col.render(item);
    }
    return String((item as Record<string, unknown>)[col.key] ?? "");
  };

  return (
    <div>
      {/* 桌面端表格视图 */}
      <div className="hidden md:block overflow-x-auto">
        <table className="w-full">
          <thead className="bg-[var(--background)]">
            <tr>
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-4 py-3 text-left text-sm font-medium text-gray-500 ${col.className || ""}`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-[var(--border-color)]">
            {paginatedData.map((item, index) => (
              <tr
                key={keyExtractor(item, startIndex - 1 + index)}
                className={`hover:bg-[var(--background)] ${onRowClick ? "cursor-pointer" : ""}`}
                onClick={() => onRowClick?.(item)}
              >
                {columns.map((col) => (
                  <td key={col.key} className={`px-4 py-3 text-sm ${col.className || ""}`}>
                    {renderCell(item, col)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 移动端卡片视图 */}
      <div className="md:hidden space-y-3">
        {paginatedData.map((item, index) => (
          <div
            key={keyExtractor(item, startIndex - 1 + index)}
            className={`p-4 rounded-xl border border-[var(--border-color)] bg-card ${
              onRowClick ? "cursor-pointer active:bg-[var(--hover-bg)]" : ""
            }`}
            onClick={() => onRowClick?.(item)}
          >
            {/* 卡片标题 */}
            {mobileColumns.titleCol && (
              <div className="font-medium text-default mb-2">
                {renderCell(item, mobileColumns.titleCol)}
              </div>
            )}
            {/* 其他字段 */}
            <div className="space-y-1.5">
              {mobileColumns.otherCols.map((col) => (
                <div key={col.key} className="flex justify-between items-start gap-2 text-sm">
                  <span className="text-muted flex-shrink-0">{col.header}</span>
                  <span className="text-default text-right">{renderCell(item, col)}</span>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      {/* 分页控制 */}
      {showPagination && totalPages > 0 && (
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 px-4 py-3 border-t border-[var(--border-color)] bg-[var(--background)]">
          {/* 左侧信息 */}
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-2 sm:gap-4 text-sm text-muted">
            <span>
              {t.table.showing} {startIndex}-{endIndex} / {t.common.total} {data.length} {t.table.entries}
            </span>
            <div className="flex items-center gap-2">
              <span>{t.table.rowsPerPage}</span>
              <select
                value={pageSize}
                onChange={(e) => handlePageSizeChange(Number(e.target.value))}
                className="px-2 py-1 bg-card border border-[var(--border-color)] rounded text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary"
              >
                {pageSizeOptions.map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
              <span>{t.table.entries}</span>
            </div>
          </div>

          {/* 右侧分页按钮 */}
          <div className="flex items-center gap-2 self-end sm:self-auto">
            <button
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPage === 1}
              className="p-2 sm:p-1.5 rounded hover:bg-card disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft className="w-5 h-5 sm:w-4 sm:h-4" />
            </button>

            {/* 页码按钮 - 移动端显示更少 */}
            <div className="flex items-center gap-1">
              {Array.from({ length: Math.min(3, totalPages) }, (_, i) => {
                let pageNum: number;
                if (totalPages <= 3) {
                  pageNum = i + 1;
                } else if (currentPage <= 2) {
                  pageNum = i + 1;
                } else if (currentPage >= totalPages - 1) {
                  pageNum = totalPages - 2 + i;
                } else {
                  pageNum = currentPage - 1 + i;
                }

                return (
                  <button
                    key={pageNum}
                    onClick={() => setCurrentPage(pageNum)}
                    className={`w-9 h-9 sm:w-8 sm:h-8 text-sm rounded transition-colors ${
                      currentPage === pageNum
                        ? "bg-primary text-white"
                        : "hover:bg-card text-muted"
                    }`}
                  >
                    {pageNum}
                  </button>
                );
              })}
            </div>

            <button
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={currentPage === totalPages}
              className="p-2 sm:p-1.5 rounded hover:bg-card disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronRight className="w-5 h-5 sm:w-4 sm:h-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
