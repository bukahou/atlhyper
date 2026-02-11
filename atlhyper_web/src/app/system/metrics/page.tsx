"use client";

import { useState, useEffect, useMemo, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import {
  RefreshCw,
  Server,
  Cpu,
  HardDrive,
  Thermometer,
  Activity,
  ChevronDown,
  ChevronRight,
  Database,
  AlertTriangle,
  Loader2,
  WifiOff,
} from "lucide-react";

// 组件
import {
  CPUCard,
  MemoryCard,
  DiskCard,
  NetworkCard,
  TemperatureCard,
  ProcessTable,
  ResourceChart,
  GPUCard,
  PSICard,
  TCPCard,
  SystemResourcesCard,
  VMStatCard,
} from "./components";

// API
import {
  getClusterNodeMetrics,
  getNodeMetricsHistory,
  type ClusterMetricsSummary,
} from "@/api/node-metrics";

// 工具函数
import { formatBytes } from "./mock/data";

import type { NodeMetricsSnapshot, MetricsDataPoint } from "@/types/node-metrics";

// ==================== 汇总卡片组件 ====================
function SummaryCard({
  icon: Icon,
  label,
  value,
  subValue,
  color,
}: {
  icon: typeof Activity;
  label: string;
  value: string;
  subValue?: string;
  color: string;
}) {
  return (
    <div className="p-2.5 sm:p-4 rounded-xl bg-card border border-[var(--border-color)]">
      <div className="flex items-center gap-2 sm:gap-3">
        <div className={`p-1.5 sm:p-2 rounded-lg ${color}`}>
          <Icon className="w-4 h-4 sm:w-5 sm:h-5" />
        </div>
        <div className="min-w-0">
          <div className="text-[10px] sm:text-xs text-muted truncate">{label}</div>
          <div className="text-base sm:text-xl font-bold text-default">{value}</div>
          {subValue && <div className="text-[10px] sm:text-xs text-muted truncate hidden sm:block">{subValue}</div>}
        </div>
      </div>
    </div>
  );
}

// ==================== 节点状态徽章 ====================
function NodeStatusBadge({ isOnline }: { isOnline: boolean }) {
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${
      isOnline
        ? "bg-emerald-500/10 text-emerald-500"
        : "bg-red-500/10 text-red-500"
    }`}>
      <span className={`w-1.5 h-1.5 rounded-full ${isOnline ? "bg-emerald-500" : "bg-red-500"}`} />
      {isOnline ? "Online" : "Offline"}
    </span>
  );
}

// ==================== 节点卡片组件 ====================
function NodeCard({
  metrics,
  historyData,
  expanded,
  onToggle,
}: {
  metrics: NodeMetricsSnapshot;
  historyData: MetricsDataPoint[];
  expanded: boolean;
  onToggle: () => void;
}) {
  const cpuUsage = metrics.cpu.usagePercent;
  const memUsage = metrics.memory.usagePercent;
  const temp = metrics.temperature.cpuTemp;

  const getUsageColor = (usage: number) => {
    if (usage >= 80) return "text-red-500";
    if (usage >= 60) return "text-yellow-500";
    return "text-emerald-500";
  };

  const getTempColor = (t: number) => {
    if (t >= 80) return "text-red-500";
    if (t >= 65) return "text-yellow-500";
    return "text-emerald-500";
  };

  return (
    <div className="border border-[var(--border-color)] rounded-xl overflow-hidden bg-card">
      {/* 节点摘要行 */}
      <button
        onClick={onToggle}
        className="w-full px-3 sm:px-4 py-2.5 sm:py-3 flex items-center gap-2 sm:gap-4 hover:bg-[var(--hover-bg)] transition-colors active:bg-[var(--hover-bg)]"
      >
        {/* 节点信息 */}
        <div className="flex items-center gap-2 sm:gap-3 flex-1 min-w-0">
          <div className="p-1.5 sm:p-2 rounded-lg bg-emerald-500/10 flex-shrink-0">
            <Server className="w-4 h-4 text-emerald-500" />
          </div>
          <div className="text-left min-w-0 flex-1">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-medium text-default text-sm sm:text-base truncate max-w-[120px] sm:max-w-none">{metrics.nodeName}</span>
              <NodeStatusBadge isOnline={true} />
            </div>
            {/* 移动端显示关键指标 */}
            <div className="flex items-center gap-3 mt-1 lg:hidden">
              <span className={`text-xs font-medium ${getUsageColor(cpuUsage)}`}>
                CPU {cpuUsage.toFixed(0)}%
              </span>
              <span className={`text-xs font-medium ${getUsageColor(memUsage)}`}>
                MEM {memUsage.toFixed(0)}%
              </span>
              {temp > 0 && (
                <span className={`text-xs font-medium ${getTempColor(temp)}`}>
                  {temp.toFixed(0)}°C
                </span>
              )}
            </div>
          </div>
        </div>

        {/* 汇总指标 - 仅桌面端 */}
        <div className="hidden lg:flex items-center gap-5">
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">CPU</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${getUsageColor(cpuUsage)}`}>
                {cpuUsage.toFixed(1)}%
              </span>
            </div>
          </div>
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">Memory</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${getUsageColor(memUsage)}`}>
                {memUsage.toFixed(1)}%
              </span>
            </div>
          </div>
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">Temperature</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${getTempColor(temp)}`}>
                {temp > 0 ? `${temp.toFixed(1)}°C` : "N/A"}
              </span>
            </div>
          </div>
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">Load Avg</div>
            <div className="flex items-center gap-1">
              <span className="text-sm font-semibold text-default">
                {metrics.cpu.loadAvg1.toFixed(2)}
              </span>
            </div>
          </div>
          <div className="w-28">
            <div className="text-[10px] text-muted mb-0.5">Disk</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${getUsageColor(metrics.disks[0]?.usagePercent || 0)}`}>
                {(metrics.disks[0]?.usagePercent || 0).toFixed(1)}%
              </span>
              <span className="text-xs text-muted">({metrics.disks[0]?.mountPoint || "/"})</span>
            </div>
          </div>
        </div>

        <div className="flex items-center flex-shrink-0">
          {expanded ? (
            <ChevronDown className="w-4 h-4 text-muted" />
          ) : (
            <ChevronRight className="w-4 h-4 text-muted" />
          )}
        </div>
      </button>

      {/* 展开详情 */}
      {expanded && (
        <div className="border-t border-[var(--border-color)] bg-[var(--background)] p-3 sm:p-6 space-y-4 sm:space-y-6">
          {/* 资源趋势图 */}
          <ResourceChart data={historyData} title="Resource Trends" />

          {/* 第一行：CPU + Memory */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <CPUCard data={metrics.cpu} />
            <MemoryCard data={metrics.memory} />
          </div>

          {/* 第二行：Disk + Network */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <DiskCard data={metrics.disks} />
            <NetworkCard data={metrics.networks} />
          </div>

          {/* 第三行：Temperature + GPU (如果有) */}
          {metrics.gpus && metrics.gpus.length > 0 ? (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
              <TemperatureCard data={metrics.temperature} />
              <GPUCard data={metrics.gpus} />
            </div>
          ) : (
            <TemperatureCard data={metrics.temperature} />
          )}

          {/* 第四行：PSI + TCP */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <PSICard data={metrics.psi} />
            <TCPCard tcp={metrics.tcp} softnet={metrics.softnet} />
          </div>

          {/* 第五行：System Resources + VMStat */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <SystemResourcesCard system={metrics.system} ntp={metrics.ntp} />
            <VMStatCard data={metrics.vmstat} />
          </div>

          {/* 进程列表 */}
          {metrics.topProcesses.length > 0 && (
            <ProcessTable data={metrics.topProcesses} />
          )}
        </div>
      )}
    </div>
  );
}

// ==================== 主页面 ====================
export default function MetricsPage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();

  const [expandedNode, setExpandedNode] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
  const [isRefreshing, setIsRefreshing] = useState(false);

  // 数据状态
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [summary, setSummary] = useState<ClusterMetricsSummary | null>(null);
  const [nodes, setNodes] = useState<NodeMetricsSnapshot[]>([]);

  // 历史数据缓存（每个节点）
  const [historyCache, setHistoryCache] = useState<Record<string, MetricsDataPoint[]>>({});

  // 加载数据
  const loadData = useCallback(async (showLoading = true) => {
    if (!currentClusterId) return;

    if (showLoading) setIsRefreshing(true);

    try {
      const result = await getClusterNodeMetrics(currentClusterId);
      setSummary(result.summary);
      setNodes(result.nodes);
      setError(null);
      setLastUpdate(new Date());
    } catch (err) {
      console.error("Failed to load node metrics:", err);
      setError("Failed to load metrics data");
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  }, [currentClusterId]);

  // 初始加载
  useEffect(() => {
    loadData();
  }, [loadData]);

  // 自动刷新 (10秒)
  useEffect(() => {
    const interval = setInterval(() => {
      loadData(false);
    }, 10000);
    return () => clearInterval(interval);
  }, [loadData]);

  // 手动刷新
  const handleRefresh = () => {
    loadData(true);
  };

  // 节点展开/收起
  const handleNodeToggle = useCallback(async (nodeName: string) => {
    const isExpanding = expandedNode !== nodeName;
    setExpandedNode(isExpanding ? nodeName : null);

    // 展开时加载历史数据（如果尚未缓存）
    if (isExpanding && currentClusterId && !historyCache[nodeName]) {
      try {
        const historyResult = await getNodeMetricsHistory(currentClusterId, nodeName, 24);
        setHistoryCache(prev => ({
          ...prev,
          [nodeName]: historyResult.data,
        }));
      } catch (err) {
        console.error("Failed to load history data:", err);
      }
    }
  }, [expandedNode, currentClusterId, historyCache]);

  // 计算告警节点数
  const warningNodes = useMemo(() => {
    return nodes.filter((node) => {
      return (
        node.cpu.usagePercent >= 80 ||
        node.memory.usagePercent >= 80 ||
        node.temperature.cpuTemp >= 75
      );
    }).length;
  }, [nodes]);

  // Loading 状态
  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  // 无集群选中
  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">No Cluster Selected</p>
          <p className="text-sm text-muted">Please select a cluster from the sidebar</p>
        </div>
      </Layout>
    );
  }

  // 错误状态
  if (error && nodes.length === 0) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <AlertTriangle className="w-12 h-12 mb-4 text-yellow-500" />
          <p className="text-default font-medium mb-2">{error}</p>
          <button
            onClick={handleRefresh}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            Retry
          </button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="-m-6">
        {/* 固定头部 + 概览卡片 */}
        <div className="sticky top-[-24px] z-10 bg-card rounded-t-2xl">
          {/* 标题栏 */}
          <div className="px-3 sm:px-6 py-3 sm:py-4 border-b border-[var(--border-color)]">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 sm:gap-3 min-w-0">
                <div className="p-1.5 sm:p-2 rounded-xl bg-gradient-to-br from-orange-100 to-red-100 dark:from-orange-900/30 dark:to-red-900/30 flex-shrink-0">
                  <Activity className="w-5 h-5 sm:w-6 sm:h-6 text-orange-600 dark:text-orange-400" />
                </div>
                <div className="min-w-0">
                  <h1 className="text-base sm:text-lg font-semibold text-default truncate">{t.nav.metrics}</h1>
                  <p className="text-[10px] sm:text-xs text-muted hidden sm:block">Node hardware metrics - CPU, Memory, Disk, Network, Temperature</p>
                </div>
              </div>
              <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
                <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
                  Last: {lastUpdate.toLocaleTimeString()}
                </span>
                <button
                  onClick={handleRefresh}
                  disabled={isRefreshing}
                  className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
                >
                  <RefreshCw className={`w-4 h-4 ${isRefreshing ? "animate-spin" : ""}`} />
                </button>
              </div>
            </div>
          </div>

          {/* 集群概览卡片 */}
          {summary && (
            <div className="px-3 sm:px-6 py-3 sm:py-4 border-b border-[var(--border-color)] bg-[var(--background)]">
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2 sm:gap-4">
                <SummaryCard
                  icon={Server}
                  label="Nodes"
                  value={`${summary.onlineNodes}/${summary.totalNodes}`}
                  subValue={`${summary.onlineNodes} with metrics`}
                  color="bg-blue-500/10 text-blue-500"
                />
                <SummaryCard
                  icon={Cpu}
                  label="Avg CPU"
                  value={`${summary.avgCPUUsage.toFixed(1)}%`}
                  subValue={`Max: ${summary.maxCPUUsage.toFixed(1)}%`}
                  color={summary.avgCPUUsage >= 80 ? "bg-red-500/10 text-red-500" : summary.avgCPUUsage >= 60 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
                />
                <SummaryCard
                  icon={HardDrive}
                  label="Avg Memory"
                  value={`${summary.avgMemoryUsage.toFixed(1)}%`}
                  subValue={`${formatBytes(summary.usedMemory)} / ${formatBytes(summary.totalMemory)}`}
                  color={summary.avgMemoryUsage >= 80 ? "bg-red-500/10 text-red-500" : summary.avgMemoryUsage >= 60 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
                />
                <SummaryCard
                  icon={Thermometer}
                  label="Max Temp"
                  value={summary.maxCPUTemp > 0 ? `${summary.maxCPUTemp.toFixed(1)}°C` : "N/A"}
                  subValue={summary.avgCPUTemp > 0 ? `Avg: ${summary.avgCPUTemp.toFixed(1)}°C` : ""}
                  color={summary.maxCPUTemp >= 80 ? "bg-red-500/10 text-red-500" : summary.maxCPUTemp >= 65 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
                />
                <SummaryCard
                  icon={Database}
                  label="Avg Disk"
                  value={`${summary.avgDiskUsage.toFixed(1)}%`}
                  subValue={`Max: ${summary.maxDiskUsage.toFixed(1)}%`}
                  color={summary.maxDiskUsage >= 80 ? "bg-red-500/10 text-red-500" : summary.maxDiskUsage >= 60 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
                />
                <SummaryCard
                  icon={AlertTriangle}
                  label="Warnings"
                  value={warningNodes.toString()}
                  subValue="nodes need attention"
                  color={warningNodes > 0 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
                />
              </div>
            </div>
          )}
        </div>

        {/* 可滚动内容区域 */}
        <div className="p-3 sm:p-6 space-y-4 sm:space-y-6 bg-[var(--background)]">
          {/* 节点列表标题 */}
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-default">
              Node Metrics
              <span className="ml-2 text-xs font-normal text-muted">
                ({nodes.length})
              </span>
            </h2>
          </div>

          {/* 节点列表 */}
          {nodes.length === 0 ? (
            <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
              <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
              <p className="text-default font-medium mb-2">No Metrics Data</p>
              <p className="text-sm text-muted">No nodes are reporting metrics. Please ensure atlhyper-metrics is deployed.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {nodes.map((node) => (
                <NodeCard
                  key={node.nodeName}
                  metrics={node}
                  historyData={historyCache[node.nodeName] || []}
                  expanded={expandedNode === node.nodeName}
                  onToggle={() => handleNodeToggle(node.nodeName)}
                />
              ))}
            </div>
          )}

          {/* 说明 - 仅桌面端显示 */}
          <div className="hidden sm:block p-4 rounded-xl bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
            <div className="flex items-start gap-3">
              <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-900/50 flex-shrink-0">
                <Activity className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="text-sm">
                <p className="font-medium text-blue-800 dark:text-blue-200 mb-1">Data Source</p>
                <p className="text-blue-700 dark:text-blue-300 text-xs leading-relaxed">
                  Hardware metrics are collected from OTel Collector (node_exporter) on each node.
                  Data includes CPU, memory, disk I/O, network traffic, temperature, PSI, TCP stack, system resources, and virtual memory stats.
                  Metrics are pushed to the Agent every 5 seconds.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
