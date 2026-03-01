"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import {
  RefreshCw,
  Loader2,
  WifiOff,
  AlertTriangle,
  Clock,
  ChevronRight,
  Activity,
} from "lucide-react";
import { getObserveHealth } from "@/datasource/observe-health";
import type { LandingPageResponse, HealthOverview } from "@/types/model/observe";
import type { ObserveLandingTranslations } from "@/types/i18n";
import { ServiceTable } from "./components/ServiceTable";
import { ServiceDetail } from "./components/ServiceDetail";
import { SectionDetail, sectionTitle } from "./components/SectionDetail";

type TimeRange = "15m" | "1d" | "7d" | "30d";
type DetailSection = "k8s" | "apm" | "slo" | "logs" | "infra";

type ViewState =
  | { level: "services" }
  | { level: "service-detail"; serviceName: string }
  | { level: "section-detail"; serviceName: string; section: DetailSection };

export default function ObserveLandingPage() {
  const { t } = useI18n();
  const tl = t.observeLanding;
  const { currentClusterId } = useClusterStore();

  const [data, setData] = useState<LandingPageResponse | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>("15m");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [view, setView] = useState<ViewState>({ level: "services" });

  const timeRangeOptions = useMemo(() => [
    { value: "15m" as TimeRange, label: tl.timeRange15m },
    { value: "1d" as TimeRange, label: tl.timeRange1d },
    { value: "7d" as TimeRange, label: tl.timeRange7d },
    { value: "30d" as TimeRange, label: tl.timeRange30d },
  ], [tl]);

  const loadData = useCallback(async (showLoading = true) => {
    if (showLoading) setIsRefreshing(true);
    try {
      const result = await getObserveHealth(currentClusterId, timeRange);
      setData(result);
      setError(null);
    } catch {
      setError(tl.loadFailed);
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  }, [currentClusterId, timeRange, tl.loadFailed]);

  const loadDataSilent = useCallback(() => { loadData(false); }, [loadData]);
  useAutoRefresh(loadDataSilent);

  useEffect(() => { loadData(true); }, [loadData]);

  // Navigation
  const goServices = useCallback(() => setView({ level: "services" }), []);
  const goServiceDetail = useCallback((name: string) => setView({ level: "service-detail", serviceName: name }), []);
  const goSectionDetail = useCallback((name: string, section: DetailSection) => setView({ level: "section-detail", serviceName: name, section }), []);

  // Guard states
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

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  if (error && !data) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <AlertTriangle className="w-12 h-12 mb-4 text-yellow-500" />
          <p className="text-default font-medium mb-2">{error}</p>
          <button
            onClick={() => loadData(true)}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            {t.common.refresh}
          </button>
        </div>
      </Layout>
    );
  }

  const overview = data?.overview;
  const services = data?.services ?? [];
  const selectedService = view.level !== "services"
    ? services.find((s) => s.name === view.serviceName)
    : undefined;

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-5">
        {/* Header: breadcrumb + controls */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <Breadcrumb view={view} tl={tl} goServices={goServices} goServiceDetail={goServiceDetail} />
            {view.level === "services" && (
              <p className="text-xs text-muted mt-0.5">{tl.description}</p>
            )}
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Clock className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted pointer-events-none" />
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(e.target.value as TimeRange)}
                className="appearance-none pl-8 pr-7 py-1.5 text-sm rounded-lg border border-default bg-secondary text-default cursor-pointer hover:border-primary/50 focus:outline-none focus:ring-1 focus:ring-primary/50 transition-colors"
              >
                {timeRangeOptions.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
              <ChevronRight className="absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted rotate-90 pointer-events-none" />
            </div>
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

        {/* Content by level */}
        {view.level === "services" && (
          <>
            {overview && <OverviewCards overview={overview} tl={tl} />}
            {services.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16 text-center">
                <Activity className="w-10 h-10 mb-3 text-muted" />
                <p className="text-sm text-muted">{tl.noServices}</p>
              </div>
            ) : (
              <ServiceTable services={services} tl={tl} onSelectService={goServiceDetail} />
            )}
          </>
        )}

        {view.level === "service-detail" && selectedService && (
          <ServiceDetail
            service={selectedService}
            tl={tl}
            totalLabel={t.common.total}
            onDrillDown={(section) => goSectionDetail(selectedService.name, section)}
          />
        )}

        {view.level === "section-detail" && selectedService && (
          <SectionDetail
            service={selectedService}
            section={view.section}
            tl={tl}
            totalLabel={t.common.total}
          />
        )}
      </div>
    </Layout>
  );
}

// ============================================================================
// Breadcrumb
// ============================================================================

function Breadcrumb({
  view,
  tl,
  goServices,
  goServiceDetail,
}: {
  view: ViewState;
  tl: ObserveLandingTranslations;
  goServices: () => void;
  goServiceDetail: (name: string) => void;
}) {
  if (view.level === "services") {
    return <h1 className="text-lg font-semibold text-default">{tl.title}</h1>;
  }

  return (
    <div className="flex items-center gap-1.5 text-sm">
      <button onClick={goServices} className="text-primary hover:text-primary/80 transition-colors">
        {tl.backToOverview}
      </button>
      <ChevronRight className="w-3.5 h-3.5 text-muted" />
      {view.level === "service-detail" ? (
        <span className="font-semibold text-default">{view.serviceName}</span>
      ) : (
        <>
          <button
            onClick={() => goServiceDetail(view.serviceName)}
            className="text-primary hover:text-primary/80 transition-colors"
          >
            {view.serviceName}
          </button>
          <ChevronRight className="w-3.5 h-3.5 text-muted" />
          <span className="font-semibold text-default">{sectionTitle(view.section, tl)}</span>
        </>
      )}
    </div>
  );
}

// ============================================================================
// Overview Cards (Level 1)
// ============================================================================

function OverviewCards({ overview, tl }: { overview: HealthOverview; tl: ObserveLandingTranslations }) {
  const cards = [
    {
      label: tl.totalServices,
      value: overview.totalServices,
      sub: `${overview.healthyServices}/${overview.warningServices}/${overview.criticalServices}`,
      color: overview.criticalServices > 0 ? "text-red-500" : overview.warningServices > 0 ? "text-yellow-500" : "text-green-500",
    },
    {
      label: tl.totalRps,
      value: overview.totalRps.toFixed(1),
      color: "text-blue-500",
    },
    {
      label: tl.sloCompliance,
      value: `${(overview.sloCompliance * 100).toFixed(1)}%`,
      color: overview.sloCompliance >= 0.99 ? "text-green-500" : overview.sloCompliance >= 0.95 ? "text-yellow-500" : "text-red-500",
    },
    {
      label: tl.totalErrorCount,
      value: overview.totalErrorCount.toLocaleString(),
      color: overview.totalErrorCount > 100 ? "text-red-500" : overview.totalErrorCount > 0 ? "text-yellow-500" : "text-green-500",
    },
    {
      label: tl.nodeStatus,
      value: `${overview.onlineNodes}/${overview.totalNodes}`,
      color: "text-cyan-500",
    },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
      {cards.map((card) => (
        <div key={card.label} className="p-3 rounded-xl border border-[var(--border-color)] bg-card">
          <p className="text-xs text-muted mb-1">{card.label}</p>
          <p className={`text-xl font-bold ${card.color}`}>{card.value}</p>
          {card.sub && <p className="text-[11px] text-muted mt-1 truncate">{card.sub}</p>}
        </div>
      ))}
    </div>
  );
}
