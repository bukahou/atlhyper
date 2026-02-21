"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import {
  RefreshCw,
  Loader2,
  WifiOff,
  AlertTriangle,
  ChevronRight,
  Calendar,
  ChevronDown,
} from "lucide-react";

import type { TraceService, TraceSummary, TraceDetail, ServiceStats, ServiceTopologyData } from "@/api/apm";
import {
  mockGetTraceServices,
  mockQueryTraces,
  mockGetTraceDetail,
  mockGetAllServiceStats,
  mockGetServiceTopology,
} from "@/api/apm-mock";

import { ServiceList } from "./components/ServiceList";
import { ServiceOverview } from "./components/ServiceOverview";
import { TraceWaterfall } from "./components/TraceWaterfall";
import { ServiceTopology } from "./components/ServiceTopology";

type ViewState =
  | { level: "services" }
  | { level: "service-detail"; serviceName: string }
  | { level: "trace-detail"; serviceName: string; traceId: string; traceIndex: number };

const TIME_RANGE_KEYS = ["15min", "1h", "24h", "7d", "15d", "30d"] as const;

export default function ApmPage() {
  const { t } = useI18n();
  const ta = t.apm;
  const { currentClusterId } = useClusterStore();

  // Time range options with i18n labels
  const TIME_RANGE_OPTIONS = useMemo(() => [
    { value: "15min", label: ta.last15min },
    { value: "1h", label: ta.last1h },
    { value: "24h", label: ta.last24h },
    { value: "7d", label: ta.last7d },
    { value: "15d", label: ta.last15d },
    { value: "30d", label: ta.last30d },
  ], [ta]);

  const timeRangeLabels = useMemo(() => {
    const map: Record<string, string> = {};
    for (const opt of TIME_RANGE_OPTIONS) map[opt.value] = opt.label;
    return map;
  }, [TIME_RANGE_OPTIONS]);

  // View navigation state
  const [view, setView] = useState<ViewState>({ level: "services" });

  // Data state
  const [services, setServices] = useState<TraceService[]>([]);
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [traceDetail, setTraceDetail] = useState<TraceDetail | null>(null);
  const [serviceStats, setServiceStats] = useState<ServiceStats[]>([]);
  const [topology, setTopology] = useState<ServiceTopologyData | null>(null);

  // UI state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [timeRange, setTimeRange] = useState("15d");
  const [showTimeDropdown, setShowTimeDropdown] = useState(false);

  // Load data
  const loadData = useCallback(async (showLoading = true) => {
    if (showLoading) setIsRefreshing(true);
    try {
      const [svcData, traceResult] = await Promise.all([
        mockGetTraceServices(),
        mockQueryTraces({ cluster_id: currentClusterId ?? "", limit: 100 }),
      ]);
      setServices(svcData);
      setTraces(traceResult.traces);
      setServiceStats(mockGetAllServiceStats());
      setTopology(mockGetServiceTopology());
      setError(null);
    } catch {
      setError(ta.loadFailed);
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  }, [currentClusterId, ta.loadFailed]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Load trace detail when navigating to Level 3
  useEffect(() => {
    if (view.level !== "trace-detail") {
      setTraceDetail(null);
      return;
    }
    (async () => {
      const detail = await mockGetTraceDetail(view.traceId);
      setTraceDetail(detail);
    })();
  }, [view]);

  // Current service object
  const currentService = useMemo(() => {
    if (view.level === "services") return null;
    return services.find((s) => s.name === view.serviceName) ?? null;
  }, [view, services]);

  // Traces for current service (for navigation in Level 3)
  const serviceTraces = useMemo(() => {
    if (view.level === "services") return [];
    return traces.filter((tr) => tr.rootService === view.serviceName);
  }, [view, traces]);

  // Navigation handlers
  const goToServices = () => setView({ level: "services" });
  const goToService = (name: string) =>
    setView({ level: "service-detail", serviceName: name });
  const goToTrace = (traceId: string) => {
    if (view.level !== "services") {
      const svcTraces = traces.filter((tr) => tr.rootService === view.serviceName);
      const idx = svcTraces.findIndex((tr) => tr.traceId === traceId);
      setView({
        level: "trace-detail",
        serviceName: view.serviceName,
        traceId,
        traceIndex: idx >= 0 ? idx : 0,
      });
    }
  };
  const navigateTrace = (index: number) => {
    if (view.level === "trace-detail" && index >= 0 && index < serviceTraces.length) {
      setView({
        ...view,
        traceId: serviceTraces[index].traceId,
        traceIndex: index,
      });
    }
  };

  // No cluster
  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">{ta.noCluster}</p>
          <p className="text-sm text-muted">{ta.noClusterDesc}</p>
        </div>
      </Layout>
    );
  }

  // Loading
  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  // Error
  if (error && services.length === 0) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <AlertTriangle className="w-12 h-12 mb-4 text-yellow-500" />
          <p className="text-default font-medium mb-2">{error}</p>
          <button
            onClick={() => loadData(true)}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            {t.common.retry}
          </button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-5">
        {/* Header with breadcrumb */}
        <div className="flex items-start justify-between gap-4">
          <div>
            {/* Breadcrumb */}
            <nav className="flex items-center gap-1 text-sm mb-1">
              <button
                onClick={goToServices}
                className={`transition-colors ${
                  view.level === "services"
                    ? "text-default font-semibold"
                    : "text-primary hover:text-primary/80"
                }`}
              >
                {ta.pageTitle}
              </button>

              {view.level !== "services" && (
                <>
                  <ChevronRight className="w-4 h-4 text-muted" />
                  <button
                    onClick={() => goToService(view.serviceName)}
                    className={`transition-colors ${
                      view.level === "service-detail"
                        ? "text-default font-semibold"
                        : "text-primary hover:text-primary/80"
                    }`}
                  >
                    {view.serviceName}
                  </button>
                </>
              )}

              {view.level === "trace-detail" && (
                <>
                  <ChevronRight className="w-4 h-4 text-muted" />
                  <span className="text-default font-semibold font-mono text-xs">
                    {view.traceId.slice(0, 12)}...
                  </span>
                </>
              )}
            </nav>

            <p className="text-xs text-muted">{ta.pageDescription}</p>
          </div>

          <div className="flex items-center gap-2">
            {/* Time range selector */}
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
              onClick={() => loadData(true)}
              disabled={isRefreshing}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-50 transition-colors"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? "animate-spin" : ""}`} />
              {t.common.refresh}
            </button>
          </div>
        </div>

        {/* View content */}
        {view.level === "services" && (
          <>
            {topology && topology.nodes.length > 0 && (
              <ServiceTopology t={ta} topology={topology} onSelectService={goToService} />
            )}
            <ServiceList
              t={ta}
              tt={t.table}
              serviceStats={serviceStats}
              traces={traces}
              onSelectService={goToService}
            />
          </>
        )}

        {view.level === "service-detail" && currentService && (
          <ServiceOverview
            t={ta}
            service={currentService}
            traces={traces}
            onSelectTrace={goToTrace}
            onNavigateToService={goToService}
          />
        )}

        {view.level === "trace-detail" && traceDetail && (
          <TraceWaterfall
            t={ta}
            trace={traceDetail}
            allTraces={serviceTraces}
            currentTraceIndex={view.traceIndex}
            onNavigateTrace={navigateTrace}
          />
        )}

        {view.level === "trace-detail" && !traceDetail && (
          <div className="flex items-center justify-center h-48">
            <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
          </div>
        )}
      </div>
    </Layout>
  );
}
