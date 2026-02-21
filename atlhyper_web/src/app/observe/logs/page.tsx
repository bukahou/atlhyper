"use client";

import { useState, useMemo, useEffect } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import {
  RefreshCw,
  WifiOff,
  Calendar,
  ChevronDown,
  ChevronRight,
  X,
  Clock,
} from "lucide-react";

import { queryLogs } from "@/datasource/logs";

import type { LogEntry } from "@/types/model/log";

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

  // Filter state
  const [search, setSearch] = useState("");
  const [selectedServices, setSelectedServices] = useState<string[]>([]);
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const [displayCount, setDisplayCount] = useState(50);

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

  // Time range dropdown
  const TIME_RANGE_OPTIONS = useMemo(() => [
    { value: "15min", label: tl.last15min },
    { value: "1h", label: tl.last1h },
    { value: "24h", label: tl.last24h },
    { value: "7d", label: tl.last7d },
    { value: "15d", label: tl.last15d },
    { value: "30d", label: tl.last30d },
  ], [tl]);

  const timeRangeLabels = useMemo(() => {
    const map: Record<string, string> = {};
    for (const opt of TIME_RANGE_OPTIONS) map[opt.value] = opt.label;
    return map;
  }, [TIME_RANGE_OPTIONS]);

  const [timeRange, setTimeRange] = useState("15min");
  const [showTimeDropdown, setShowTimeDropdown] = useState(false);

  // Clear brush when query filters change (data distribution changes)
  useEffect(() => {
    setBrushTimeRange(null);
  }, [search, selectedServices, selectedSeverities, selectedScopes, timeRange]);

  // Query logs
  const result = useMemo(() => queryLogs({
    search,
    services: selectedServices,
    severities: selectedSeverities,
    scopes: selectedScopes,
    timeRange,
    limit: displayCount,
  }), [search, selectedServices, selectedSeverities, selectedScopes, timeRange, displayCount]);

  // Filter logs by brush time range (client-side)
  const filteredLogs = useMemo(() => {
    if (!brushTimeRange) return result.logs;
    const [start, end] = brushTimeRange;
    return result.logs.filter((log) => {
      const ts = new Date(log.timestamp).getTime();
      return ts >= start && ts <= end;
    });
  }, [result.logs, brushTimeRange]);

  const handleLoadMore = () => setDisplayCount((c) => c + 50);

  // Filter pills helpers
  const hasAnyFilter =
    selectedServices.length > 0 ||
    selectedSeverities.length > 0 ||
    selectedScopes.length > 0 ||
    brushTimeRange !== null;

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
      <div className="space-y-4 sm:space-y-5">
        {/* Header */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl sm:text-2xl font-bold text-default">{tl.pageTitle}</h1>
            <p className="text-xs text-muted mt-1">{tl.pageDescription}</p>
          </div>

          <div className="flex items-center gap-2">
            {/* Time range dropdown */}
            <div className="relative">
              <button
                onClick={() => setShowTimeDropdown((v) => !v)}
                className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] transition-colors"
              >
                <Calendar className="w-3.5 h-3.5 text-muted" />
                <span className="text-default">{timeRangeLabels[timeRange]}</span>
                <ChevronDown className="w-3.5 h-3.5 text-muted" />
              </button>
              {showTimeDropdown && (
                <>
                  <div className="fixed inset-0 z-40" onClick={() => setShowTimeDropdown(false)} />
                  <div className="absolute right-0 top-full mt-1 z-50 min-w-[160px] py-1 rounded-lg border border-[var(--border-color)] bg-card shadow-lg">
                    {TIME_RANGE_OPTIONS.map((opt) => (
                      <button
                        key={opt.value}
                        onClick={() => {
                          setTimeRange(opt.value);
                          setShowTimeDropdown(false);
                        }}
                        className={`w-full text-left px-3 py-1.5 text-sm transition-colors ${
                          timeRange === opt.value
                            ? "text-primary bg-primary/5"
                            : "text-default hover:bg-[var(--hover-bg)]"
                        }`}
                      >
                        {opt.label}
                      </button>
                    ))}
                  </div>
                </>
              )}
            </div>

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
          displayCount={Math.min(displayCount, result.total)}
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
          data={result.histogram}
          title={tl.logVolume}
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
              logs={filteredLogs}
              total={result.total}
              displayCount={displayCount}
              onLoadMore={handleLoadMore}
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
