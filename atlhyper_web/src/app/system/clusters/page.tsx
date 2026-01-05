"use client";

import { useState, useEffect, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import {
  Server,
  Box,
  Cpu,
  HardDrive,
  CheckCircle,
  AlertCircle,
  Clock,
  RefreshCw,
  Plus,
  Settings,
  ExternalLink,
  Info
} from "lucide-react";
import { getClusterOverview } from "@/api/overview";

// 集群信息类型
interface ClusterInfo {
  id: string;
  name: string;
  status: "healthy" | "degraded" | "unhealthy" | "unknown";
  nodeTotal: number;
  nodeReady: number;
  cpuUsage: number;
  memUsage: number;
  podCount: number;
  isActive: boolean;
  lastSync: string;
}

// 状态配置
const statusConfig = {
  healthy: {
    label: "健康",
    color: "text-green-500",
    bgColor: "bg-green-100 dark:bg-green-900/30",
    borderColor: "border-green-500",
    icon: CheckCircle,
  },
  degraded: {
    label: "降级",
    color: "text-yellow-500",
    bgColor: "bg-yellow-100 dark:bg-yellow-900/30",
    borderColor: "border-yellow-500",
    icon: AlertCircle,
  },
  unhealthy: {
    label: "异常",
    color: "text-red-500",
    bgColor: "bg-red-100 dark:bg-red-900/30",
    borderColor: "border-red-500",
    icon: AlertCircle,
  },
  unknown: {
    label: "未知",
    color: "text-gray-500",
    bgColor: "bg-gray-100 dark:bg-gray-700",
    borderColor: "border-gray-500",
    icon: AlertCircle,
  },
};

// 集群卡片组件
function ClusterCard({
  cluster,
  onSelect,
}: {
  cluster: ClusterInfo;
  onSelect: () => void;
}) {
  const config = statusConfig[cluster.status];
  const StatusIcon = config.icon;

  return (
    <div
      className={`bg-card rounded-xl border-2 p-6 transition-all hover:shadow-lg cursor-pointer ${
        cluster.isActive
          ? `${config.borderColor} shadow-md`
          : "border-[var(--border-color)] hover:border-primary/50"
      }`}
      onClick={onSelect}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`p-3 rounded-lg ${config.bgColor}`}>
            <Server className={`w-6 h-6 ${config.color}`} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-lg font-semibold text-default">{cluster.name}</h3>
              {cluster.isActive && (
                <span className="px-2 py-0.5 text-xs font-medium bg-primary text-white rounded-full">
                  当前
                </span>
              )}
            </div>
            <p className="text-xs text-muted font-mono">{cluster.id}</p>
          </div>
        </div>
        <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${config.bgColor} ${config.color}`}>
          <StatusIcon className="w-3 h-3" />
          {config.label}
        </span>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-2 gap-4 mb-4">
        <div className="flex items-center gap-2">
          <Server className="w-4 h-4 text-muted" />
          <div>
            <p className="text-xs text-muted">Nodes</p>
            <p className="text-sm font-medium text-default">
              {cluster.nodeReady}/{cluster.nodeTotal}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Box className="w-4 h-4 text-muted" />
          <div>
            <p className="text-xs text-muted">Pods</p>
            <p className="text-sm font-medium text-default">{cluster.podCount}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Cpu className="w-4 h-4 text-muted" />
          <div>
            <p className="text-xs text-muted">CPU</p>
            <p className="text-sm font-medium text-default">{cluster.cpuUsage.toFixed(1)}%</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <HardDrive className="w-4 h-4 text-muted" />
          <div>
            <p className="text-xs text-muted">Memory</p>
            <p className="text-sm font-medium text-default">{cluster.memUsage.toFixed(1)}%</p>
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="pt-4 border-t border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-1 text-xs text-muted">
          <Clock className="w-3 h-3" />
          <span>Last sync: {cluster.lastSync}</span>
        </div>
        <button
          className="p-1.5 hover:bg-[var(--background)] rounded-lg transition-colors"
          onClick={(e) => {
            e.stopPropagation();
          }}
        >
          <Settings className="w-4 h-4 text-muted" />
        </button>
      </div>
    </div>
  );
}

export default function ClustersPage() {
  const { t } = useI18n();
  const [clusters, setClusters] = useState<ClusterInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const fetchClusters = useCallback(async () => {
    try {
      setLoading(true);

      // 获取当前集群信息
      const res = await getClusterOverview({ ClusterID: "ZGMF-X10A" });
      const data = res.data.data;

      // 将当前集群转换为 ClusterInfo
      const clusterList: ClusterInfo[] = [
        {
          id: data.clusterId || "ZGMF-X10A",
          name: "Production Cluster",
          status: data.cards.clusterHealth.status.toLowerCase() as ClusterInfo["status"],
          nodeTotal: data.cards.nodeReady.total,
          nodeReady: data.cards.nodeReady.ready,
          cpuUsage: data.cards.cpuUsage.percent,
          memUsage: data.cards.memUsage.percent,
          podCount: data.nodes?.usage?.length || 0,
          isActive: true,
          lastSync: "刚刚",
        },
      ];

      setClusters(clusterList);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchClusters();
  }, [fetchClusters]);

  const handleSelectCluster = (clusterId: string) => {
    // 在多集群场景下，这里会切换当前活动集群
    console.log("Selected cluster:", clusterId);
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title="集群管理"
          description="查看和管理 Kubernetes 集群"
          actions={
            <div className="flex items-center gap-3">
              <button
                onClick={fetchClusters}
                disabled={loading}
                className="flex items-center gap-2 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
                刷新
              </button>
              <button
                disabled
                className="flex items-center gap-2 px-4 py-2 bg-primary/50 text-white rounded-lg cursor-not-allowed"
                title="多集群支持开发中"
              >
                <Plus className="w-4 h-4" />
                添加集群
              </button>
            </div>
          }
        />

        {/* 统计概览 */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-primary/10">
                <Server className="w-5 h-5 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold text-default">{clusters.length}</p>
                <p className="text-sm text-muted">总集群数</p>
              </div>
            </div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-green-100 dark:bg-green-900/30">
                <CheckCircle className="w-5 h-5 text-green-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-green-500">
                  {clusters.filter((c) => c.status === "healthy").length}
                </p>
                <p className="text-sm text-muted">健康集群</p>
              </div>
            </div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900/30">
                <Server className="w-5 h-5 text-blue-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-blue-500">
                  {clusters.reduce((sum, c) => sum + c.nodeTotal, 0)}
                </p>
                <p className="text-sm text-muted">总节点数</p>
              </div>
            </div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-purple-100 dark:bg-purple-900/30">
                <Box className="w-5 h-5 text-purple-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-purple-500">
                  {clusters.reduce((sum, c) => sum + c.podCount, 0)}
                </p>
                <p className="text-sm text-muted">总 Pod 数</p>
              </div>
            </div>
          </div>
        </div>

        {/* 集群列表 */}
        {loading ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : error ? (
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-8 text-center">
            <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
            <p className="text-red-500 mb-4">{error}</p>
            <button
              onClick={fetchClusters}
              className="px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg"
            >
              重试
            </button>
          </div>
        ) : clusters.length === 0 ? (
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-8 text-center">
            <Server className="w-12 h-12 text-muted mx-auto mb-4" />
            <p className="text-muted mb-4">暂无集群</p>
            <button className="px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg">
              添加集群
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {clusters.map((cluster) => (
              <ClusterCard
                key={cluster.id}
                cluster={cluster}
                onSelect={() => handleSelectCluster(cluster.id)}
              />
            ))}
          </div>
        )}

        {/* 功能说明 */}
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-4">
          <div className="flex items-start gap-3">
            <Info className="w-5 h-5 text-blue-500 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-blue-800 dark:text-blue-300">多集群管理</p>
              <p className="text-sm text-blue-700 dark:text-blue-400 mt-1">
                当前版本支持单集群监控。多集群联邦管理功能正在开发中，
                将支持跨集群资源查看、统一告警和集群切换等功能。
              </p>
            </div>
          </div>
        </div>

        {/* 快速链接 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
          <h3 className="font-semibold text-default mb-4">快速操作</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <a
              href="/cluster/node"
              className="flex items-center gap-3 p-4 rounded-lg border border-[var(--border-color)] hover:bg-[var(--background)] transition-colors"
            >
              <Server className="w-5 h-5 text-primary" />
              <div className="flex-1">
                <p className="text-sm font-medium text-default">节点管理</p>
                <p className="text-xs text-muted">查看集群节点状态</p>
              </div>
              <ExternalLink className="w-4 h-4 text-muted" />
            </a>
            <a
              href="/cluster/pod"
              className="flex items-center gap-3 p-4 rounded-lg border border-[var(--border-color)] hover:bg-[var(--background)] transition-colors"
            >
              <Box className="w-5 h-5 text-primary" />
              <div className="flex-1">
                <p className="text-sm font-medium text-default">Pod 管理</p>
                <p className="text-xs text-muted">查看和管理 Pod</p>
              </div>
              <ExternalLink className="w-4 h-4 text-muted" />
            </a>
            <a
              href="/system/agents"
              className="flex items-center gap-3 p-4 rounded-lg border border-[var(--border-color)] hover:bg-[var(--background)] transition-colors"
            >
              <Settings className="w-5 h-5 text-primary" />
              <div className="flex-1">
                <p className="text-sm font-medium text-default">Agent 状态</p>
                <p className="text-xs text-muted">监控 Agent 连接</p>
              </div>
              <ExternalLink className="w-4 h-4 text-muted" />
            </a>
          </div>
        </div>
      </div>
    </Layout>
  );
}
