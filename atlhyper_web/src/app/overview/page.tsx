"use client";

import { useState, useEffect, useRef, memo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getClusterOverview } from "@/api/overview";
import { getClusterList } from "@/api/cluster";
import { LoadingSpinner, PageHeader, Modal } from "@/components/common";
import { AlertTrendsChart } from "@/components/charts";
import { safeNumber, safeString, safePercent, safeTimestamp } from "@/utils/safeData";
import {
  Server,
  Cpu,
  HardDrive,
  AlertTriangle,
  CheckCircle,
  XCircle,
  AlertCircle,
  Package,
  Box,
  Layers,
  Clock,
} from "lucide-react";
import type { ClusterOverview, TransformedOverview } from "@/types/overview";

// 内置刷新间隔：10秒
const REFRESH_INTERVAL = 10000;

// 健康状态卡片（使用 memo 避免不必要重渲染）
const HealthCard = memo(function HealthCard({ data, t }: { data: TransformedOverview["healthCard"]; t: ReturnType<typeof useI18n>["t"] }) {
  const getStatusColor = (status: string) => {
    const s = status.toLowerCase();
    if (s === "healthy") return "text-green-500";
    if (s === "degraded") return "text-yellow-500";
    return "text-red-500";
  };

  const getStatusBg = (status: string) => {
    const s = status.toLowerCase();
    if (s === "healthy") return "bg-green-100 dark:bg-green-900/30";
    if (s === "degraded") return "bg-yellow-100 dark:bg-yellow-900/30";
    return "bg-red-100 dark:bg-red-900/30";
  };

  const StatusIcon = data.status.toLowerCase() === "healthy" ? CheckCircle :
    data.status.toLowerCase() === "degraded" ? AlertCircle : XCircle;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-full">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-default">{t.overview.clusterHealth}</h3>
        <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${getStatusBg(data.status)} ${getStatusColor(data.status)}`}>
          <StatusIcon className="w-3 h-3" />
          {data.status}
        </span>
      </div>
      {data.reason && (
        <p className="text-sm text-muted mb-4 truncate" title={data.reason}>{data.reason}</p>
      )}
      <div className="space-y-3">
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span className="text-muted">{t.overview.nodeReady}</span>
            <span className="text-default transition-all duration-300">{data.nodeReadyPct.toFixed(1)}%</span>
          </div>
          <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${data.nodeReadyPct >= 90 ? "bg-green-500" : data.nodeReadyPct >= 70 ? "bg-yellow-500" : "bg-red-500"}`}
              style={{ width: `${Math.min(100, data.nodeReadyPct)}%` }}
            />
          </div>
        </div>
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span className="text-muted">{t.overview.podHealthy}</span>
            <span className="text-default transition-all duration-300">{data.podHealthyPct.toFixed(1)}%</span>
          </div>
          <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${data.podHealthyPct >= 90 ? "bg-green-500" : data.podHealthyPct >= 70 ? "bg-yellow-500" : "bg-red-500"}`}
              style={{ width: `${Math.min(100, data.podHealthyPct)}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  );
});

// 统计卡片（使用 memo 避免不必要重渲染）
const StatCard = memo(function StatCard({
  title,
  value,
  subText,
  icon: Icon,
  percent,
  accentColor = "#14b8a6",
}: {
  title: string;
  value: string | number;
  subText?: string;
  icon: typeof Server;
  percent?: number;
  accentColor?: string;
}) {
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-full">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-[var(--background)] rounded-lg">
            <Icon className="w-4 h-4" style={{ color: accentColor }} />
          </div>
          <span className="text-sm font-semibold text-default">{title}</span>
        </div>
        {subText && (
          <span className="text-xs text-muted bg-[var(--background)] px-2 py-1 rounded-full">
            {subText}
          </span>
        )}
      </div>
      <div className="text-2xl font-bold text-default mb-2 transition-all duration-300">
        {typeof value === "number" && percent !== undefined
          ? `${value.toFixed(1)}%`
          : value}
      </div>
      {percent !== undefined && (
        <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-300"
            style={{ width: `${Math.min(100, percent)}%`, backgroundColor: accentColor }}
          />
        </div>
      )}
    </div>
  );
});

