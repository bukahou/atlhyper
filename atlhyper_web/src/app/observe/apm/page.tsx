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
} from "lucide-react";

import type { TraceSummary, TraceDetail, APMService, Topology, OperationStats } from "@/types/model/apm";
import {
  getAPMServices,
  queryTraces,
  getTraceDetail,
  getTopology,
  getOperations,
} from "@/datasource/apm";

import { ServiceList } from "./components/ServiceList";
import { ServiceOverview } from "./components/ServiceOverview";
import { TraceWaterfall } from "./components/TraceWaterfall";
import { ServiceTopology } from "./components/ServiceTopology";

type ViewState =
  | { level: "services" }
  | { level: "service-detail"; serviceName: string }
  | { level: "trace-detail"; serviceName: string; traceId: string; traceIndex: number };

export default function ApmPage() {
  const { t } = useI18n();
  const ta = t.apm;
  const { currentClusterId } = useClusterStore();

  const [view, setView] = useState<ViewState>({ level: "services" });
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [traceDetail, setTraceDetail] = useState<TraceDetail | null>(null);
  const [serviceStats, setServiceStats] = useState<APMService[]>([]);
  const [topology, setTopology] = useState<Topology | null>(null);
  const [operations, setOperations] = useState<OperationStats[]>([]);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const loadData = useCallback(async (showLoading = true) => {
    if (showLoading) setIsRefreshing(true);
    try {
      const [traceResult, svcStats, topo, ops] = await Promise.all([
        queryTraces(currentClusterId, { limit: 500 }),
        getAPMServices(currentClusterId),
        getTopology(currentClusterId),
        getOperations(currentClusterId),
      ]);
      setTraces(traceResult.traces);
      setServiceStats(svcStats);
      setTopology(topo);
      setOperations(ops);
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

  useEffect(() => {
    if (view.level !== "trace-detail") {
      setTraceDetail(null);
      return;
    }
    getTraceDetail(view.traceId, currentClusterId).then(setTraceDetail);
  }, [view, currentClusterId]);

  const serviceTraces = useMemo(() => {
    if (view.level === "services") return [];
    return traces.filter((tr) => tr.rootService === view.serviceName);
  }, [view, traces]);

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

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  if (error && serviceStats.length === 0) {
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

          <button
            onClick={() => loadData(true)}
            disabled={isRefreshing}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? "animate-spin" : ""}`} />
            {t.common.refresh}
          </button>
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
              onSelectService={goToService}
            />
          </>
        )}

        {view.level === "service-detail" && (
          <ServiceOverview
            t={ta}
            serviceName={view.serviceName}
            traces={traces}
            operations={operations}
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
