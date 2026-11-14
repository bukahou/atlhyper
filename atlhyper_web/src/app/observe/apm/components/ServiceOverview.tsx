"use client";

import { useMemo, useState } from "react";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { OperationStats, Topology } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { QueryView } from "@/components/common";
import { getDependenciesFromTopology } from "@/datasource/apm";
import { formatDurationMs } from "@/lib/format";
import type { TimeRangeParams } from "@/lib/time-range";
import { apmDbStatsQuery, apmServiceSeriesQuery, apmTracesQuery } from "@/queries/observe-apm";
import { TransactionsTable } from "./TransactionsTable";
import { DependenciesTable } from "./DependenciesTable";
import { ServiceTrendCharts } from "./ServiceTrendCharts";
import { ErrorTracesList } from "./ErrorTracesList";
import { CorrelationsPanel } from "./CorrelationsPanel";
import { StatusCodeSection } from "./StatusCodeSection";
import { DBStatsTable } from "./DBStatsTable";
import { SlowTracesList } from "./SlowTracesList";
import { PermissionLock } from "@/components/common/PermissionLock";

/** 取该服务最近多少条链路，用来挑出错的与最慢的 */
const SERVICE_TRACE_LIMIT = 200;
/** 「慢链路」列表显示最慢的多少条 */
const SLOW_TRACE_COUNT = 20;

interface ServiceOverviewProps {
  t: ApmTranslations;
  serviceName: string;
  operations: OperationStats[];
  /** 服务拓扑的查询：依赖列表从它派生，取不到时依赖列表处显示出错而不是「没有依赖」 */
  topologyQuery: UseQueryResult<Topology>;
  clusterId: string;
  timeRange: TimeRangeParams;
  /** 是否有 traces.read；链路列表与相关性是内容级数据，没有权限就不发请求 */
  canReadTraces: boolean;
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
  topologyQuery,
  clusterId,
  timeRange,
  canReadTraces,
  onSelectOperation,
  onNavigateToService,
  onSelectTrace,
}: ServiceOverviewProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");

  // 这个服务自己的数据：趋势、数据库统计、最近的链路。各是一个查询，服务或时间范围变了各自重取
  const trendQuery = useQuery(apmServiceSeriesQuery(clusterId, serviceName, timeRange));
  const dbStatsQuery = useQuery(apmDbStatsQuery(clusterId, serviceName, timeRange));
  const serviceTracesQuery = useQuery(
    apmTracesQuery(clusterId, { service: serviceName, limit: SERVICE_TRACE_LIMIT }, timeRange, canReadTraces),
  );
  const serviceTraces = serviceTracesQuery.data?.traces;

  const serviceOperations = useMemo(
    () => operations.filter((op) => op.serviceName === serviceName),
    [operations, serviceName]
  );

  const metrics = useMemo(() => aggregateMetrics(serviceOperations), [serviceOperations]);

  const errorTraces = useMemo(
    () => (serviceTraces ?? []).filter((tr) => tr.hasError),
    [serviceTraces]
  );

  const slowTraces = useMemo(
    () => [...(serviceTraces ?? [])].sort((a, b) => b.durationMs - a.durationMs).slice(0, SLOW_TRACE_COUNT),
    [serviceTraces]
  );

  // 依赖列表在「概览」与「依赖」两个 tab 里各出现一次
  const dependenciesTable = (
    <QueryView query={topologyQuery}>
      {(topology) => (
        <DependenciesTable
          t={t}
          dependencies={getDependenciesFromTopology(serviceName, topology)}
          onSelectDependency={onNavigateToService}
        />
      )}
    </QueryView>
  );

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
          <QueryView query={trendQuery}>
            {(points) => <ServiceTrendCharts t={t} points={points} />}
          </QueryView>

          {/* HTTP Status Code Distribution */}
          <StatusCodeSection
            t={t}
            clusterId={clusterId}
            serviceName={serviceName}
            timeRange={timeRange}
            canReadTraces={canReadTraces}
            onSelectTrace={onSelectTrace}
          />

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
            {dependenciesTable}
          </div>

          {/* DB Stats */}
          <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
            <QueryView query={dbStatsQuery}>
              {(stats) => <DBStatsTable t={t} stats={stats} />}
            </QueryView>
          </div>

          {/* Slow Traces */}
          <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
            <PermissionLock permission="traces.read">
              <QueryView query={serviceTracesQuery}>
                {() => <SlowTracesList t={t} traces={slowTraces} onSelectTrace={onSelectTrace} />}
              </QueryView>
            </PermissionLock>
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
          {dependenciesTable}
        </div>
      )}

      {/* Errors Tab */}
      {activeTab === "errors" && (
        <div className="space-y-4">
          <PermissionLock permission="traces.read" className="border border-[var(--border-color)] rounded-xl bg-card">
            <QueryView query={serviceTracesQuery}>
              {() => <ErrorTracesList t={t} traces={errorTraces} onSelectTrace={onSelectTrace} />}
            </QueryView>
            <CorrelationsPanel
              t={t}
              clusterId={clusterId}
              serviceName={serviceName}
              timeRange={timeRange}
              canReadTraces={canReadTraces}
            />
          </PermissionLock>
        </div>
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
