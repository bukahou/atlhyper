"use client";

import { useMemo, useState, useEffect } from "react";
import type { OperationStats, Topology, APMTimePoint, TraceSummary, HTTPStats, DBOperationStats } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { getDependenciesFromTopology, getServiceTimeSeries, getHTTPStats, getDBStats, queryTraces, type TimeParams } from "@/datasource/apm";
import { formatDurationMs } from "@/lib/format";
import { TransactionsTable } from "./TransactionsTable";
import { DependenciesTable } from "./DependenciesTable";
import { ServiceTrendCharts } from "./ServiceTrendCharts";
import { ErrorTracesList } from "./ErrorTracesList";
import { StatusCodeChart } from "./StatusCodeChart";
import { DBStatsTable } from "./DBStatsTable";
import { SlowTracesList } from "./SlowTracesList";

interface ServiceOverviewProps {
  t: ApmTranslations;
  serviceName: string;
  operations: OperationStats[];
  topology: Topology | null;
  clusterId?: string;
  timeParams?: TimeParams;
  onSelectOperation: (operation: string) => void;
  onNavigateToService?: (serviceName: string) => void;
  onSelectTrace?: (traceId: string) => void;
}

const TABS = ["overview", "transactions", "dependencies", "errors"] as const;
type TabType = (typeof TABS)[number];

/** 从 operations 聚合计算服务级指标 */
function aggregateMetrics(ops: OperationStats[]) {
  if (ops.length === 0) {
    return { totalRequests: 0, avgLatencyMs: 0, p50Ms: 0, p99Ms: 0, tpm: 0, errorRate: 0 };
  }
  const totalRequests = ops.reduce((s, o) => s + o.spanCount, 0);
  const totalRps = ops.reduce((s, o) => s + o.rps, 0);
  const totalErrors = ops.reduce((s, o) => s + o.errorCount, 0);
  const avgLatencyMs = totalRequests > 0
    ? ops.reduce((s, o) => s + o.avgDurationMs * o.spanCount, 0) / totalRequests
    : 0;
  const p50Ms = totalRequests > 0
    ? ops.reduce((s, o) => s + o.p50Ms * o.spanCount, 0) / totalRequests
    : 0;
  const p99Ms = Math.max(...ops.map((o) => o.p99Ms), 0);
  const errorRate = totalRequests > 0 ? (totalErrors / totalRequests) * 100 : 0;
  return { totalRequests, avgLatencyMs, p50Ms, p99Ms, tpm: totalRps * 60, errorRate };
}

