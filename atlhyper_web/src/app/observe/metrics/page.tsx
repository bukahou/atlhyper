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
  ClusterOverviewChart,
} from "./components";

// 数据源代理层（自动切换 mock / api）
import {
  getClusterNodeMetrics,
  getNodeMetricsHistory,
} from "@/datasource/metrics";
import type { ClusterMetricsSummary } from "@/datasource/metrics";

// 工具函数
import { formatBytes } from "@/lib/format";

import type { NodeMetricsSnapshot, MetricsDataPoint } from "@/types/node-metrics";

// ==================== 工具函数 ====================
const uptimeStr = (s: number) => {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600);
  return d > 0 ? `${d}d ${h}h` : `${h}h`;
};

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
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3">
      <div className="flex items-center gap-2 mb-2">
        <div className={`p-1.5 rounded-lg ${color}`}>
          <Icon className="w-4 h-4 sm:w-5 sm:h-5" />
        </div>
        <span className="text-xs text-muted">{label}</span>
      </div>
      <div className="text-xl font-bold text-default">{value}</div>
      {subValue && <div className="text-[10px] text-muted mt-0.5">{subValue}</div>}
    </div>
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
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const cpuUsage = metrics.cpu.usagePercent;
  const memUsage = metrics.memory.usagePercent;
  const temp = metrics.temperature.cpuTemp;
  const rootDisk = metrics.disks.find(d => d.mountPoint === "/") || metrics.disks[0];
  const diskPct = rootDisk?.usagePercent || 0;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      {/* 节点摘要行 - 紧凑内联风格 */}
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between p-3 sm:p-4 hover:bg-[var(--background)] transition-colors"
      >
        <div className="flex items-center gap-3">
          {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
          <div className="flex items-center gap-2">
            <Server className="w-4 h-4 text-indigo-500" />
            <span className="text-sm font-semibold text-default">{metrics.nodeName}</span>
            {metrics.cpu.coreCount > 8 ? (
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-500">{nm.node.controlPlane}</span>
            ) : (
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-500">{nm.node.worker}</span>
            )}
            <span className="text-[10px] text-muted hidden sm:inline">{metrics.os || "linux"}</span>
          </div>
        </div>
        <div className="flex items-center gap-3 sm:gap-5 text-xs">
          <span><span className="text-muted">CPU </span><span className={getUsageColor(cpuUsage)}>{cpuUsage.toFixed(1)}%</span></span>
          <span><span className="text-muted">Mem </span><span className={getUsageColor(memUsage)}>{memUsage.toFixed(1)}%</span></span>
          <span className="hidden sm:inline"><span className="text-muted">Disk </span><span className={getUsageColor(diskPct)}>{diskPct.toFixed(1)}%</span><span className="text-muted"> ({rootDisk?.mountPoint || "/"})</span></span>
          <span className="hidden sm:inline"><span className="text-muted">Temp </span><span className={getTempColor(temp)}>{temp > 0 ? `${temp.toFixed(1)}°C` : "N/A"}</span></span>
          <span className="hidden lg:inline text-muted">up {uptimeStr(metrics.uptime)}</span>
        </div>
      </button>

      {/* 展开详情 - 无额外背景色，透明融合 */}
      {expanded && (
        <div className="px-3 sm:px-4 pb-3 sm:pb-4 space-y-4 sm:space-y-6">
          {/* 系统信息条 */}
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-muted px-1">
            {metrics.os && <span>{metrics.os}</span>}
            {metrics.kernel && <span>{metrics.kernel}</span>}
            <span>{nm.node.uptime}: {uptimeStr(metrics.uptime)}</span>
          </div>

          {/* 资源趋势图 */}
          <ResourceChart data={historyData} />

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
  const nm = t.nodeMetrics;
  const { currentClusterId } = useClusterStore();

  const [expandedNode, setExpandedNode] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
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
  const handleNodeToggle = useCallback((nodeName: string) => {
    const isExpanding = expandedNode !== nodeName;
    setExpandedNode(isExpanding ? nodeName : null);

    // 展开时加载历史数据（如果尚未缓存）
    if (isExpanding && !historyCache[nodeName] && currentClusterId) {
      getNodeMetricsHistory(currentClusterId, nodeName, 24).then(historyResult => {
        setHistoryCache(prev => ({
          ...prev,
          [nodeName]: historyResult.data,
        }));
      });
    }
  }, [expandedNode, historyCache, currentClusterId]);

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

  // 过滤后的节点列表
  const displayedNodes = selectedNode ? nodes.filter(n => n.nodeName === selectedNode) : nodes;

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
          <p className="text-default font-medium mb-2">{nm.noCluster}</p>
          <p className="text-sm text-muted">{nm.noClusterDesc}</p>
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
            {nm.retry}
          </button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-6">
        {/* 标题栏 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg sm:text-xl font-bold text-default">{t.nav.metrics}</h1>
            <p className="text-xs sm:text-sm text-muted mt-1">
              {nm.pageDescription}
            </p>
          </div>
          <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
            <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
              {nm.lastUpdate}: {lastUpdate.toLocaleTimeString()}
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

        {/* 集群概览卡片 */}
        {summary && (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
            <SummaryCard
              icon={Server}
              label={nm.summary.nodes}
              value={`${summary.onlineNodes}/${summary.totalNodes}`}
              subValue={`${warningNodes} ${nm.summary.warnings}`}
              color="bg-indigo-500/10 text-indigo-500"
            />
            <SummaryCard
              icon={Cpu}
              label={nm.summary.avgCpu}
              value={`${summary.avgCPUUsage.toFixed(1)}%`}
              subValue={`Max: ${summary.maxCPUUsage.toFixed(1)}%`}
              color={summary.avgCPUUsage >= 80 ? "bg-red-500/10 text-red-500" : summary.avgCPUUsage >= 60 ? "bg-yellow-500/10 text-yellow-500" : "bg-orange-500/10 text-orange-500"}
            />
            <SummaryCard
              icon={HardDrive}
              label={nm.summary.avgMemory}
              value={`${summary.avgMemoryUsage.toFixed(1)}%`}
              subValue={`${formatBytes(summary.usedMemory)} / ${formatBytes(summary.totalMemory)}`}
              color={summary.avgMemoryUsage >= 80 ? "bg-red-500/10 text-red-500" : summary.avgMemoryUsage >= 60 ? "bg-yellow-500/10 text-yellow-500" : "bg-green-500/10 text-green-500"}
            />
            <SummaryCard
              icon={Thermometer}
              label={nm.summary.maxTemp}
              value={summary.maxCPUTemp > 0 ? `${summary.maxCPUTemp.toFixed(1)}°C` : nm.temperature.na}
              subValue={summary.avgCPUTemp > 0 ? `Avg: ${summary.avgCPUTemp.toFixed(1)}°C` : ""}
              color={summary.maxCPUTemp >= 80 ? "bg-red-500/10 text-red-500" : summary.maxCPUTemp >= 65 ? "bg-yellow-500/10 text-yellow-500" : "bg-cyan-500/10 text-cyan-500"}
            />
            <SummaryCard
              icon={Database}
              label={nm.summary.avgDisk}
              value={`${summary.avgDiskUsage.toFixed(1)}%`}
              subValue={`Max: ${summary.maxDiskUsage.toFixed(1)}%`}
              color={summary.maxDiskUsage >= 80 ? "bg-red-500/10 text-red-500" : summary.maxDiskUsage >= 60 ? "bg-yellow-500/10 text-yellow-500" : "bg-blue-500/10 text-blue-500"}
            />
            <SummaryCard
              icon={AlertTriangle}
              label={nm.summary.warnings}
              value={warningNodes.toString()}
              subValue={nm.summary.nodesNeedAttention}
              color={warningNodes > 0 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
            />
          </div>
        )}

        {/* 集群概览趋势图 */}
        {nodes.length > 1 && currentClusterId && (
          <ClusterOverviewChart nodes={nodes} clusterId={currentClusterId} />
        )}

        {/* 节点过滤 chip */}
        <div className="flex flex-wrap gap-2">
          <button
            className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${!selectedNode ? "bg-indigo-500 text-white border-indigo-500" : "bg-card text-muted border-[var(--border-color)] hover:text-default"}`}
            onClick={() => setSelectedNode(null)}
          >
            {nm.allNodes} ({nodes.length})
          </button>
          {nodes.map(n => (
            <button
              key={n.nodeName}
              className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${selectedNode === n.nodeName ? "bg-indigo-500 text-white border-indigo-500" : "bg-card text-muted border-[var(--border-color)] hover:text-default"}`}
              onClick={() => setSelectedNode(selectedNode === n.nodeName ? null : n.nodeName)}
            >
              {n.nodeName}
            </button>
          ))}
        </div>

        {/* 节点列表 */}
        {nodes.length === 0 ? (
          <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
            <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
            <p className="text-default font-medium mb-2">{nm.noMetricsData}</p>
            <p className="text-sm text-muted">{nm.noMetricsDesc}</p>
          </div>
        ) : (
          <div className="space-y-3">
            {displayedNodes.map((node) => (
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
      </div>
    </Layout>
  );
}