// 节点资源使用卡片（使用 memo，固定高度，内部滚动，支持排序）
const NodeResourceCard = memo(function NodeResourceCard({ nodes, t }: { nodes: TransformedOverview["nodeUsages"]; t: ReturnType<typeof useI18n>["t"] }) {
  const [sortBy, setSortBy] = useState<"cpu" | "memory">("cpu");

  const getUsageColor = (usage: number) => {
    if (usage >= 80) return "bg-red-500";
    if (usage >= 60) return "bg-yellow-500";
    return "bg-green-500";
  };

  const sortedNodes = [...nodes].sort((a, b) => {
    if (sortBy === "cpu") return b.cpuPercent - a.cpuPercent;
    return b.memoryPercent - a.memoryPercent;
  });

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-[320px] flex flex-col">
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <h3 className="text-lg font-semibold text-default">{t.overview.nodeResourceUsage}</h3>
        <div className="flex gap-1">
          <button
            onClick={() => setSortBy("cpu")}
            className={`px-2 py-1 text-xs rounded transition-colors ${
              sortBy === "cpu"
                ? "bg-orange-500 text-white"
                : "bg-[var(--background)] text-muted hover:text-default"
            }`}
          >
            CPU
          </button>
          <button
            onClick={() => setSortBy("memory")}
            className={`px-2 py-1 text-xs rounded transition-colors ${
              sortBy === "memory"
                ? "bg-green-500 text-white"
                : "bg-[var(--background)] text-muted hover:text-default"
            }`}
          >
            Memory
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto space-y-4 pr-2">
        {nodes.length === 0 ? (
          <div className="text-center py-8 text-muted">{t.overview.noNodeData}</div>
        ) : (
          sortedNodes.map((node) => (
            <div key={node.nodeName} className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-default">{node.nodeName}</span>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-muted">CPU</span>
                    <span className="transition-all duration-300">{node.cpuPercent.toFixed(1)}%</span>
                  </div>
                  <div className="h-1.5 bg-[var(--background)] rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${getUsageColor(node.cpuPercent)}`}
                      style={{ width: `${Math.min(100, node.cpuPercent)}%` }}
                    />
                  </div>
                </div>
                <div>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-muted">Memory</span>
                    <span className="transition-all duration-300">{node.memoryPercent.toFixed(1)}%</span>
                  </div>
                  <div className="h-1.5 bg-[var(--background)] rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${getUsageColor(node.memoryPercent)}`}
                      style={{ width: `${Math.min(100, node.memoryPercent)}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
});

// 告警类型定义
type AlertItem = TransformedOverview["recentAlerts"][number];

// 最近告警卡片（使用 memo，固定高度，内部滚动）
const RecentAlertsCard = memo(function RecentAlertsCard({
  alerts,
  onAlertClick,
  t,
}: {
  alerts: TransformedOverview["recentAlerts"];
  onAlertClick: (alert: AlertItem) => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const getSeverityStyle = (severity: string) => {
    const s = severity.toLowerCase();
    if (s === "critical") return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";
    if (s === "warning") return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400";
    return "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400";
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-[320px] flex flex-col">
      <h3 className="text-lg font-semibold text-default mb-4 flex-shrink-0">{t.overview.recentAlerts}</h3>
      <div className="flex-1 overflow-y-auto space-y-3 pr-2">
        {alerts.length === 0 ? (
          <div className="text-center py-8 text-muted">{t.overview.noRecentAlerts}</div>
        ) : (
          alerts.map((alert, index) => (
            <div
              key={index}
              className="flex items-start gap-3 p-3 bg-[var(--background)] rounded-lg cursor-pointer hover:bg-[var(--background-hover)] transition-colors"
              onClick={() => onAlertClick(alert)}
            >
              <AlertTriangle className={`w-4 h-4 flex-shrink-0 mt-0.5 ${
                alert.severity === "critical" ? "text-red-500" :
                alert.severity === "warning" ? "text-yellow-500" : "text-blue-500"
              }`} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className={`inline-flex px-2 py-0.5 text-xs font-medium rounded-full ${getSeverityStyle(alert.severity)}`}>
                    {alert.severity}
                  </span>
                  <span className="text-xs text-muted">{alert.namespace}</span>
                </div>
                <p className="text-sm text-default truncate" title={alert.message}>
                  {alert.message || alert.reason}
                </p>
                <p className="text-xs text-muted mt-1">
                  {new Date(alert.time).toLocaleString()}
                </p>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
});

// 工作负载汇总卡片
const WorkloadSummaryCard = memo(function WorkloadSummaryCard({
  workloads,
  podStatus,
  peakStats,
  t,
}: {
  workloads: TransformedOverview["workloads"];
  podStatus: TransformedOverview["podStatus"];
  peakStats: TransformedOverview["peakStats"];
  t: ReturnType<typeof useI18n>["t"];
}) {
  const workloadItems = [
    { name: t.overview.deploymentsLabel, icon: Package, total: workloads.deployments.total, ready: workloads.deployments.ready, color: "#6366F1" },
    { name: t.overview.daemonSetsLabel, icon: Layers, total: workloads.daemonsets.total, ready: workloads.daemonsets.ready, color: "#8B5CF6" },
    { name: t.overview.statefulSetsLabel, icon: Box, total: workloads.statefulsets.total, ready: workloads.statefulsets.ready, color: "#EC4899" },
  ];

  const getStatusColor = (ready: number, total: number) => {
    if (total === 0) return "text-muted";
    if (ready === total) return "text-green-500";
    if (ready >= total * 0.5) return "text-yellow-500";
    return "text-red-500";
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4 h-[290px] flex flex-col">
      <h3 className="text-base font-semibold text-default mb-2 flex-shrink-0">{t.overview.workloadSummary}</h3>

      <div className="flex-1 space-y-2 overflow-hidden">
        {/* 工作负载统计 - 3列 */}
        <div className="grid grid-cols-3 gap-2">
          {workloadItems.map((item) => (
            <div key={item.name} className="bg-[var(--background)] rounded-lg p-2">
              <div className="flex items-center gap-1.5 mb-1">
                <item.icon className="w-3.5 h-3.5" style={{ color: item.color }} />
                <span className="text-xs text-muted truncate">{item.name}</span>
              </div>
              <div className={`text-base font-bold ${getStatusColor(item.ready, item.total)}`}>
                {item.ready}/{item.total}
              </div>
            </div>
          ))}
        </div>

        {/* Jobs 单独一行，显示更多信息 */}
        <div className="bg-[var(--background)] rounded-lg p-2 flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5 text-amber-500" />
            <span className="text-xs text-muted">{t.overview.jobsLabel}</span>
            <span className="text-sm font-bold text-default ml-1">{workloads.jobs.total}</span>
          </div>
          <div className="flex gap-3 text-xs">
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-500" />
              <span className="text-muted">{t.overview.run}</span> <strong>{workloads.jobs.running}</strong>
            </span>
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
              <span className="text-muted">{t.overview.done}</span> <strong>{workloads.jobs.succeeded}</strong>
            </span>
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
              <span className="text-muted">{t.overview.fail}</span> <strong>{workloads.jobs.failed}</strong>
            </span>
          </div>
        </div>

        {/* Pod 状态分布 */}
        <div className="bg-[var(--background)] rounded-lg p-2.5">
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-xs text-muted">{t.overview.podStatus}</span>
            <span className="text-xs text-muted">{t.common.total}: {podStatus.total}</span>
          </div>
          <div className="h-2.5 bg-[var(--card-bg)] rounded-full overflow-hidden flex">
            {podStatus.runningPercent > 0 && (
              <div
                className="h-full bg-green-500"
                style={{ width: `${podStatus.runningPercent}%` }}
                title={`Running: ${podStatus.running} (${podStatus.runningPercent.toFixed(1)}%)`}
              />
            )}
            {podStatus.pendingPercent > 0 && (
              <div
                className="h-full bg-yellow-500"
                style={{ width: `${podStatus.pendingPercent}%` }}
                title={`Pending: ${podStatus.pending} (${podStatus.pendingPercent.toFixed(1)}%)`}
              />
            )}
            {podStatus.succeededPercent > 0 && (
              <div
                className="h-full bg-blue-500"
                style={{ width: `${podStatus.succeededPercent}%` }}
                title={`Succeeded: ${podStatus.succeeded} (${podStatus.succeededPercent.toFixed(1)}%)`}
              />
            )}
            {podStatus.failedPercent > 0 && (
              <div
                className="h-full bg-red-500"
                style={{ width: `${podStatus.failedPercent}%` }}
                title={`Failed: ${podStatus.failed} (${podStatus.failedPercent.toFixed(1)}%)`}
              />
            )}
          </div>
          <div className="flex gap-3 mt-1.5 text-xs">
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-green-500" />
              {podStatus.running}
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-yellow-500" />
              {podStatus.pending}
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-blue-500" />
              {podStatus.succeeded}
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-red-500" />
              {podStatus.failed}
            </span>
          </div>
        </div>

        {/* 峰值统计 */}
        {peakStats.hasData && (
          <div className="flex gap-3 text-xs text-muted px-1">
            <span>{t.overview.cpuPeak}: <strong className="text-orange-500">{peakStats.peakCpu.toFixed(1)}%</strong></span>
            <span>{t.overview.memPeak}: <strong className="text-green-500">{peakStats.peakMem.toFixed(1)}%</strong></span>
          </div>
        )}
      </div>
    </div>
  );
});

// 默认空数据结构
const emptyData: TransformedOverview = {
  clusterId: "",
  healthCard: { status: "Unknown", reason: "", nodeReadyPct: 0, podHealthyPct: 0 },
  nodesCard: { totalNodes: 0, readyNodes: 0, nodeReadyPct: 0 },
  cpuCard: { percent: 0 },
  memCard: { percent: 0 },
  alertsTotal: 0,
  workloads: {
    deployments: { total: 0, ready: 0 },
    daemonsets: { total: 0, ready: 0 },
    statefulsets: { total: 0, ready: 0 },
    jobs: { total: 0, running: 0, succeeded: 0, failed: 0 },
  },
  podStatus: {
    total: 0,
    running: 0,
    pending: 0,
    failed: 0,
    succeeded: 0,
    runningPercent: 0,
    pendingPercent: 0,
    failedPercent: 0,
    succeededPercent: 0,
  },
  peakStats: {
    peakCpu: 0,
    peakCpuNode: "",
    peakMem: 0,
    peakMemNode: "",
    hasData: false,
  },
  alertTrends: [],
  recentAlerts: [],
  nodeUsages: [],
};

// 安全转换 API 数据
function transformOverview(data: ClusterOverview | null | undefined): TransformedOverview {
  if (!data) return emptyData;

  const cards = data.cards || {};
  const workloads = data.workloads || {};
  const alerts = data.alerts || {};
  const nodes = data.nodes || {};

  return {
    clusterId: safeString(data.clusterId),
    healthCard: {
      status: safeString(cards.clusterHealth?.status, "Unknown"),
      reason: safeString(cards.clusterHealth?.reason),
      nodeReadyPct: safePercent(cards.clusterHealth?.nodeReadyPercent),
      podHealthyPct: safePercent(cards.clusterHealth?.podReadyPercent),
    },
    nodesCard: {
      totalNodes: safeNumber(cards.nodeReady?.total),
      readyNodes: safeNumber(cards.nodeReady?.ready),
      nodeReadyPct: safePercent(cards.nodeReady?.percent),
    },
    cpuCard: { percent: safePercent(cards.cpuUsage?.percent) },
    memCard: { percent: safePercent(cards.memUsage?.percent) },
    alertsTotal: safeNumber(cards.events24h),
    workloads: {
      deployments: {
        total: safeNumber(workloads.summary?.deployments?.total),
        ready: safeNumber(workloads.summary?.deployments?.ready),
      },
      daemonsets: {
        total: safeNumber(workloads.summary?.daemonsets?.total),
        ready: safeNumber(workloads.summary?.daemonsets?.ready),
      },
      statefulsets: {
        total: safeNumber(workloads.summary?.statefulsets?.total),
        ready: safeNumber(workloads.summary?.statefulsets?.ready),
      },
      jobs: {
        total: safeNumber(workloads.summary?.jobs?.total),
        running: safeNumber(workloads.summary?.jobs?.running),
        succeeded: safeNumber(workloads.summary?.jobs?.succeeded),
        failed: safeNumber(workloads.summary?.jobs?.failed),
      },
    },
    podStatus: {
      total: safeNumber(workloads.podStatus?.total),
      running: safeNumber(workloads.podStatus?.running),
      pending: safeNumber(workloads.podStatus?.pending),
      failed: safeNumber(workloads.podStatus?.failed),
      succeeded: safeNumber(workloads.podStatus?.succeeded),
      runningPercent: safePercent(workloads.podStatus?.runningPercent),
      pendingPercent: safePercent(workloads.podStatus?.pendingPercent),
      failedPercent: safePercent(workloads.podStatus?.failedPercent),
      succeededPercent: safePercent(workloads.podStatus?.succeededPercent),
    },
    peakStats: {
      peakCpu: safeNumber(workloads.peakStats?.peakCpu),
      peakCpuNode: safeString(workloads.peakStats?.peakCpuNode),
      peakMem: safeNumber(workloads.peakStats?.peakMem),
      peakMemNode: safeString(workloads.peakStats?.peakMemNode),
      hasData: workloads.peakStats?.hasData ?? false,
    },
    alertTrends: (alerts.trend || [])
      .map((p) => ({
        ts: safeTimestamp(p.at),
        kinds: (p.kinds || {}) as Record<string, number>, // 按资源类型统计
      }))
      .filter((it) => it.ts > 0),
    recentAlerts: (alerts.recent || []).map((x) => ({
      time: safeString(x.timestamp),
      severity: safeString(x.severity),
      kind: safeString(x.kind),
      namespace: safeString(x.namespace),
      message: safeString(x.message),
      reason: safeString(x.reason),
      name: safeString(x.name),
    })),
    nodeUsages: (nodes.usage || []).map((it) => ({
      nodeName: safeString(it.node),
      cpuPercent: safePercent(it.cpuUsage),
      memoryPercent: safePercent(it.memUsage),
    })),
  };
}

export default function OverviewPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<TransformedOverview>(emptyData);
  const [error, setError] = useState("");
  const isMountedRef = useRef(true);
  const isFirstLoadRef = useRef(true);

  // 告警详情弹窗状态
  const [selectedAlert, setSelectedAlert] = useState<AlertItem | null>(null);

  // 异步获取数据（静默刷新，不影响 UI）
  useEffect(() => {
    isMountedRef.current = true;

    const fetchData = async () => {
      try {
        // 先获取集群列表，使用第一个可用的集群
        const clusterRes = await getClusterList();
        const clusters = clusterRes.data?.clusters || [];
        if (clusters.length === 0) {
          if (isMountedRef.current && isFirstLoadRef.current) {
            setError(t.common.noCluster);
          }
          return;
        }

        const clusterId = clusters[0].cluster_id;
        const res = await getClusterOverview({ cluster_id: clusterId });
        if (isMountedRef.current) {
          setData(transformOverview(res.data?.data));
          setError("");
        }
      } catch (err) {
        if (isMountedRef.current) {
          // 静默处理错误，保留现有数据
          console.warn("[Overview] Fetch error:", err);
          // 仅首次加载时显示错误
          if (isFirstLoadRef.current) {
            setError(err instanceof Error ? err.message : "Failed to load data");
          }
        }
      } finally {
        if (isMountedRef.current) {
          setLoading(false);
          isFirstLoadRef.current = false;
        }
      }
    };

    // 立即执行一次
    fetchData();

    // 设置 10s 定时刷新
    const intervalId = setInterval(fetchData, REFRESH_INTERVAL);

    return () => {
      isMountedRef.current = false;
      clearInterval(intervalId);
    };
  }, []);

  if (loading) {
    return (
      <Layout>
        <LoadingSpinner />
      </Layout>
    );
  }

  if (error && data === emptyData) {
    return (
      <Layout>
        <div className="text-center py-12 text-red-500">{error}</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.overview} />

        {/* Top Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          <HealthCard data={data.healthCard} t={t} />
          <StatCard
            title={t.overview.nodes}
            value={`${data.nodesCard.readyNodes} / ${data.nodesCard.totalNodes}`}
            subText={`${t.status.ready}: ${data.nodesCard.nodeReadyPct.toFixed(1)}%`}
            icon={Server}
            percent={data.nodesCard.nodeReadyPct}
            accentColor="#6366F1"
          />
          <StatCard
            title={t.overview.clusterAvgCpu}
            value={data.cpuCard.percent}
            icon={Cpu}
            percent={data.cpuCard.percent}
            accentColor="#F97316"
          />
          <StatCard
            title={t.overview.clusterAvgMem}
            value={data.memCard.percent}
            icon={HardDrive}
            percent={data.memCard.percent}
            accentColor="#10B981"
          />
          <StatCard
            title={t.overview.alerts}
            value={data.alertsTotal}
            subText="24h"
            icon={AlertTriangle}
            accentColor="#EF4444"
          />
        </div>

        {/* Charts Section */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <WorkloadSummaryCard workloads={data.workloads} podStatus={data.podStatus} peakStats={data.peakStats} t={t} />
          <AlertTrendsChart series={data.alertTrends} />
        </div>

        {/* Bottom Section */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <NodeResourceCard nodes={data.nodeUsages} t={t} />
          <RecentAlertsCard alerts={data.recentAlerts} onAlertClick={setSelectedAlert} t={t} />
        </div>
      </div>

      {/* 告警详情弹窗 */}
      <Modal
        isOpen={!!selectedAlert}
        onClose={() => setSelectedAlert(null)}
        title={t.overview.alertDetails}
        size="md"
      >
        {selectedAlert && (
          <div className="p-6 space-y-4">
            {/* 严重程度标签 */}
            <div className="flex items-center gap-3">
              <AlertTriangle className={`w-6 h-6 ${
                selectedAlert.severity === "critical" ? "text-red-500" :
                selectedAlert.severity === "warning" ? "text-yellow-500" : "text-blue-500"
              }`} />
              <span className={`inline-flex px-3 py-1 text-sm font-medium rounded-full ${
                selectedAlert.severity === "critical" ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" :
                selectedAlert.severity === "warning" ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400" :
                "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"
              }`}>
                {selectedAlert.severity.toUpperCase()}
              </span>
            </div>

            {/* 详情信息 */}
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-muted">{t.overview.time}</span>
                <p className="text-default font-medium">{new Date(selectedAlert.time).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted">{t.overview.kind}</span>
                <p className="text-default font-medium">{selectedAlert.kind}</p>
              </div>
              <div>
                <span className="text-muted">{t.common.namespace}</span>
                <p className="text-default font-medium">{selectedAlert.namespace}</p>
              </div>
              <div>
                <span className="text-muted">{t.common.name}</span>
                <p className="text-default font-medium">{selectedAlert.name}</p>
              </div>
              <div>
                <span className="text-muted">{t.overview.reason}</span>
                <p className="text-default font-medium">{selectedAlert.reason}</p>
              </div>
            </div>

            {/* 完整消息 */}
            <div>
              <span className="text-sm text-muted">{t.alert.message}</span>
              <div className="mt-2 p-4 bg-[var(--background)] rounded-lg">
                <p className="text-sm text-default whitespace-pre-wrap break-words">
                  {selectedAlert.message}
                </p>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </Layout>
  );
}
