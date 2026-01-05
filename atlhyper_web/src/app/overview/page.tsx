"use client";

import { useState, useEffect, useRef, memo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getClusterOverview } from "@/api/overview";
import { LoadingSpinner, PageHeader } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { ResourceTrendsChart, AlertTrendsChart } from "@/components/charts";
import { safeNumber, safeString, safePercent, safeTimestamp } from "@/utils/safeData";
import {
  Server,
  Cpu,
  HardDrive,
  AlertTriangle,
  CheckCircle,
  XCircle,
  AlertCircle,
} from "lucide-react";
import type { ClusterOverview, TransformedOverview } from "@/types/overview";

// 内置刷新间隔：10秒
const REFRESH_INTERVAL = 10000;

// 健康状态卡片（使用 memo 避免不必要重渲染）
const HealthCard = memo(function HealthCard({ data }: { data: TransformedOverview["healthCard"] }) {
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
        <h3 className="text-sm font-semibold text-default">Cluster Health</h3>
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
            <span className="text-muted">Node Ready</span>
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
            <span className="text-muted">Pod Healthy</span>
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
const NodeResourceCard = memo(function NodeResourceCard({ nodes }: { nodes: TransformedOverview["nodeUsages"] }) {
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
        <h3 className="text-lg font-semibold text-default">Node Resource Usage</h3>
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
          <div className="text-center py-8 text-muted">No node data</div>
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

// 最近告警卡片（使用 memo，固定高度，内部滚动）
const RecentAlertsCard = memo(function RecentAlertsCard({ alerts }: { alerts: TransformedOverview["recentAlerts"] }) {
  const getSeverityStyle = (severity: string) => {
    const s = severity.toLowerCase();
    if (s === "critical") return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";
    if (s === "warning") return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400";
    return "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400";
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 h-[320px] flex flex-col">
      <h3 className="text-lg font-semibold text-default mb-4 flex-shrink-0">Recent Alerts (24h)</h3>
      <div className="flex-1 overflow-y-auto space-y-3 pr-2">
        {alerts.length === 0 ? (
          <div className="text-center py-8 text-muted">No recent alerts</div>
        ) : (
          alerts.map((alert, index) => (
            <div key={index} className="flex items-start gap-3 p-3 bg-[var(--background)] rounded-lg">
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

// 默认空数据结构
const emptyData: TransformedOverview = {
  clusterId: "",
  healthCard: { status: "Unknown", reason: "", nodeReadyPct: 0, podHealthyPct: 0 },
  nodesCard: { totalNodes: 0, readyNodes: 0, nodeReadyPct: 0 },
  cpuCard: { percent: 0 },
  memCard: { percent: 0 },
  alertsTotal: 0,
  cpuSeries: [],
  memSeries: [],
  tempSeries: [],
  alertTrends: [],
  recentAlerts: [],
  nodeUsages: [],
  peakStats: {
    peakCpu: 0,
    peakCpuNode: "",
    peakMem: 0,
    peakMemNode: "",
    peakTemp: 0,
    peakTempNode: "",
    netRxKBps: 0,
    netTxKBps: 0,
    hasData: false,
  },
};

// 安全转换 API 数据
function transformOverview(data: ClusterOverview | null | undefined): TransformedOverview {
  if (!data) return emptyData;

  const cards = data.cards || {};
  const trends = data.trends || {};
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
    cpuSeries: (trends.resourceUsage || []).map((p) => [
      safeTimestamp(p.at),
      safePercent(safeNumber(p.cpuPeak) * 100),
    ] as [number, number]),
    memSeries: (trends.resourceUsage || []).map((p) => [
      safeTimestamp(p.at),
      safePercent(safeNumber(p.memPeak) * 100),
    ] as [number, number]),
    tempSeries: (trends.resourceUsage || []).map((p) => [
      safeTimestamp(p.at),
      safeNumber(p.tempPeak),
    ] as [number, number]),
    alertTrends: (alerts.trend || [])
      .map((p) => ({
        ts: safeTimestamp(p.at),
        critical: safeNumber(p.critical),
        warning: safeNumber(p.warning),
        info: safeNumber(p.info),
      }))
      .filter((it) => it.ts > 0),
    recentAlerts: (alerts.recent || []).map((x) => ({
      time: safeString(x.Timestamp),
      severity: safeString(x.Severity),
      kind: safeString(x.Kind),
      namespace: safeString(x.Namespace),
      message: safeString(x.Message),
      reason: safeString(x.ReasonCode),
      name: safeString(x.Name),
    })),
    nodeUsages: (nodes.usage || []).map((it) => ({
      nodeName: safeString(it.node),
      cpuPercent: safePercent(it.cpuUsage),
      memoryPercent: safePercent(it.memUsage),
    })),
    peakStats: {
      peakCpu: safeNumber(trends.peakStats?.peakCpu),
      peakCpuNode: safeString(trends.peakStats?.peakCpuNode),
      peakMem: safeNumber(trends.peakStats?.peakMem),
      peakMemNode: safeString(trends.peakStats?.peakMemNode),
      peakTemp: safeNumber(trends.peakStats?.peakTemp),
      peakTempNode: safeString(trends.peakStats?.peakTempNode),
      netRxKBps: safeNumber(trends.peakStats?.netRxKBps),
      netTxKBps: safeNumber(trends.peakStats?.netTxKBps),
      hasData: trends.peakStats?.hasData ?? false,
    },
  };
}

export default function OverviewPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<TransformedOverview>(emptyData);
  const [error, setError] = useState("");
  const isMountedRef = useRef(true);
  const isFirstLoadRef = useRef(true);

  // 异步获取数据（静默刷新，不影响 UI）
  useEffect(() => {
    isMountedRef.current = true;

    const fetchData = async () => {
      try {
        const res = await getClusterOverview({ ClusterID: getCurrentClusterId() });
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
          <HealthCard data={data.healthCard} />
          <StatCard
            title="Nodes"
            value={`${data.nodesCard.readyNodes} / ${data.nodesCard.totalNodes}`}
            subText={`Ready: ${data.nodesCard.nodeReadyPct.toFixed(1)}%`}
            icon={Server}
            percent={data.nodesCard.nodeReadyPct}
            accentColor="#6366F1"
          />
          <StatCard
            title="CPU Usage"
            value={data.cpuCard.percent}
            icon={Cpu}
            percent={data.cpuCard.percent}
            accentColor="#F97316"
          />
          <StatCard
            title="Memory Usage"
            value={data.memCard.percent}
            icon={HardDrive}
            percent={data.memCard.percent}
            accentColor="#10B981"
          />
          <StatCard
            title="Alerts"
            value={data.alertsTotal}
            subText="24h"
            icon={AlertTriangle}
            accentColor="#EF4444"
          />
        </div>

        {/* Charts Section */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <ResourceTrendsChart cpu={data.cpuSeries} mem={data.memSeries} temp={data.tempSeries} peakStats={data.peakStats} />
          <AlertTrendsChart series={data.alertTrends} />
        </div>

        {/* Bottom Section */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <NodeResourceCard nodes={data.nodeUsages} />
          <RecentAlertsCard alerts={data.recentAlerts} />
        </div>
      </div>
    </Layout>
  );
}
