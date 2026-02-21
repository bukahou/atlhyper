"use client";

import { useState, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import {
  RefreshCw,
  WifiOff,
  Calendar,
  ChevronDown,
} from "lucide-react";

import { mockQueryLogs } from "@/mock/logs";

import { LogToolbar } from "./components/LogToolbar";
import { LogFacets } from "./components/LogFacets";
import { LogList } from "./components/LogList";

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

  // Time range
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

  const [timeRange, setTimeRange] = useState("15d");
  const [showTimeDropdown, setShowTimeDropdown] = useState(false);

  // Query logs
  const result = useMemo(() => mockQueryLogs({
    search,
    services: selectedServices,
    severities: selectedSeverities,
    scopes: selectedScopes,
    limit: displayCount,
  }), [search, selectedServices, selectedSeverities, selectedScopes, displayCount]);

  const handleLoadMore = () => setDisplayCount((c) => c + 50);

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

        {/* Main content: facets + log list */}
        <div className="flex gap-4">
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
            t={tl}
          />

          <div className="flex-1 min-w-0">
            <LogList
              logs={result.logs}
              total={result.total}
              displayCount={displayCount}
              onLoadMore={handleLoadMore}
              t={tl}
            />
          </div>
        </div>
      </div>
    </Layout>
  );
}
