"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import {
  Server,
  Wifi,
  WifiOff,
  Cpu,
  HardDrive,
  Clock,
  RefreshCw,
  CheckCircle,
  AlertCircle,
  Info
} from "lucide-react";
import { getNodeOverview } from "@/api/node";
import { getClusterOverview } from "@/api/overview";

// Agent 状态类型
interface AgentInfo {
  nodeId: string;
  nodeName: string;
  status: "online" | "offline" | "degraded";
  lastSeen: string;
  version: string;
  cpuUsage: number;
  memUsage: number;
  cpuCores: number;
  memoryGiB: number;
  internalIP: string;
  osImage: string;
}

// 状态显示配置
const statusConfig = {
  online: {
    label: "在线",
    color: "text-green-500",
    bgColor: "bg-green-100 dark:bg-green-900/30",
    icon: CheckCircle,
  },
  offline: {
    label: "离线",
    color: "text-red-500",
    bgColor: "bg-red-100 dark:bg-red-900/30",
    icon: AlertCircle,
  },
  degraded: {
    label: "异常",
    color: "text-yellow-500",
    bgColor: "bg-yellow-100 dark:bg-yellow-900/30",
    icon: AlertCircle,
  },
};

// 单个 Agent 卡片
function AgentCard({ agent }: { agent: AgentInfo }) {
  const config = statusConfig[agent.status];
  const StatusIcon = config.icon;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5 hover:shadow-lg transition-shadow">
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-lg ${config.bgColor}`}>
            <Server className={`w-5 h-5 ${config.color}`} />
          </div>
          <div>
            <h3 className="font-semibold text-default">{agent.nodeName}</h3>
            <p className="text-xs text-muted">{agent.internalIP}</p>
          </div>
        </div>
        <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${config.bgColor} ${config.color}`}>
          <StatusIcon className="w-3 h-3" />
          {config.label}
        </span>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 mb-4">
        <div className="flex items-center gap-2">
          <Cpu className="w-4 h-4 text-muted" />
          <div>
            <p className="text-xs text-muted">CPU ({agent.cpuCores} cores)</p>
            <p className="text-sm font-medium text-default">{agent.cpuUsage.toFixed(1)}%</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <HardDrive className="w-4 h-4 text-muted" />
          <div>
            <p className="text-xs text-muted">Mem ({agent.memoryGiB.toFixed(1)} GiB)</p>
            <p className="text-sm font-medium text-default">{agent.memUsage.toFixed(1)}%</p>
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="pt-4 border-t border-[var(--border-color)] space-y-2">
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">Version</span>
          <span className="text-default font-mono">{agent.version}</span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">OS</span>
          <span className="text-default text-xs">{agent.osImage}</span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">Last Seen</span>
          <span className="text-default">{agent.lastSeen}</span>
        </div>
      </div>
    </div>
  );
}

export default function AgentsPage() {
  const { t } = useI18n();
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  const fetchAgents = useCallback(async () => {
    try {
      setLoading(true);
      const clusterID = "ZGMF-X10A";

      // 并行获取节点信息和使用率数据
      const [nodeRes, overviewRes] = await Promise.all([
        getNodeOverview({ ClusterID: clusterID }),
        getClusterOverview({ ClusterID: clusterID }),
      ]);

      const nodeData = nodeRes.data.data;
      const nodeUsages = overviewRes.data.data?.nodes?.usage || [];

      // 创建使用率映射
      const usageMap = new Map<string, { cpuUsage: number; memUsage: number }>();
      nodeUsages.forEach((u: { node: string; cpuUsage: number; memUsage: number }) => {
        usageMap.set(u.node, { cpuUsage: u.cpuUsage, memUsage: u.memUsage });
      });

      // 将节点信息转换为 Agent 状态
      const agentList: AgentInfo[] = nodeData.rows.map((node: {
        name: string;
        ready: boolean;
        internalIP: string;
        osImage: string;
        cpuCores: number;
        memoryGiB: number;
        schedulable: boolean;
      }) => {
        const usage = usageMap.get(node.name) || { cpuUsage: 0, memUsage: 0 };
        return {
          nodeId: node.name,
          nodeName: node.name,
          status: node.ready ? "online" : "offline" as const,
          lastSeen: node.ready ? "刚刚" : "未知",
          version: "v1.0.0",
          cpuUsage: usage.cpuUsage,
          memUsage: usage.memUsage,
          cpuCores: node.cpuCores,
          memoryGiB: node.memoryGiB,
          internalIP: node.internalIP,
          osImage: node.osImage,
        };
      });

      setAgents(agentList);
      setError("");
      setLastRefresh(new Date());
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAgents();
    // 每 30 秒自动刷新
    const interval = setInterval(fetchAgents, 30000);
    return () => clearInterval(interval);
  }, [fetchAgents]);

  const onlineCount = agents.filter((a) => a.status === "online").length;
  const offlineCount = agents.filter((a) => a.status === "offline").length;
  const degradedCount = agents.filter((a) => a.status === "degraded").length;

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title="Agents"
          description="监控各节点 Agent 连接状态"
          actions={
            <button
              onClick={fetchAgents}
              disabled={loading}
              className="flex items-center gap-2 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
              刷新
            </button>
          }
        />

        {/* 统计卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-primary/10">
                <Server className="w-5 h-5 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold text-default">{agents.length}</p>
                <p className="text-sm text-muted">Total Agents</p>
              </div>
            </div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-green-100 dark:bg-green-900/30">
                <Wifi className="w-5 h-5 text-green-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-green-500">{onlineCount}</p>
                <p className="text-sm text-muted">Online</p>
              </div>
            </div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-yellow-100 dark:bg-yellow-900/30">
                <AlertCircle className="w-5 h-5 text-yellow-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-yellow-500">{degradedCount}</p>
                <p className="text-sm text-muted">Degraded</p>
              </div>
            </div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-red-100 dark:bg-red-900/30">
                <WifiOff className="w-5 h-5 text-red-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-red-500">{offlineCount}</p>
                <p className="text-sm text-muted">Offline</p>
              </div>
            </div>
          </div>
        </div>

        {/* Agent 列表 */}
        {loading && agents.length === 0 ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : error ? (
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-8 text-center">
            <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
            <p className="text-red-500 mb-4">{error}</p>
            <button
              onClick={fetchAgents}
              className="px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg"
            >
              重试
            </button>
          </div>
        ) : agents.length === 0 ? (
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-8 text-center">
            <Server className="w-12 h-12 text-muted mx-auto mb-4" />
            <p className="text-muted">暂无 Agent 连接</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {agents.map((agent) => (
              <AgentCard key={agent.nodeId} agent={agent} />
            ))}
          </div>
        )}

        {/* 提示信息 */}
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-4">
          <div className="flex items-start gap-3">
            <Info className="w-5 h-5 text-blue-500 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-blue-800 dark:text-blue-300">Agent 状态说明</p>
              <p className="text-sm text-blue-700 dark:text-blue-400 mt-1">
                Agent 部署在每个 Kubernetes 节点上，负责收集 Metrics、日志和事件数据。
                当前显示的是基于节点数据的 Agent 状态。完整的 Agent 管理功能将在后续版本中提供。
              </p>
            </div>
          </div>
        </div>

        {/* 最后刷新时间 */}
        <div className="flex items-center justify-end gap-2 text-sm text-muted">
          <Clock className="w-4 h-4" />
          <span>Last updated: {lastRefresh.toLocaleTimeString()}</span>
        </div>
      </div>
    </Layout>
  );
}
