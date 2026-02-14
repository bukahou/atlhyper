"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Globe,
  ChevronDown,
  ChevronRight,
  Settings2,
  Activity,
  Network,
  Calendar,
  BarChart3,
} from "lucide-react";
import { StatusBadge, ErrorBudgetBar, TrendIcon, formatNumber, formatLatency } from "./common";
import { SLOTargetModal } from "./SLOTargetModal";
import { OverviewTab } from "./OverviewTab";
import { MeshTab } from "./MeshTab";
import { CompareTab } from "./CompareTab";
import { LatencyTab } from "./LatencyTab";
import { getSLODomainHistory, getSLOLatencyDistribution } from "@/api/slo";
import { getMeshTopology } from "@/api/mesh";
import type { DomainSLOV2, LatencyDistributionResponse } from "@/types/slo";
import type { MeshTopologyResponse } from "@/types/mesh";

type TimeRange = "1d" | "7d" | "30d";

export interface DomainCardTranslations {
  services: string;
  availability: string;
  p95Latency: string;
  p99Latency: string;
  errorRate: string;
  errorBudget: string;
  throughput: string;
  // Tabs
  tabOverview: string;
  tabMesh: string;
  tabCompare: string;
  tabLatency: string;
  configTarget: string;
  // Overview tab
  totalRequests: string;
  target: string;
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  // Mesh tab
  serviceTopology: string;
  meshOverview: string;
  service: string;
  rps: string;
  mtls: string;
  status: string;
  healthy: string;
  warning: string;
  critical: string;
  unknown: string;
  inbound: string;
  outbound: string;
  noCallData: string;
  callRelation: string;
  p50Latency: string;
  avgLatency: string;
  // Compare tab
  currentVsPrevious: string;
  previousPeriod: string;
  // Modal
  configSloTarget: string;
  targetDomain: string;
  selectPeriod: string;
  day: string;
  week: string;
  month: string;
  targetAvailability: string;
  targetAvailabilityHint: string;
  targetP95: string;
  targetP95Hint: string;
  errorRateThreshold: string;
  errorRateAutoCalc: string;
  cancel: string;
  save: string;
  saving: string;
  estimatedExhaust: string;
  // Mesh detail + Latency tab (shared)
  statusCodeBreakdown: string;
  latencyDistribution: string;
  requests: string;
  loading: string;
  methodBreakdown: string;
  clearSelection: string;
}

