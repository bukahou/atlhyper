"use client";

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import {
  RefreshCw,
  WifiOff,
  ChevronRight,
  X,
  Clock,
} from "lucide-react";

import { queryLogs, queryLogHistogram } from "@/datasource/logs";
import { TimeRangePicker } from "@/components/common";
import { toSince, toAbsoluteParams, toSpanMs } from "@/lib/time-range";
import type { TimeRangeSelection } from "@/types/time-range";

import type { LogEntry, LogQueryResult, LogHistogramBucket } from "@/types/model/log";

import { LogToolbar } from "./components/LogToolbar";
import { LogFacets } from "./components/LogFacets";
import { LogList } from "./components/LogList";
import { LogHistogram } from "./components/LogHistogram";
import { LogDetailDrawer } from "./components/LogDetail";

/** Pill color by filter type */
function pillStyle(type: "severity" | "service" | "scope", value?: string) {
  if (type === "severity") {
    switch (value?.toUpperCase()) {
      case "ERROR": return "bg-red-500/15 text-red-600 dark:text-red-400";
      case "WARN": return "bg-amber-500/15 text-amber-600 dark:text-amber-400";
      case "INFO": return "bg-blue-500/15 text-blue-600 dark:text-blue-400";
      case "DEBUG": return "bg-gray-500/15 text-gray-600 dark:text-gray-400";
    }
  }
  if (type === "service") return "bg-purple-500/15 text-purple-600 dark:text-purple-400";
  return "bg-gray-500/10 text-gray-600 dark:text-gray-400";
}

/** Format epoch ms pair to "HH:MM — HH:MM" */
function formatBrushRange(start: number, end: number): string {
  const fmt = (ts: number) =>
    new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  return `${fmt(start)} — ${fmt(end)}`;
}

