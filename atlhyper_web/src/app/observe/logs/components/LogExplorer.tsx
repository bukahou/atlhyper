"use client";

import { useMemo, useState } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";
import { QueryView } from "@/components/common";
import { PermissionLock } from "@/components/common/PermissionLock";
import type { BrushedBuckets } from "@/queries/observe-logs";
import type { LogTranslations } from "@/types/i18n";
import type { LogEntry, LogFacets as LogFacetCounts, LogHistogramResult, LogQueryResult } from "@/types/model/log";
import { LogFacets } from "./LogFacets";
import { LogFilterPills } from "./LogFilterPills";
import { LogHistogram } from "./LogHistogram";
import { LogList } from "./LogList";
import { LogToolbar } from "./LogToolbar";

/** 列表还没取到（或没有权限取）时，分面面板显示的空分面 */
const NO_FACETS: LogFacetCounts = { services: [], severities: [], scopes: [] };

interface LogExplorerProps {
  tl: LogTranslations;
  histogram: LogHistogramResult;
  /** 日志列表（正文 + 分面）。没有 logs.read 时这个查询不启用，列表处显示权限锁 */
  listQuery: UseQueryResult<LogQueryResult>;
  timeSpanMs: number;
  /** 搜索框里的原始输入 */
  searchInput: string;
  onSearchInputChange: (value: string) => void;
  /** 已经生效的搜索词（去抖后），列表按它高亮 */
  searchHighlight: string;
  selectedServices: string[];
  selectedSeverities: string[];
  selectedScopes: string[];
  onServicesChange: (services: string[]) => void;
  onSeveritiesChange: (severities: string[]) => void;
  onScopesChange: (scopes: string[]) => void;
  brush: BrushedBuckets | null;
  onBrushChange: (brush: BrushedBuckets | null) => void;
  urlTraceId?: string;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  selectedIdx: number | null;
  onSelectEntry: (entry: LogEntry, idx: number) => void;
}

/** 日志页的主体：搜索栏、筛选标签、直方图、分面与列表。只在直方图取到之后渲染 */
export function LogExplorer({
  tl,
  histogram,
  listQuery,
  timeSpanMs,
  searchInput,
  onSearchInputChange,
  searchHighlight,
  selectedServices,
  selectedSeverities,
  selectedScopes,
  onServicesChange,
  onSeveritiesChange,
  onScopesChange,
  brush,
  onBrushChange,
  urlTraceId,
  page,
  pageSize,
  onPageChange,
  selectedIdx,
  onSelectEntry,
}: LogExplorerProps) {
  const [facetsCollapsed, setFacetsCollapsed] = useState(false);
  const list = listQuery.data;

  // 筛选标签与直方图高亮用的选区：[第一个桶的起点, 最后一个桶的起点]
  const brushedBucketRange = useMemo<[number, number] | null>(
    () => (brush ? [brush.firstBucketStartMs, brush.lastBucketStartMs] : null),
    [brush],
  );

  const handleBrushSelect = (bucketRange: [number, number] | null) => {
    // 桶宽在选中这一刻记下来：之后列表按「最后一个桶的起点 + 桶宽」算结束时间，不再回头读直方图
    onBrushChange(
      bucketRange && {
        firstBucketStartMs: bucketRange[0],
        lastBucketStartMs: bucketRange[1],
        bucketIntervalMs: histogram.intervalMs,
      },
    );
  };

  const clearAllFilters = () => {
    onServicesChange([]);
    onSeveritiesChange([]);
    onScopesChange([]);
    onBrushChange(null);
  };

  return (
    <div className="space-y-4 sm:space-y-5">
      {/* Search toolbar */}
      <LogToolbar
        search={searchInput}
        onSearchChange={onSearchInputChange}
        page={page}
        pageSize={pageSize}
        total={list?.total ?? 0}
        t={tl}
      />

      {/* Filter pills */}
      <LogFilterPills
        selectedSeverities={selectedSeverities}
        selectedServices={selectedServices}
        selectedScopes={selectedScopes}
        brushTimeRange={brushedBucketRange}
        urlTraceId={urlTraceId}
        onRemoveSeverity={(severity) => onSeveritiesChange(selectedSeverities.filter((s) => s !== severity))}
        onRemoveService={(service) => onServicesChange(selectedServices.filter((s) => s !== service))}
        onRemoveScope={(scope) => onScopesChange(selectedScopes.filter((s) => s !== scope))}
        onClearBrushTimeRange={() => onBrushChange(null)}
        onClearAll={clearAllFilters}
        clearFiltersLabel={tl.clearFilters}
      />

      {/* Log volume histogram */}
      <LogHistogram
        data={histogram.buckets}
        intervalMs={histogram.intervalMs}
        title={tl.logVolume}
        timeSpanMs={timeSpanMs}
        selectedTimeRange={brushedBucketRange}
        onTimeRangeSelect={handleBrushSelect}
      />

      {/* Main content: facets + log list */}
      <div className="flex gap-4">
        {facetsCollapsed ? (
          <button
            onClick={() => setFacetsCollapsed(false)}
            className="w-7 flex-shrink-0 flex flex-col items-center pt-1 rounded-lg border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] transition-colors cursor-pointer"
          >
            <ChevronRight className="w-3.5 h-3.5 text-muted" />
          </button>
        ) : (
          <LogFacets
            services={(list?.facets ?? NO_FACETS).services}
            severities={(list?.facets ?? NO_FACETS).severities}
            scopes={(list?.facets ?? NO_FACETS).scopes}
            selectedServices={selectedServices}
            selectedSeverities={selectedSeverities}
            selectedScopes={selectedScopes}
            onServicesChange={onServicesChange}
            onSeveritiesChange={onSeveritiesChange}
            onScopesChange={onScopesChange}
            onCollapse={() => setFacetsCollapsed(true)}
            t={tl}
          />
        )}

        <div className="flex-1 min-w-0">
          <PermissionLock permission="logs.read" className="bg-card rounded-xl border border-[var(--border-color)]">
            <QueryView query={listQuery}>
              {(result) => (
                <LogList
                  logs={result.logs}
                  total={result.total}
                  page={page}
                  pageSize={pageSize}
                  onPageChange={onPageChange}
                  onSelectEntry={onSelectEntry}
                  selectedIdx={selectedIdx}
                  searchHighlight={searchHighlight || undefined}
                  t={tl}
                />
              )}
            </QueryView>
          </PermissionLock>
        </div>
      </div>
    </div>
  );
}
