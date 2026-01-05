"use client";

import { useEffect, useRef, memo } from "react";
import * as echarts from "echarts";
import { Cpu, HardDrive, Thermometer, ArrowDownToLine, ArrowUpFromLine } from "lucide-react";

interface PeakStats {
  peakCpu: number;
  peakCpuNode: string;
  peakMem: number;
  peakMemNode: string;
  peakTemp: number;
  peakTempNode: string;
  netRxKBps: number;
  netTxKBps: number;
  hasData: boolean;
}

interface ResourceTrendsChartProps {
  cpu: [number, number][];
  mem: [number, number][];
  temp: [number, number][];
  peakStats?: PeakStats;
  height?: string;
}

// 使用 memo 避免不必要的重渲染
export const ResourceTrendsChart = memo(function ResourceTrendsChart({
  cpu,
  mem,
  temp,
  peakStats,
  height = "200px",
}: ResourceTrendsChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);
  const isInitializedRef = useRef(false);

  // 获取主题颜色
  const getThemeColors = () => {
    const isDark = document.documentElement.classList.contains("dark");
    return {
      textColor: isDark ? "#9ca3af" : "#6b7280",
      lineColor: isDark ? "#374151" : "#e5e7eb",
      splitLineColor: isDark ? "#1f2937" : "#f3f4f6",
      tooltipBg: isDark ? "#1f2937" : "#fff",
      tooltipBorder: isDark ? "#374151" : "#e5e7eb",
      tooltipText: isDark ? "#e5e7eb" : "#111827",
    };
  };

  // 初始化图表（仅执行一次）
  useEffect(() => {
    if (!chartRef.current || isInitializedRef.current) return;

    chartInstance.current = echarts.init(chartRef.current);
    isInitializedRef.current = true;

    const colors = getThemeColors();

    const baseOption: echarts.EChartsOption = {
      animation: true,
      animationDuration: 300,
      animationEasing: "cubicOut",
      grid: { left: 50, right: 50, top: 40, bottom: 30 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        backgroundColor: colors.tooltipBg,
        borderColor: colors.tooltipBorder,
        textStyle: { color: colors.tooltipText },
        formatter: (params: unknown) => {
          const items = params as { value: [number, number]; seriesName: string; marker: string }[];
          if (!items?.length) return "";
          const dt = new Date(items[0].value[0]);
          const time = `${String(dt.getHours()).padStart(2, "0")}:${String(dt.getMinutes()).padStart(2, "0")}`;
          const lines = items.map((it) => {
            const unit = it.seriesName === "Temp" ? "°C" : "%";
            return `${it.marker}${it.seriesName}: ${it.value[1]?.toFixed(1) ?? 0}${unit}`;
          });
          return `${time}<br/>${lines.join("<br/>")}`;
        },
      },
      legend: {
        top: 6,
        data: ["CPU", "Memory", "Temp"],
        textStyle: { color: colors.textColor },
      },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: colors.lineColor } },
        axisLabel: { color: colors.textColor, fontSize: 11 },
        splitLine: { show: false },
      },
      yAxis: [
        {
          type: "value",
          name: "%",
          min: 0,
          max: 100,
          position: "left",
          axisLabel: { color: colors.textColor, formatter: "{value}", fontSize: 11 },
          axisLine: { show: false },
          splitLine: { lineStyle: { color: colors.splitLineColor } },
        },
        {
          type: "value",
          name: "°C",
          min: 0,
          max: 100,
          position: "right",
          axisLabel: { color: "#EF4444", formatter: "{value}", fontSize: 11 },
          axisLine: { show: true, lineStyle: { color: "#EF4444" } },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: "CPU",
          type: "line",
          smooth: true,
          showSymbol: false,
          yAxisIndex: 0,
          data: [],
          lineStyle: { width: 2, color: "#F97316" },
          areaStyle: { opacity: 0.1, color: "#F97316" },
          itemStyle: { color: "#F97316" },
        },
        {
          name: "Memory",
          type: "line",
          smooth: true,
          showSymbol: false,
          yAxisIndex: 0,
          data: [],
          lineStyle: { width: 2, color: "#10B981" },
          areaStyle: { opacity: 0.1, color: "#10B981" },
          itemStyle: { color: "#10B981" },
        },
        {
          name: "Temp",
          type: "line",
          smooth: true,
          showSymbol: false,
          yAxisIndex: 1,
          data: [],
          lineStyle: { width: 2, color: "#EF4444" },
          itemStyle: { color: "#EF4444" },
        },
      ],
    };

    chartInstance.current.setOption(baseOption);

    // 监听窗口大小变化
    const handleResize = () => chartInstance.current?.resize();
    window.addEventListener("resize", handleResize);

    // 监听主题变化
    const observer = new MutationObserver(() => {
      if (chartInstance.current) {
        const newColors = getThemeColors();
        chartInstance.current.setOption({
          tooltip: {
            backgroundColor: newColors.tooltipBg,
            borderColor: newColors.tooltipBorder,
            textStyle: { color: newColors.tooltipText },
          },
          legend: { textStyle: { color: newColors.textColor } },
          xAxis: {
            axisLine: { lineStyle: { color: newColors.lineColor } },
            axisLabel: { color: newColors.textColor },
          },
          yAxis: [
            {
              axisLabel: { color: newColors.textColor },
              splitLine: { lineStyle: { color: newColors.splitLineColor } },
            },
            {
              axisLabel: { color: "#EF4444" },
            },
          ],
        });
      }
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

    return () => {
      window.removeEventListener("resize", handleResize);
      observer.disconnect();
      chartInstance.current?.dispose();
      chartInstance.current = null;
      isInitializedRef.current = false;
    };
  }, []);

  // 数据更新时平滑更新图表
  useEffect(() => {
    if (!chartInstance.current) return;

    // 使用 setOption 平滑更新数据
    chartInstance.current.setOption({
      series: [
        { data: cpu },
        { data: mem },
        { data: temp },
      ],
    });
  }, [cpu, mem, temp]);

  const hasData = cpu.length > 0 || mem.length > 0;
  const hasMetricsPlugin = peakStats?.hasData ?? false;

  // 底部状态卡片组件
  const StatMiniCard = ({
    icon: Icon,
    label,
    value,
    node,
    color
  }: {
    icon: typeof Cpu;
    label: string;
    value: string;
    node?: string;
    color: string;
  }) => (
    <div className="flex items-center gap-2 bg-[var(--background)] rounded-lg px-3 py-2 min-w-0">
      <Icon className="w-4 h-4 flex-shrink-0" style={{ color }} />
      <div className="min-w-0 flex-1">
        <div className="text-xs text-muted truncate">{label}</div>
        <div className="text-sm font-semibold text-default">{value}</div>
        {node && <div className="text-xs text-muted truncate" title={node}>{node}</div>}
      </div>
    </div>
  );

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <h3 className="text-base font-semibold text-default mb-2">Resource Usage Trends</h3>

      {/* 趋势图 */}
      {!hasData ? (
        <div className="flex items-center justify-center text-muted" style={{ height }}>
          No trend data available
        </div>
      ) : (
        <div ref={chartRef} style={{ width: "100%", height }} />
      )}

      {/* 底部状态卡片 */}
      {!hasMetricsPlugin ? (
        <div className="mt-3 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
          <p className="text-sm text-yellow-700 dark:text-yellow-400">
            Metrics plugin not installed. Install the metrics agent for detailed resource monitoring.
          </p>
        </div>
      ) : peakStats && (
        <div className="mt-3 grid grid-cols-2 md:grid-cols-4 gap-2">
          <StatMiniCard
            icon={Cpu}
            label="Peak CPU"
            value={`${peakStats.peakCpu.toFixed(1)}%`}
            node={peakStats.peakCpuNode}
            color="#F97316"
          />
          <StatMiniCard
            icon={HardDrive}
            label="Peak Memory"
            value={`${peakStats.peakMem.toFixed(1)}%`}
            node={peakStats.peakMemNode}
            color="#10B981"
          />
          <StatMiniCard
            icon={Thermometer}
            label="Max Temp"
            value={`${peakStats.peakTemp.toFixed(1)}°C`}
            node={peakStats.peakTempNode}
            color="#EF4444"
          />
          <div className="flex items-center gap-2 bg-[var(--background)] rounded-lg px-3 py-2 min-w-0">
            <div className="flex flex-col gap-0.5">
              <div className="flex items-center gap-1">
                <ArrowDownToLine className="w-3 h-3 text-blue-500" />
                <span className="text-xs text-muted">Rx</span>
                <span className="text-xs font-semibold text-default">{peakStats.netRxKBps.toFixed(1)} KB/s</span>
              </div>
              <div className="flex items-center gap-1">
                <ArrowUpFromLine className="w-3 h-3 text-purple-500" />
                <span className="text-xs text-muted">Tx</span>
                <span className="text-xs font-semibold text-default">{peakStats.netTxKBps.toFixed(1)} KB/s</span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
});
