"use client";

import { useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getMetricsOverview } from "@/api/metrics";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn, LoadingSpinner } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import {
  Cpu,
  MemoryStick,
  Thermometer,
  HardDrive,
  Network,
  Eye,
  Activity,
} from "lucide-react";
import type { MetricsOverview, NodeMetricsRow } from "@/types/cluster";
import { NodeMetricsDetailModal } from "@/components/metrics";

// 获取使用率状态
function getUsageStatus(percent: number): "success" | "warning" | "error" | "default" {
  if (percent >= 90) return "error";
  if (percent >= 70) return "warning";
  if (percent >= 0) return "success";
  return "default";
}

// 格式化百分比
function formatPercent(value: number): string {
  return `${value.toFixed(1)}%`;
}

// 格式化网络速度
function formatKBps(value: number): string {
  if (value >= 1024) {
    return `${(value / 1024).toFixed(1)} MB/s`;
  }
  return `${value.toFixed(1)} KB/s`;
}

export default function MetricsPage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<MetricsOverview | null>(null);
  const [error, setError] = useState("");

  // 详情弹窗状态
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getMetricsOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 查看节点详情
  const handleViewDetail = (row: NodeMetricsRow) => {
    setSelectedNode(row.node);
    setDetailOpen(true);
  };

  const columns: TableColumn<NodeMetricsRow>[] = [
    {
      key: "node",
      header: "节点",
      render: (row) => (
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-primary" />
          <span className="font-medium text-default">{row.node}</span>
        </div>
      ),
    },
    {
      key: "cpu",
      header: "CPU",
      render: (row) => (
        <div className="flex items-center gap-2">
          <div className="w-16">
            <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${
                  row.cpuPercent >= 90
                    ? "bg-red-500"
                    : row.cpuPercent >= 70
                    ? "bg-yellow-500"
                    : "bg-green-500"
                }`}
                style={{ width: `${Math.min(100, row.cpuPercent)}%` }}
              />
            </div>
          </div>
          <span className="text-sm font-mono text-default w-14">{formatPercent(row.cpuPercent)}</span>
        </div>
      ),
    },
    {
      key: "memory",
      header: "内存",
      render: (row) => (
        <div className="flex items-center gap-2">
          <div className="w-16">
            <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${
                  row.memPercent >= 90
                    ? "bg-red-500"
                    : row.memPercent >= 70
                    ? "bg-yellow-500"
                    : "bg-green-500"
                }`}
                style={{ width: `${Math.min(100, row.memPercent)}%` }}
              />
            </div>
          </div>
          <span className="text-sm font-mono text-default w-14">{formatPercent(row.memPercent)}</span>
        </div>
      ),
    },
    {
      key: "temp",
      header: "温度",
      render: (row) => (
        <div className="flex items-center gap-1">
          <Thermometer className={`w-4 h-4 ${row.cpuTempC >= 80 ? "text-red-500" : row.cpuTempC >= 60 ? "text-yellow-500" : "text-green-500"}`} />
          <span className="text-sm font-mono">{row.cpuTempC > 0 ? `${row.cpuTempC.toFixed(0)}°C` : "-"}</span>
        </div>
      ),
    },
    {
      key: "disk",
      header: "磁盘",
      render: (row) => (
        <StatusBadge
          status={formatPercent(row.diskUsedPercent)}
          type={getUsageStatus(row.diskUsedPercent)}
        />
      ),
    },
    {
      key: "network",
      header: "网络 (TX/RX)",
      render: (row) => (
        <div className="text-sm">
          <span className="text-green-600 dark:text-green-400">↑ {formatKBps(row.eth0TxKBps)}</span>
          <span className="text-muted mx-1">/</span>
          <span className="text-blue-600 dark:text-blue-400">↓ {formatKBps(row.eth0RxKBps)}</span>
        </div>
      ),
    },
    {
      key: "topProcess",
      header: "Top 进程",
      render: (row) => (
        <span className="text-sm font-mono text-muted truncate max-w-[120px] block" title={row.topCPUProcess}>
          {row.topCPUProcess || "-"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      render: (row) => (
        <button
          onClick={() => handleViewDetail(row)}
          className="p-2 hover-bg rounded-lg"
          title="查看详情"
        >
          <Eye className="w-4 h-4 text-muted hover:text-primary" />
        </button>
      ),
    },
  ];

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.metrics}
          description="主机资源指标监控"
          autoRefreshSeconds={intervalSeconds}
        />

        {/* 统计卡片 */}
        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard
              label="平均 CPU"
              value={`${data.cards.avgCPUPercent.toFixed(1)}%`}
              icon={Cpu}
              iconColor={data.cards.avgCPUPercent >= 70 ? "text-yellow-500" : "text-green-500"}
            />
            <StatsCard
              label="平均内存"
              value={`${data.cards.avgMemPercent.toFixed(1)}%`}
              icon={MemoryStick}
              iconColor={data.cards.avgMemPercent >= 70 ? "text-yellow-500" : "text-green-500"}
            />
            <StatsCard
              label="峰值温度"
              value={data.cards.peakTempC > 0 ? `${data.cards.peakTempC.toFixed(0)}°C` : "-"}
              icon={Thermometer}
              iconColor={data.cards.peakTempC >= 80 ? "text-red-500" : data.cards.peakTempC >= 60 ? "text-yellow-500" : "text-green-500"}
              subtitle={data.cards.peakTempNode || undefined}
            />
            <StatsCard
              label="峰值磁盘"
              value={`${data.cards.peakDiskPercent.toFixed(1)}%`}
              icon={HardDrive}
              iconColor={data.cards.peakDiskPercent >= 80 ? "text-red-500" : data.cards.peakDiskPercent >= 60 ? "text-yellow-500" : "text-green-500"}
              subtitle={data.cards.peakDiskNode || undefined}
            />
          </div>
        )}

        {/* 节点列表 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          {loading ? (
            <div className="py-12">
              <LoadingSpinner />
            </div>
          ) : error ? (
            <div className="text-center py-12 text-red-500">{error}</div>
          ) : (
            <DataTable
              columns={columns}
              data={data?.rows || []}
              loading={false}
              error=""
              keyExtractor={(row, index) => `${index}-${row.node}`}
              pageSize={10}
            />
          )}
        </div>
      </div>

      {/* 节点详情弹窗 */}
      {selectedNode && (
        <NodeMetricsDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          nodeName={selectedNode}
        />
      )}
    </Layout>
  );
}