export function ServiceOverview({
  t,
  serviceName,
  operations,
  topology,
  clusterId,
  timeParams,
  onSelectOperation,
  onNavigateToService,
  onSelectTrace,
}: ServiceOverviewProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [trendPoints, setTrendPoints] = useState<APMTimePoint[]>([]);
  const [serviceTraces, setServiceTraces] = useState<TraceSummary[]>([]);
  const [httpStats, setHttpStats] = useState<HTTPStats[]>([]);
  const [dbStats, setDbStats] = useState<DBOperationStats[]>([]);

  const serviceOperations = useMemo(
    () => operations.filter((op) => op.serviceName === serviceName),
    [operations, serviceName]
  );

  const metrics = useMemo(() => aggregateMetrics(serviceOperations), [serviceOperations]);

  const dependencies = useMemo(
    () => getDependenciesFromTopology(serviceName, topology),
    [serviceName, topology]
  );

  const errorTraces = useMemo(
    () => serviceTraces.filter((tr) => tr.hasError),
    [serviceTraces]
  );

  const slowTraces = useMemo(
    () => [...serviceTraces].sort((a, b) => b.durationMs - a.durationMs).slice(0, 20),
    [serviceTraces]
  );

  // Load trend data + service traces + HTTP stats + DB stats
  useEffect(() => {
    if (!clusterId) return;
    getServiceTimeSeries(clusterId, serviceName, timeParams?.since || "15m").then(setTrendPoints);
    queryTraces(clusterId, { service: serviceName, limit: 200 }, timeParams).then((r) => setServiceTraces(r.traces));
    getHTTPStats(clusterId, serviceName, timeParams).then(setHttpStats);
    getDBStats(clusterId, serviceName, timeParams).then(setDbStats);
  }, [clusterId, serviceName, timeParams]);

  const tabLabels: Record<TabType, string> = {
    overview: t.overview,
    transactions: t.transactions,
    dependencies: t.dependencies,
    errors: `${t.errors}${errorTraces.length > 0 ? ` (${errorTraces.length})` : ""}`,
  };

  return (
    <div className="space-y-6">
      {/* Tabs */}
      <div className="flex gap-0 border-b border-[var(--border-color)]">
        {TABS.map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              tab === activeTab
                ? "text-primary border-primary"
                : "text-muted hover:text-default border-transparent"
            }`}
          >
            {tabLabels[tab]}
          </button>
        ))}
      </div>

      {/* Overview Tab */}
      {activeTab === "overview" && (
        <>
          {/* Service-level summary metrics */}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
            <MetricCard label={t.totalRequests} value={metrics.totalRequests.toLocaleString()} />
            <MetricCard label={t.latencyAvg} value={formatDurationMs(metrics.avgLatencyMs)} />
            <MetricCard label="P50" value={formatDurationMs(metrics.p50Ms)} />
            <MetricCard label="P99" value={formatDurationMs(metrics.p99Ms)} />
            <MetricCard label={t.throughput} value={`${metrics.tpm.toFixed(1)} tpm`} />
            <MetricCard
              label={t.errorRate}
              value={`${metrics.errorRate.toFixed(2)}%`}
              variant={metrics.errorRate > 5 ? "danger" : metrics.errorRate > 1 ? "warning" : "success"}
            />
          </div>

          {/* Trend charts */}
          <ServiceTrendCharts t={t} points={trendPoints} />

          {/* HTTP Status Code Distribution */}
          <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
            <StatusCodeChart t={t} stats={httpStats} />
          </div>

          {/* Transactions table */}
          <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
            <TransactionsTable
              t={t}
              operations={serviceOperations}
              onSelectOperation={onSelectOperation}
            />
          </div>

          {/* Dependencies */}
          <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
            <DependenciesTable
              t={t}
              dependencies={dependencies}
              onSelectDependency={onNavigateToService}
            />
          </div>

          {/* DB Stats */}
          <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
            <DBStatsTable t={t} stats={dbStats} />
          </div>

          {/* Slow Traces */}
          <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
            <SlowTracesList t={t} traces={slowTraces} onSelectTrace={onSelectTrace} />
          </div>
        </>
      )}

      {/* Transactions Tab */}
      {activeTab === "transactions" && (
        <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
          <TransactionsTable
            t={t}
            operations={serviceOperations}
            onSelectOperation={onSelectOperation}
          />
        </div>
      )}

      {/* Dependencies Tab */}
      {activeTab === "dependencies" && (
        <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
          <DependenciesTable
            t={t}
            dependencies={dependencies}
            onSelectDependency={onNavigateToService}
          />
        </div>
      )}

      {/* Errors Tab */}
      {activeTab === "errors" && (
        <ErrorTracesList t={t} traces={errorTraces} onSelectTrace={onSelectTrace} />
      )}
    </div>
  );
}

function MetricCard({ label, value, variant }: { label: string; value: string; variant?: "success" | "warning" | "danger" }) {
  const colorClass = variant === "danger"
    ? "text-red-500"
    : variant === "warning"
      ? "text-orange-500"
      : variant === "success"
        ? "text-emerald-500"
        : "text-default";
  return (
    <div className="border border-[var(--border-color)] rounded-xl px-4 py-3 bg-card">
      <div className="text-[10px] text-muted uppercase tracking-wider mb-1">{label}</div>
      <div className={`text-lg font-semibold ${colorClass}`}>{value}</div>
    </div>
  );
}
