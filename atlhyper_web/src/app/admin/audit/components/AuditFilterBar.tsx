import { Filter } from "lucide-react";
import type { AuditTranslations, CommonTranslations } from "@/types/i18n";

export type FilterResult = "all" | "success" | "failed";

interface AuditFilterBarProps {
  auditT: AuditTranslations;
  commonT: CommonTranslations;
  timeRange: number;
  onTimeRangeChange: (value: number) => void;
  filterUser: string;
  onFilterUserChange: (value: string) => void;
  filterResult: FilterResult;
  onFilterResultChange: (value: FilterResult) => void;
  stats: { total: number; success: number; failed: number };
}

/** 审计日志筛选栏 + 统计信息 */
export function AuditFilterBar({
  auditT,
  commonT,
  timeRange,
  onTimeRangeChange,
  filterUser,
  onFilterUserChange,
  filterResult,
  onFilterResultChange,
  stats,
}: AuditFilterBarProps) {
  const timeRanges = [
    { label: auditT.lastHour, value: 1 },
    { label: auditT.last24Hours, value: 24 },
    { label: auditT.last7Days, value: 168 },
    { label: auditT.allTime, value: 0 },
  ];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Filter className="w-4 h-4 text-muted" />
        <span className="text-sm font-medium text-default">{auditT.filterLabel}</span>
      </div>
      <div className="flex flex-wrap gap-4">
        {/* 时间范围 */}
        <div>
          <label className="block text-xs text-muted mb-1">{auditT.timeRange}</label>
          <select
            value={timeRange}
            onChange={(e) => onTimeRangeChange(Number(e.target.value))}
            className="px-3 py-1.5 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none"
          >
            {timeRanges.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        {/* 用户过滤 */}
        <div>
          <label className="block text-xs text-muted mb-1">{auditT.user}</label>
          <input
            type="text"
            placeholder={commonT.search + "..."}
            value={filterUser}
            onChange={(e) => onFilterUserChange(e.target.value)}
            className="px-3 py-1.5 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none w-40"
          />
        </div>

        {/* 结果过滤 */}
        <div>
          <label className="block text-xs text-muted mb-1">{auditT.result}</label>
          <select
            value={filterResult}
            onChange={(e) => onFilterResultChange(e.target.value as FilterResult)}
            className="px-3 py-1.5 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none"
          >
            <option value="all">{auditT.all}</option>
            <option value="success">{auditT.successOnly}</option>
            <option value="failed">{auditT.failedOnly}</option>
          </select>
        </div>

        {/* 统计信息 */}
        <div className="flex items-end gap-4 ml-auto text-sm">
          <span className="text-muted">
            {auditT.total} <span className="font-medium text-default">{stats.total}</span>
          </span>
          <span className="text-green-600 dark:text-green-400">
            {auditT.successCount} {stats.success}
          </span>
          <span className="text-red-600 dark:text-red-400">
            {auditT.failedCount} {stats.failed}
          </span>
        </div>
      </div>
    </div>
  );
}