export default function LogsPage() {
  const { t } = useI18n();
  const tl = t.logs;
  const { currentClusterId } = useClusterStore();

  // 跨信号关联：URL ?traceId= / ?spanId= 参数过滤
  const searchParams = useSearchParams();
  const urlTraceId = searchParams.get("traceId") || undefined;
  const urlSpanId = searchParams.get("spanId") || undefined;

  // Filter state
  const [search, setSearch] = useState("");
  const [selectedServices, setSelectedServices] = useState<string[]>([]);
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const PAGE_SIZE = 50;
  const [page, setPage] = useState(1);

  // Facets panel collapse
  const [facetsCollapsed, setFacetsCollapsed] = useState(false);

  // Histogram brush time range
  const [brushTimeRange, setBrushTimeRange] = useState<[number, number] | null>(null);

  // Log detail drawer
  const [selectedEntry, setSelectedEntry] = useState<LogEntry | null>(null);
  const [selectedIdx, setSelectedIdx] = useState<number | null>(null);

  const handleSelectEntry = (entry: LogEntry, idx: number) => {
    if (selectedIdx === idx) {
      setSelectedEntry(null);
      setSelectedIdx(null);
    } else {
      setSelectedEntry(entry);
      setSelectedIdx(idx);
    }
  };
  const handleCloseDrawer = () => {
    setSelectedEntry(null);
    setSelectedIdx(null);
  };

  // Time range selection
  const [timeSelection, setTimeSelection] = useState<TimeRangeSelection>({ mode: "preset", preset: "15min" });

  // Reset page when any filter changes (including brush)
  useEffect(() => {
    setPage(1);
  }, [search, selectedServices, selectedSeverities, selectedScopes, timeSelection, brushTimeRange]);

  // Clear brush when non-brush filters change
  useEffect(() => {
    setBrushTimeRange(null);
  }, [search, selectedServices, selectedSeverities, selectedScopes, timeSelection]);

  // Histogram: 独立请求，不依赖 page 和 brushTimeRange
  const [histogramData, setHistogramData] = useState<LogHistogramBucket[]>([]);
  const [histogramIntervalMs, setHistogramIntervalMs] = useState(0);

  const loadHistogram = useCallback(async () => {
    const since = toSince(timeSelection);
    const abs = toAbsoluteParams(timeSelection);
    const data = await queryLogHistogram({
      clusterId: currentClusterId,
      search,
      services: selectedServices,
      severities: selectedSeverities,
      scopes: selectedScopes,
      since: since,
      startTime: abs.startTime,
      endTime: abs.endTime,
    });
    setHistogramData(data.buckets);
    setHistogramIntervalMs(data.intervalMs);
  }, [currentClusterId, search, selectedServices, selectedSeverities, selectedScopes, timeSelection]);

  useEffect(() => {
    loadHistogram();
  }, [loadHistogram]);

  // Query logs (async — supports both mock and real API)
  const emptyResult: LogQueryResult = { logs: [], total: 0, facets: { services: [], severities: [], scopes: [] } };
  const [result, setResult] = useState<LogQueryResult>(emptyResult);

  const loadLogs = useCallback(async () => {
    // since 始终传递（后端 facets 依赖 since 计算时间窗口）
    // Brush 选区时：since + startTime/endTime 都传，后端 buildWhere 优先使用绝对时间
    const since = toSince(timeSelection);
    const abs = toAbsoluteParams(timeSelection);
    const brushActive = brushTimeRange !== null;

    const data = await queryLogs({
      clusterId: currentClusterId,
      search,
      services: selectedServices,
      severities: selectedSeverities,
      scopes: selectedScopes,
      since,
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
      startTime: brushActive ? brushTimeRange[0] : (abs.startTime ? new Date(abs.startTime).getTime() : undefined),
      endTime: brushActive ? brushTimeRange[1] + histogramIntervalMs : (abs.endTime ? new Date(abs.endTime).getTime() : undefined),
      traceId: urlTraceId,
      spanId: urlSpanId,
    });
    setResult(data);
  }, [currentClusterId, search, selectedServices, selectedSeverities, selectedScopes, timeSelection, page, brushTimeRange, histogramIntervalMs, urlTraceId, urlSpanId]);

  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  const handlePageChange = (p: number) => {
    setPage(p);
    setSelectedEntry(null);
    setSelectedIdx(null);
  };

  // Filter pills helpers
  const hasAnyFilter =
    selectedServices.length > 0 ||
    selectedSeverities.length > 0 ||
    selectedScopes.length > 0 ||
    brushTimeRange !== null ||
    !!urlTraceId;

  const removeService = (v: string) => setSelectedServices((prev) => prev.filter((s) => s !== v));
  const removeSeverity = (v: string) => setSelectedSeverities((prev) => prev.filter((s) => s !== v));
  const removeScope = (v: string) => setSelectedScopes((prev) => prev.filter((s) => s !== v));
  const clearAllFilters = () => {
    setSelectedServices([]);
    setSelectedSeverities([]);
    setSelectedScopes([]);
    setBrushTimeRange(null);
  };

  // No cluster selected
  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">{tl.noCluster}</p>
          <p className="text-sm text-muted">{tl.noClusterDesc}</p>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-5 overflow-hidden">
        {/* Header */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl sm:text-2xl font-bold text-default">{tl.pageTitle}</h1>
            <p className="text-xs text-muted mt-1">{tl.pageDescription}</p>
          </div>

          <div className="flex items-center gap-2">
            {/* Time range picker */}
            <TimeRangePicker
              value={timeSelection}
              onChange={setTimeSelection}
              t={tl}
            />

            {/* Refresh button */}
            <button
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 transition-colors"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              {t.common.refresh}
            </button>
          </div>
        </div>

        {/* Search toolbar */}
        <LogToolbar
          search={search}
          onSearchChange={setSearch}
          page={page}
          pageSize={PAGE_SIZE}
          total={result.total}
          t={tl}
        />

        {/* Filter pills */}
        {hasAnyFilter && (
          <div className="flex flex-wrap items-center gap-1.5">
            {/* Brush time range pill */}
            {brushTimeRange && (
              <button
                onClick={() => setBrushTimeRange(null)}
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary/15 text-primary transition-colors hover:opacity-80"
              >
                <Clock className="w-3 h-3" />
                {formatBrushRange(brushTimeRange[0], brushTimeRange[1])}
                <X className="w-3 h-3" />
              </button>
            )}
            {selectedSeverities.map((v) => (
              <button
                key={`sev-${v}`}
                onClick={() => removeSeverity(v)}
                className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors hover:opacity-80 ${pillStyle("severity", v)}`}
              >
                {v}
                <X className="w-3 h-3" />
              </button>
            ))}
            {selectedServices.map((v) => (
              <button
                key={`svc-${v}`}
                onClick={() => removeService(v)}
                className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors hover:opacity-80 ${pillStyle("service")}`}
              >
                {v.replace("geass-", "")}
                <X className="w-3 h-3" />
              </button>
            ))}
            {selectedScopes.map((v) => (
              <button
                key={`scope-${v}`}
                onClick={() => removeScope(v)}
                className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-colors hover:opacity-80 ${pillStyle("scope")}`}
              >
                {v.split(".").pop()}
                <X className="w-3 h-3" />
              </button>
            ))}
            {urlTraceId && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary/15 text-primary">
                Trace: {urlTraceId.slice(0, 8)}...
              </span>
            )}
            <button
              onClick={clearAllFilters}
              className="text-xs text-muted hover:text-default transition-colors ml-1"
            >
              {tl.clearFilters}
            </button>
          </div>
        )}

        {/* Log volume histogram */}
        <LogHistogram
          data={histogramData}
          intervalMs={histogramIntervalMs}
          title={tl.logVolume}
          timeSpanMs={toSpanMs(timeSelection)}
          selectedTimeRange={brushTimeRange}
          onTimeRangeSelect={setBrushTimeRange}
        />

        {/* Main content: facets + log list */}
        <div className="flex gap-4">
          {/* Facets panel — collapsible */}
          {facetsCollapsed ? (
            <button
              onClick={() => setFacetsCollapsed(false)}
              className="w-7 flex-shrink-0 flex flex-col items-center pt-1 rounded-lg border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] transition-colors cursor-pointer"
            >
              <ChevronRight className="w-3.5 h-3.5 text-muted" />
            </button>
          ) : (
            <LogFacets
              services={result.facets.services}
              severities={result.facets.severities}
              scopes={result.facets.scopes}
              selectedServices={selectedServices}
              selectedSeverities={selectedSeverities}
              selectedScopes={selectedScopes}
              onServicesChange={setSelectedServices}
              onSeveritiesChange={setSelectedSeverities}
              onScopesChange={setSelectedScopes}
              onCollapse={() => setFacetsCollapsed(true)}
              t={tl}
            />
          )}

          <div className="flex-1 min-w-0">
            <LogList
              logs={result.logs}
              total={result.total}
              page={page}
              pageSize={PAGE_SIZE}
              onPageChange={handlePageChange}
              onSelectEntry={handleSelectEntry}
              selectedIdx={selectedIdx}
              searchHighlight={search || undefined}
              t={tl}
            />
          </div>
        </div>
      </div>

      {/* Log detail drawer */}
      <LogDetailDrawer
        entry={selectedEntry}
        onClose={handleCloseDrawer}
        t={tl}
      />
    </Layout>
  );
}