export function DomainCard({ domain, expanded, onToggle, timeRange, clusterId, onRefresh, t }: {
  domain: DomainSLOV2;
  expanded: boolean;
  onToggle: () => void;
  timeRange: TimeRange;
  clusterId: string;
  onRefresh: () => void;
  t: DomainCardTranslations;
}) {
  const [activeTab, setActiveTab] = useState<"overview" | "mesh" | "compare" | "latency">("overview");
  const [showTargetModal, setShowTargetModal] = useState(false);
  const [meshTopology, setMeshTopology] = useState<MeshTopologyResponse | null>(null);
  const [history, setHistory] = useState<{ timestamp: string; p95Latency: number; p99Latency: number; errorRate: number; availability: number; rps: number }[]>([]);
  const [latencyData, setLatencyData] = useState<LatencyDistributionResponse | null>(null);

  const availability = domain.summary?.availability ?? 0;
  const p95Latency = domain.summary?.p95Latency ?? 0;
  const errorRate = domain.summary?.errorRate ?? 0;
  const rps = domain.summary?.requestsPerSec ?? 0;

  // Compute previous period from services
  const prevAvailability = domain.services.length > 0
    ? domain.services.reduce((sum, s) => sum + (s.previous?.availability ?? s.current?.availability ?? 0), 0) / domain.services.length
    : availability;
  const prevP95Latency = domain.services.length > 0
    ? domain.services.reduce((sum, s) => sum + (s.previous?.p95Latency ?? s.current?.p95Latency ?? 0), 0) / domain.services.length
    : p95Latency;
  const prevErrorRate = domain.services.length > 0
    ? domain.services.reduce((sum, s) => sum + (s.previous?.errorRate ?? s.current?.errorRate ?? 0), 0) / domain.services.length
    : errorRate;

  const trend = availability > prevAvailability ? "up" : availability < prevAvailability ? "down" : "stable";
  const domainTargets = domain.targets?.[timeRange] || domain.targets?.["1d"] || { availability: 95, p95Latency: 300 };
  const targets = { availability: domainTargets.availability, p95Latency: domainTargets.p95Latency };

  const statusLabels = { healthy: t.healthy, warning: t.warning, critical: t.critical, unknown: t.unknown };

  // Reset cached data when timeRange changes
  useEffect(() => {
    setMeshTopology(null);
    setLatencyData(null);
    setHistory([]);
  }, [timeRange]);

  // Lazy-load mesh topology when mesh tab is opened, filtered to this domain's namespaces
  const loadMeshData = useCallback(async () => {
    if (meshTopology) return;
    try {
      const res = await getMeshTopology({ clusterId, timeRange });
      if (res.data) {
        // Filter to only show services in this domain's namespaces
        const domainNamespaces = new Set(domain.services.map(s => s.namespace));
        if (domainNamespaces.size > 0) {
          const filteredNodes = res.data.nodes.filter(n => domainNamespaces.has(n.namespace));
          const nodeIds = new Set(filteredNodes.map(n => n.id));
          const filteredEdges = res.data.edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));
          setMeshTopology({ nodes: filteredNodes, edges: filteredEdges });
        } else {
          setMeshTopology(res.data);
        }
      }
    } catch (err) {
      console.warn("[SLO] Mesh topology load failed:", err);
    }
  }, [clusterId, timeRange, meshTopology, domain.services]);

  // Lazy-load latency data when latency tab is opened
  const loadLatencyData = useCallback(async () => {
    if (latencyData) return;
    try {
      const res = await getSLOLatencyDistribution({ clusterId, domain: domain.domain, timeRange });
      if (res.data) setLatencyData(res.data);
    } catch {
      // Latency data may not be available yet
    }
  }, [clusterId, domain.domain, timeRange, latencyData]);

  // Lazy-load history when overview tab is opened
  const loadHistory = useCallback(async () => {
    if (history.length > 0) return;
    try {
      const res = await getSLODomainHistory({ clusterId, host: domain.domain, timeRange });
      if (res.data?.history) setHistory(res.data.history);
    } catch (err) {
      console.warn("[SLO] History fetch error:", err);
    }
  }, [clusterId, domain.domain, timeRange, history.length]);

  useEffect(() => {
    if (!expanded) return;
    if (activeTab === "mesh") loadMeshData();
    if (activeTab === "overview") loadHistory();
    if (activeTab === "latency") loadLatencyData();
  }, [expanded, activeTab, loadMeshData, loadHistory, loadLatencyData]);

  return (
    <div className="border border-[var(--border-color)] rounded-xl overflow-hidden bg-card">
      {/* Summary Row */}
      <button onClick={onToggle} className="w-full px-3 sm:px-4 py-3 flex flex-col lg:flex-row lg:items-center gap-2 lg:gap-4 hover:bg-[var(--hover-bg)] transition-colors">
        {/* Domain Info */}
        <div className="flex items-center gap-2 sm:gap-3 flex-1 min-w-0">
          <div className={`p-1.5 sm:p-2 rounded-lg flex-shrink-0 ${
            domain.status === "healthy" ? "bg-emerald-500/10" :
            domain.status === "warning" ? "bg-amber-500/10" : "bg-red-500/10"
          }`}>
            <Globe className={`w-4 h-4 ${
              domain.status === "healthy" ? "text-emerald-500" :
              domain.status === "warning" ? "text-amber-500" : "text-red-500"
            }`} />
          </div>
          <div className="text-left min-w-0 flex-1">
            <div className="flex items-center gap-1.5 sm:gap-2 flex-wrap">
              {domain.tls && <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">HTTPS</span>}
              <span className="font-medium text-default text-sm sm:text-base truncate max-w-[150px] sm:max-w-none">{domain.domain}</span>
              <StatusBadge status={domain.status} labels={statusLabels} />
              <span className="text-xs text-muted hidden sm:inline">({domain.services.length} {t.services})</span>
            </div>
          </div>
          <div className="flex items-center gap-2 lg:hidden">
            <TrendIcon trend={trend} />
            {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
          </div>
        </div>

        {/* Mobile Metrics */}
        <div className="flex items-center gap-3 lg:hidden ml-8 sm:ml-10">
          <div className="text-center">
            <div className={`text-sm font-semibold ${availability >= targets.availability ? "text-emerald-500" : "text-red-500"}`}>{availability.toFixed(1)}%</div>
            <div className="text-[10px] text-muted">{t.availability}</div>
          </div>
          <div className="text-center">
            <div className={`text-sm font-semibold ${errorRate <= 1 ? "text-emerald-500" : "text-red-500"}`}>{errorRate.toFixed(2)}%</div>
            <div className="text-[10px] text-muted">{t.errorRate}</div>
          </div>
          <div className="text-center">
            <div className="text-sm font-semibold text-default">{formatNumber(rps)}/s</div>
            <div className="text-[10px] text-muted">{t.throughput}</div>
          </div>
        </div>

        {/* Desktop Metrics */}
        <div className="hidden lg:flex items-center gap-5">
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">{t.availability}</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${availability >= targets.availability ? "text-emerald-500" : "text-red-500"}`}>{availability.toFixed(2)}%</span>
              <span className="text-xs text-muted">/ {targets.availability}%</span>
            </div>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">{t.p95Latency}</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${p95Latency <= targets.p95Latency ? "text-emerald-500" : "text-amber-500"}`}>{formatLatency(p95Latency)}</span>
              <span className="text-xs text-muted">/ {formatLatency(targets.p95Latency)}</span>
            </div>
          </div>
          <div className="w-28">
            <div className="text-[10px] text-muted mb-0.5">{t.errorRate}</div>
            <span className={`text-sm font-semibold ${errorRate <= 1 ? "text-emerald-500" : "text-red-500"}`}>{errorRate.toFixed(2)}%</span>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">{t.errorBudget}</div>
            <ErrorBudgetBar percent={domain.errorBudgetRemaining} />
          </div>
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">{t.throughput}</div>
            <span className="text-sm font-semibold text-default">{formatNumber(rps)}/s</span>
          </div>
        </div>

        {/* Desktop expand */}
        <div className="hidden lg:flex items-center gap-2">
          <TrendIcon trend={trend} />
          {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
        </div>
      </button>

      {/* Expanded Details */}
      {expanded && (
        <div className="border-t border-[var(--border-color)]">
          {/* Tabs */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 px-3 sm:px-4 pt-3 pb-2 border-b border-[var(--border-color)]">
            <div className="flex items-center gap-1 overflow-x-auto">
              {[
                { id: "overview" as const, label: t.tabOverview, icon: Activity },
                { id: "latency" as const, label: t.tabLatency, icon: BarChart3 },
                { id: "mesh" as const, label: t.tabMesh, icon: Network },
                { id: "compare" as const, label: t.tabCompare, icon: Calendar },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-1 sm:gap-1.5 px-2.5 sm:px-3 py-1.5 text-xs rounded-lg transition-colors whitespace-nowrap ${
                    activeTab === tab.id ? "bg-primary/10 text-primary" : "text-muted hover:text-default hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  <tab.icon className="w-3.5 h-3.5" />
                  <span className="hidden sm:inline">{tab.label}</span>
                </button>
              ))}
            </div>
            <button
              onClick={() => setShowTargetModal(true)}
              className="flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 text-xs rounded-lg text-muted hover:text-default hover:bg-[var(--hover-bg)] transition-colors self-end sm:self-auto"
            >
              <Settings2 className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">{t.configTarget}</span>
            </button>
          </div>

          <div className="bg-[var(--background)]">
            {activeTab === "overview" && (
              <div className="p-3 sm:p-4">
                <OverviewTab
                  summary={domain.summary}
                  errorBudgetRemaining={domain.errorBudgetRemaining}
                  targets={targets}
                  history={history.length > 0 ? history : undefined}
                  t={{
                    availability: t.availability,
                    p95Latency: t.p95Latency,
                    p99Latency: t.p99Latency,
                    errorRate: t.errorRate,
                    totalRequests: t.totalRequests,
                    errorBudget: t.errorBudget,
                    target: t.target,
                    throughput: t.throughput,
                    sloTrend: t.sloTrend,
                    errorBudgetBurn: t.errorBudgetBurn,
                    current: t.current,
                    estimatedExhaust: t.estimatedExhaust,
                    noData: t.noCallData,
                  }}
                />
              </div>
            )}

            {activeTab === "mesh" && (
              <div className="p-3 sm:p-4">
                <MeshTab
                  topology={meshTopology}
                  clusterId={clusterId}
                  timeRange={timeRange}
                  t={{
                    serviceTopology: t.serviceTopology,
                    meshOverview: t.meshOverview,
                    service: t.service,
                    rps: t.rps,
                    p95Latency: t.p95Latency,
                    errorRate: t.errorRate,
                    mtls: t.mtls,
                    status: t.status,
                    healthy: t.healthy,
                    warning: t.warning,
                    critical: t.critical,
                    inbound: t.inbound,
                    outbound: t.outbound,
                    noCallData: t.noCallData,
                    callRelation: t.callRelation,
                    p50Latency: t.p50Latency,
                    p99Latency: t.p99Latency,
                    totalRequests: t.totalRequests,
                    avgLatency: t.avgLatency,
                    statusCodeBreakdown: t.statusCodeBreakdown,
                    latencyDistribution: t.latencyDistribution,
                    requests: t.requests,
                    loading: t.loading,
                  }}
                />
              </div>
            )}

            {activeTab === "latency" && (
              <div className="p-3 sm:p-4">
                <LatencyTab
                  data={latencyData}
                  timeRange={timeRange}
                  t={{
                    latencyDistribution: t.latencyDistribution,
                    methodBreakdown: t.methodBreakdown,
                    statusCodeBreakdown: t.statusCodeBreakdown,
                    requests: t.totalRequests,
                    clearSelection: t.clearSelection,
                    noData: t.tabLatency,
                  }}
                />
              </div>
            )}

            {activeTab === "compare" && (
              <CompareTab
                current={{ availability, p95Latency, errorRate }}
                previous={{ availability: prevAvailability, p95Latency: prevP95Latency, errorRate: prevErrorRate }}
                t={{
                  currentVsPrevious: t.currentVsPrevious,
                  previousPeriod: t.previousPeriod,
                  availability: t.availability,
                  p95Latency: t.p95Latency,
                  errorRate: t.errorRate,
                }}
              />
            )}
          </div>
        </div>
      )}

      <SLOTargetModal
        isOpen={showTargetModal}
        onClose={() => setShowTargetModal(false)}
        domain={domain.domain}
        clusterId={clusterId}
        timeRange={timeRange}
        onSaved={onRefresh}
        t={{
          configSloTarget: t.configSloTarget,
          targetDomain: t.targetDomain,
          selectPeriod: t.selectPeriod,
          day: t.day,
          week: t.week,
          month: t.month,
          targetAvailability: t.targetAvailability,
          targetAvailabilityHint: t.targetAvailabilityHint,
          targetP95: t.targetP95,
          targetP95Hint: t.targetP95Hint,
          errorRateThreshold: t.errorRateThreshold,
          errorRateAutoCalc: t.errorRateAutoCalc,
          cancel: t.cancel,
          save: t.save,
          saving: t.saving,
        }}
      />
    </div>
  );
}
