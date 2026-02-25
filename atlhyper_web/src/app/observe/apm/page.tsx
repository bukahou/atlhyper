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
  ChevronRight,
  Clock,
} from "lucide-react";

import type { TraceSummary, TraceDetail, APMService, Topology, OperationStats, Span } from "@/types/model/apm";
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
  | { level: "trace-detail"; serviceName: string; operationName: string; traceId: string; traceIndex: number };

export default function ApmPage() {
  const { t } = useI18n();
  const ta = t.apm;
  const { currentClusterId } = useClusterStore();

  const [view, setView] = useState<ViewState>({ level: "services" });
  const [traceDetail, setTraceDetail] = useState<TraceDetail | null>(null);
  const [serviceStats, setServiceStats] = useState<APMService[]>([]);
  const [topology, setTopology] = useState<Topology | null>(null);
  const [operations, setOperations] = useState<OperationStats[]>([]);
  const [operationTraces, setOperationTraces] = useState<TraceSummary[]>([]);
  const [timeRange, setTimeRange] = useState("15m");

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const timeRangeOptions = useMemo(() => [
    { value: "15m", label: ta.last15min },
    { value: "1h", label: ta.last1h },
    { value: "6h", label: ta.last6h },
    { value: "24h", label: ta.last24h },
    { value: "168h", label: ta.last7d },
    { value: "360h", label: ta.last15d },
    { value: "720h", label: ta.last30d },
  ], [ta]);

  const loadData = useCallback(async (showLoading = true) => {
    if (showLoading) setIsRefreshing(true);
    try {
      const tr = timeRange === "15m" ? undefined : timeRange;
      const [svcStats, topo, ops] = await Promise.all([
        getAPMServices(currentClusterId, tr),
        getTopology(currentClusterId, tr),
        getOperations(currentClusterId, tr),
      ]);
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
  }, [currentClusterId, timeRange, ta.loadFailed]);

  // 静默刷新（自动刷新用，不显示 loading 状态）
  const loadDataSilent = useCallback(() => {
    loadData(false);
  }, [loadData]);

  // 服务列表 / 服务详情自动刷新，trace 瀑布图禁用（历史数据，刷新可能丢失）
  useAutoRefresh(loadDataSilent, {
    enabled: view.level !== "trace-detail",
  });

  useEffect(() => {
    if (view.level !== "trace-detail") {
      setTraceDetail(null);
      return;
    }
    getTraceDetail(view.traceId, currentClusterId).then(setTraceDetail);
  }, [view, currentClusterId]);

  const goToServices = () => setView({ level: "services" });
  const goToService = (name: string) =>
    setView({ level: "service-detail", serviceName: name });

  // 点击事务行：查该 operation 的 traces，然后进入第三层
  const goToTraceForOperation = async (serviceName: string, operation: string) => {
    const tr = timeRange === "15m" ? undefined : timeRange;
    const result = await queryTraces(currentClusterId, {
      service: serviceName, operation, limit: 200,
    }, tr);
    if (result.traces.length > 0) {
      setOperationTraces(result.traces);
      setView({
        level: "trace-detail",
        serviceName,
        operationName: operation,
        traceId: result.traces[0].traceId,
        traceIndex: 0,
      });
    }
  };

  const navigateTrace = (index: number) => {
    if (view.level === "trace-detail" && index >= 0 && index < operationTraces.length) {
      setView({
        ...view,
        traceId: operationTraces[index].traceId,
        traceIndex: index,
      });
    }
  };

  // 从子服务进入追踪详情时，裁剪 Span 树：只保留该服务的入口 Span 及其所有后代
  const focusedTrace = useMemo(() => {
    if (!traceDetail || view.level !== "trace-detail") return null;
    return filterTraceForService(traceDetail, view.serviceName);
  }, [traceDetail, view]);

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
                  <span className="text-default text-xs truncate max-w-[200px]">
                    {view.operationName}
                  </span>
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
            <div className="relative">
              <Clock className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted pointer-events-none" />
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(e.target.value)}
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

        {/* View content */}
        {view.level === "services" && (
          <>
            <ServiceTopology t={ta} topology={topology ?? { nodes: [], edges: [] }} onSelectService={goToService} />
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
            operations={operations}
            topology={topology}
            clusterId={currentClusterId}
            timeRange={timeRange}
            onSelectOperation={(op) => goToTraceForOperation(view.serviceName, op)}
            onNavigateToService={goToService}
            onSelectTrace={(traceId) => {
              setView({
                level: "trace-detail",
                serviceName: view.serviceName,
                operationName: "",
                traceId,
                traceIndex: 0,
              });
            }}
          />
        )}

        {view.level === "trace-detail" && focusedTrace && (
          <TraceWaterfall
            t={ta}
            trace={focusedTrace}
            allTraces={operationTraces}
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

/**
 * 按聚焦服务裁剪 Trace：只保留该服务的入口 Span 及其所有后代。
 * 上层调用者（如网关）被过滤掉，入口 Span 成为新的根节点。
 */
function filterTraceForService(trace: TraceDetail, focusService: string): TraceDetail {
  const { spans } = trace;
  if (spans.length === 0) return trace;

  // 构建查找表
  const spanMap = new Map<string, Span>();
  const childrenMap = new Map<string, string[]>();
  for (const span of spans) {
    spanMap.set(span.spanId, span);
    if (span.parentSpanId) {
      const list = childrenMap.get(span.parentSpanId) ?? [];
      list.push(span.spanId);
      childrenMap.set(span.parentSpanId, list);
    }
  }

  // 找到聚焦服务的入口 Span：自身 serviceName 匹配，但父 Span 的 serviceName 不匹配（或无父）
  const entryIds: string[] = [];
  for (const span of spans) {
    if (span.serviceName !== focusService) continue;
    const parent = span.parentSpanId ? spanMap.get(span.parentSpanId) : undefined;
    if (!parent || parent.serviceName !== focusService) {
      entryIds.push(span.spanId);
    }
  }

  // 无匹配时回退显示完整 Trace
  if (entryIds.length === 0) return trace;

  // 收集入口 Span 及其所有后代
  const included = new Set<string>();
  const collect = (id: string) => {
    included.add(id);
    for (const childId of childrenMap.get(id) ?? []) collect(childId);
  };
  entryIds.forEach(collect);

  // 过滤 + 将入口 Span 的 parentSpanId 清空（使其成为根）
  const entrySet = new Set(entryIds);
  const filtered = spans
    .filter((s) => included.has(s.spanId))
    .map((s) => (entrySet.has(s.spanId) ? { ...s, parentSpanId: "" } : s));

  return {
    traceId: trace.traceId,
    spans: filtered,
    spanCount: filtered.length,
    serviceCount: new Set(filtered.map((s) => s.serviceName)).size,
    durationMs: trace.durationMs,
  };
}
