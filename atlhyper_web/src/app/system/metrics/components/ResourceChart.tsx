"use client";

import { useEffect, useRef, memo, useState } from "react";
import * as echarts from "echarts";
import { TrendingUp, Clock } from "lucide-react";
import type { MetricsDataPoint } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";

interface ResourceChartProps {
  data: MetricsDataPoint[];
  title?: string;
}

type TimeRange = "1h" | "6h" | "24h";

export const ResourceChart = memo(function ResourceChart({
  data,
  title,
}: ResourceChartProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const displayTitle = title || nm.chart.title;
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>("1h");

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

  // 初始化图表
  useEffect(() => {
    if (!chartRef.current) return;

    chartInstance.current = echarts.init(chartRef.current);
    const colors = getThemeColors();

    const baseOption: echarts.EChartsOption = {
      animation: true,
      animationDuration: 300,
      grid: { left: 50, right: 20, top: 40, bottom: 40 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        backgroundColor: colors.tooltipBg,
        borderColor: colors.tooltipBorder,
        textStyle: { color: colors.tooltipText, fontSize: 12 },
        formatter: (params: unknown) => {
          const items = params as { seriesName: string; value: [number, number]; color: string }[];
          if (!items || !items.length) return "";
          const time = new Date(items[0].value[0]).toLocaleTimeString();
          const lines = items
            .filter((item) => item.value && item.value[1] != null)
            .map(
              (item) => `<span style="color:${item.color}">●</span> ${item.seriesName}: ${item.value[1].toFixed(1)}%`
            );
          return `${time}<br/>${lines.join("<br/>")}`;
        },
      },
      legend: {
        top: 5,
        textStyle: { color: colors.textColor, fontSize: 11 },
      },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: colors.lineColor } },
        axisLabel: { color: colors.textColor, fontSize: 10 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        name: "%",
        min: 0,
        max: 100,
        axisLabel: { color: colors.textColor, fontSize: 10 },
        splitLine: { lineStyle: { color: colors.splitLineColor } },
      },
      series: [
        {
          name: "CPU",
          type: "line",
          smooth: true,
          showSymbol: false,
          data: [],
          color: "#F97316",
          lineStyle: { width: 2 },
          areaStyle: { opacity: 0.1 },
        },
        {
          name: "Memory",
          type: "line",
          smooth: true,
          showSymbol: false,
          data: [],
          color: "#10B981",
          lineStyle: { width: 2 },
          areaStyle: { opacity: 0.1 },
        },
        {
          name: "Disk",
          type: "line",
          smooth: true,
          showSymbol: false,
          data: [],
          color: "#8B5CF6",
          lineStyle: { width: 2 },
          areaStyle: { opacity: 0.1 },
        },
        {
          name: "Temp",
          type: "line",
          smooth: true,
          showSymbol: false,
          data: [],
          color: "#EF4444",
          lineStyle: { width: 2 },
        },
      ],
    };

    chartInstance.current.setOption(baseOption);

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
          yAxis: {
            axisLabel: { color: newColors.textColor },
            splitLine: { lineStyle: { color: newColors.splitLineColor } },
          },
        });
      }
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

    return () => {
      window.removeEventListener("resize", handleResize);
      observer.disconnect();
      chartInstance.current?.dispose();
      chartInstance.current = null;
    };
  }, []);

  // 数据更新
  useEffect(() => {
    if (!chartInstance.current || !data.length) return;

    // 根据时间范围过滤数据
    const now = Date.now();
    const rangeMs = {
      "1h": 60 * 60 * 1000,
      "6h": 6 * 60 * 60 * 1000,
      "24h": 24 * 60 * 60 * 1000,
    }[timeRange];
    const filteredData = data.filter((d) => d.timestamp > now - rangeMs);

    // 计算实际数据的时间范围
    let xAxisMin: number | undefined;
    let xAxisMax: number | undefined;
    if (filteredData.length > 0) {
      const timestamps = filteredData.map((d) => d.timestamp);
      const dataMin = Math.min(...timestamps);
      // X 轴从第一个数据点开始，到当前时间结束
      xAxisMin = dataMin;
      xAxisMax = now;
    }

    chartInstance.current.setOption({
      xAxis: {
        min: xAxisMin,
        max: xAxisMax,
      },
      series: [
        { data: filteredData.map((d) => [d.timestamp, d.cpuUsage]) },
        { data: filteredData.map((d) => [d.timestamp, d.memUsage]) },
        { data: filteredData.map((d) => [d.timestamp, d.diskUsage]) },
        { data: filteredData.map((d) => [d.timestamp, d.temperature]) },
      ],
    });
  }, [data, timeRange]);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-teal-500/10 rounded-lg">
            <TrendingUp className="w-5 h-5 text-teal-500" />
          </div>
          <h3 className="text-base font-semibold text-default">{displayTitle}</h3>
        </div>

        {/* 时间范围选择 */}
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-muted" />
          <div className="flex gap-1">
            {(["1h", "6h", "24h"] as TimeRange[]).map((range) => (
              <button
                key={range}
                onClick={() => setTimeRange(range)}
                className={`px-2 py-1 text-xs rounded transition-colors ${
                  timeRange === range
                    ? "bg-teal-500 text-white"
                    : "bg-[var(--background)] text-muted hover:text-default"
                }`}
              >
                {range}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* 图表 */}
      <div className="relative">
        <div ref={chartRef} style={{ width: "100%", height: "300px" }} />
        {data.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center bg-card text-muted">
            {nm.chart.noData}
          </div>
        )}
      </div>

      {/* 图例说明 */}
      <div className="mt-3 pt-3 border-t border-[var(--border-color)]">
        <div className="flex flex-wrap gap-4 text-xs">
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-0.5 bg-orange-500 rounded" />
            <span className="text-muted">{nm.cpu.title}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-0.5 bg-green-500 rounded" />
            <span className="text-muted">{nm.memory.title}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-0.5 bg-purple-500 rounded" />
            <span className="text-muted">{nm.disk.title}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-0.5 bg-red-500 rounded" />
            <span className="text-muted">{nm.temperature.title}</span>
          </div>
        </div>
      </div>
    </div>
  );
});
