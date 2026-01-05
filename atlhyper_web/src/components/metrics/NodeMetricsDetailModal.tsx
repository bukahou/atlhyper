"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getNodeMetricsDetail } from "@/api/metrics";
import { getCurrentClusterId } from "@/config/cluster";
import type { NodeMetricsDetail, TopCPUProcess } from "@/types/cluster";
import * as echarts from "echarts";
import {
  Cpu,
  MemoryStick,
  Thermometer,
  HardDrive,
  Network,
  Activity,
} from "lucide-react";

interface NodeMetricsDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
}

type TabType = "overview" | "processes";

export function NodeMetricsDetailModal({
  isOpen,
  onClose,
  nodeName,
}: NodeMetricsDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<NodeMetricsDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!nodeName) return;
    setLoading(true);
    setError("");
    try {
      const res = await getNodeMetricsDetail({
        clusterID: getCurrentClusterId(),
        nodeID: nodeName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [nodeName]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: "概览", icon: <Activity className="w-4 h-4" /> },
    { key: "processes", label: "进程", icon: <Cpu className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`节点: ${nodeName}`} size="xl">
      {loading ? (
        <div className="py-12">
          <LoadingSpinner />
        </div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
        <div className="flex flex-col h-[75vh]">
          {/* Tabs */}
          <div className="flex border-b border-[var(--border-color)] px-4 shrink-0">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.key
                    ? "border-primary text-primary"
                    : "border-transparent text-muted hover:text-default"
                }`}
              >
                {tab.icon}
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-auto p-6">
            {activeTab === "overview" && <OverviewTab detail={detail} />}
            {activeTab === "processes" && <ProcessesTab processes={detail.processes || []} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: NodeMetricsDetail }) {
  const { latest, series } = detail;

  // 格式化指标项
  const metrics = [
    { label: "CPU", value: `${latest.cpuPercent.toFixed(1)}%`, icon: Cpu, color: "text-orange-500" },
    { label: "内存", value: `${latest.memPercent.toFixed(1)}%`, icon: MemoryStick, color: "text-green-500" },
    { label: "温度", value: latest.cpuTempC > 0 ? `${latest.cpuTempC.toFixed(0)}°C` : "-", icon: Thermometer, color: "text-red-500" },
    { label: "磁盘", value: `${latest.diskUsedPercent.toFixed(1)}%`, icon: HardDrive, color: "text-purple-500" },
    { label: "网络 TX", value: `${latest.eth0TxKBps.toFixed(1)} KB/s`, icon: Network, color: "text-blue-500" },
    { label: "网络 RX", value: `${latest.eth0RxKBps.toFixed(1)} KB/s`, icon: Network, color: "text-cyan-500" },
  ];

  return (
    <div className="space-y-6">
      {/* 当前指标 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">当前指标</h3>
        <div className="grid grid-cols-3 md:grid-cols-6 gap-3">
          {metrics.map((m, i) => {
            const Icon = m.icon;
            return (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
                <Icon className={`w-5 h-5 ${m.color} mx-auto mb-2`} />
                <div className="text-lg font-bold text-default">{m.value}</div>
                <div className="text-xs text-muted">{m.label}</div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Top 进程 */}
      {latest.topCPUProcess && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-2">Top CPU 进程</h3>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <span className="font-mono text-sm text-default">{latest.topCPUProcess}</span>
          </div>
        </div>
      )}

      {/* 趋势图表 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">15 分钟趋势</h3>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <MetricsChart
            title="CPU / 内存"
            series={[
              { name: "CPU", data: series.cpuPct, color: "#F97316" },
              { name: "Memory", data: series.memPct, color: "#10B981" },
            ]}
            times={series.at}
            unit="%"
            max={100}
          />
          <MetricsChart
            title="温度 / 磁盘"
            series={[
              { name: "温度", data: series.tempC, color: "#EF4444" },
              { name: "磁盘", data: series.diskPct, color: "#8B5CF6" },
            ]}
            times={series.at}
            unit=""
          />
          <MetricsChart
            title="网络流量"
            series={[
              { name: "TX", data: series.eth0TxKBps, color: "#3B82F6" },
              { name: "RX", data: series.eth0RxKBps, color: "#06B6D4" },
            ]}
            times={series.at}
            unit="KB/s"
          />
        </div>
      </div>
    </div>
  );
}

// 图表组件
function MetricsChart({
  title,
  series,
  times,
  unit,
  max,
}: {
  title: string;
  series: { name: string; data: number[]; color: string }[];
  times: string[];
  unit: string;
  max?: number;
}) {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    if (!chartRef.current) return;

    if (!chartInstance.current) {
      chartInstance.current = echarts.init(chartRef.current);
    }

    const isDark = document.documentElement.classList.contains("dark");
    const textColor = isDark ? "#9ca3af" : "#6b7280";
    const lineColor = isDark ? "#374151" : "#e5e7eb";

    const option: echarts.EChartsOption = {
      animation: true,
      animationDuration: 300,
      grid: { left: 50, right: 16, top: 30, bottom: 30 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        backgroundColor: isDark ? "#1f2937" : "#fff",
        borderColor: isDark ? "#374151" : "#e5e7eb",
        textStyle: { color: isDark ? "#e5e7eb" : "#111827" },
      },
      legend: {
        top: 0,
        textStyle: { color: textColor },
      },
      xAxis: {
        type: "category",
        data: times.map((t) => {
          const d = new Date(t);
          return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
        }),
        axisLine: { lineStyle: { color: lineColor } },
        axisLabel: { color: textColor, fontSize: 10 },
      },
      yAxis: {
        type: "value",
        name: unit,
        min: 0,
        max: max,
        axisLabel: { color: textColor, fontSize: 10 },
        splitLine: { lineStyle: { color: isDark ? "#1f2937" : "#f3f4f6" } },
      },
      series: series.map((s) => ({
        name: s.name,
        type: "line",
        smooth: true,
        showSymbol: false,
        data: s.data,
        lineStyle: { width: 2, color: s.color },
        areaStyle: { opacity: 0.1, color: s.color },
        itemStyle: { color: s.color },
      })),
    };

    chartInstance.current.setOption(option);

    const handleResize = () => chartInstance.current?.resize();
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, [series, times, unit, max]);

  useEffect(() => {
    return () => {
      chartInstance.current?.dispose();
      chartInstance.current = null;
    };
  }, []);

  const hasData = series.some((s) => s.data.length > 0);

  return (
    <div className="bg-[var(--background)] rounded-lg p-4">
      <h4 className="text-sm font-medium text-default mb-2">{title}</h4>
      {!hasData ? (
        <div className="h-48 flex items-center justify-center text-muted">暂无数据</div>
      ) : (
        <div ref={chartRef} style={{ width: "100%", height: "180px" }} />
      )}
    </div>
  );
}

// 进程 Tab
function ProcessesTab({ processes }: { processes: TopCPUProcess[] }) {
  if (!processes || processes.length === 0) {
    return <div className="text-center py-8 text-muted">暂无进程信息</div>;
  }

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-12 gap-2 px-3 py-2 text-xs font-medium text-muted border-b border-[var(--border-color)]">
        <div className="col-span-2">PID</div>
        <div className="col-span-2">用户</div>
        <div className="col-span-5">命令</div>
        <div className="col-span-3 text-right">CPU%</div>
      </div>
      {processes.map((proc, i) => (
        <div
          key={i}
          className="grid grid-cols-12 gap-2 px-3 py-2 text-sm bg-[var(--background)] rounded-lg hover:bg-card transition-colors"
        >
          <div className="col-span-2 font-mono text-muted">{proc.pid}</div>
          <div className="col-span-2 text-default">{proc.user}</div>
          <div className="col-span-5 font-mono text-default truncate" title={proc.command}>
            {proc.command}
          </div>
          <div className="col-span-3 text-right">
            <StatusBadge
              status={proc.cpuUsage || `${proc.cpuPercent.toFixed(1)}%`}
              type={proc.cpuPercent >= 50 ? "warning" : "info"}
            />
          </div>
        </div>
      ))}
    </div>
  );
}
