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

  if (loading) {
    return <LoadingSpinner />;
  }

  if (error) {
    return <div className="text-center py-12 text-red-500">{error}</div>;
  }

  if (!data.length) {
    return <div className="text-center py-12 text-gray-500">{t.common.noData}</div>;
  }

  return (
    <div>
      <div className="overflow-x-auto">
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
                    {col.render
                      ? col.render(item)
                      : String((item as Record<string, unknown>)[col.key] ?? "")}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 分页控制 */}
      {showPagination && totalPages > 0 && (
        <div className="flex items-center justify-between px-4 py-3 border-t border-[var(--border-color)] bg-[var(--background)]">
          <div className="flex items-center gap-4 text-sm text-muted">
            <span>
              显示 {startIndex}-{endIndex} / 共 {data.length} 条
            </span>
            <div className="flex items-center gap-2">
              <span>每页</span>
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
              <span>条</span>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPage === 1}
              className="p-1.5 rounded hover:bg-card disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>

            {/* 页码按钮 */}
            <div className="flex items-center gap-1">
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                let pageNum: number;
                if (totalPages <= 5) {
                  pageNum = i + 1;
                } else if (currentPage <= 3) {
                  pageNum = i + 1;
                } else if (currentPage >= totalPages - 2) {
                  pageNum = totalPages - 4 + i;
                } else {
                  pageNum = currentPage - 2 + i;
                }

                return (
                  <button
                    key={pageNum}
                    onClick={() => setCurrentPage(pageNum)}
                    className={`w-8 h-8 text-sm rounded transition-colors ${
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
              className="p-1.5 rounded hover:bg-card disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
