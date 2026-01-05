"use client";

import { useEffect, useRef, memo } from "react";
import * as echarts from "echarts";

interface AlertTrendPoint {
  ts: number;
  critical: number;
  warning: number;
  info: number;
}

interface AlertTrendsChartProps {
  series: AlertTrendPoint[];
  height?: string;
}

// 使用 memo 避免不必要的重渲染
export const AlertTrendsChart = memo(function AlertTrendsChart({
  series,
  height = "280px",
}: AlertTrendsChartProps) {
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
      grid: { left: 50, right: 16, top: 40, bottom: 30 },
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
          const total = items.reduce((s, it) => s + (Number(it.value[1]) || 0), 0);
          const lines = items.map((it) => `${it.marker}${it.seriesName}: ${it.value[1]}`);
          return `${time}  (total ${total})<br/>${lines.join("<br/>")}`;
        },
      },
      legend: {
        top: 6,
        data: ["Critical", "Warning", "Info"],
        textStyle: { color: colors.textColor },
      },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: colors.lineColor } },
        axisLabel: { color: colors.textColor, fontSize: 11 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        min: 0,
        axisLabel: { color: colors.textColor, fontSize: 11 },
        axisLine: { show: false },
        splitLine: { lineStyle: { color: colors.splitLineColor } },
      },
      series: [
        {
          name: "Critical",
          type: "line",
          stack: "total",
          areaStyle: { opacity: 0.6 },
          showSymbol: false,
          smooth: true,
          lineStyle: { width: 2 },
          emphasis: { focus: "series" },
          data: [],
          itemStyle: { color: "#EF4444" },
        },
        {
          name: "Warning",
          type: "line",
          stack: "total",
          areaStyle: { opacity: 0.6 },
          showSymbol: false,
          smooth: true,
          lineStyle: { width: 2 },
          emphasis: { focus: "series" },
          data: [],
          itemStyle: { color: "#F59E0B" },
        },
        {
          name: "Info",
          type: "line",
          stack: "total",
          areaStyle: { opacity: 0.6 },
          showSymbol: false,
          smooth: true,
          lineStyle: { width: 2 },
          emphasis: { focus: "series" },
          data: [],
          itemStyle: { color: "#3B82F6" },
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
      isInitializedRef.current = false;
    };
  }, []);

  // 数据更新时平滑更新图表
  useEffect(() => {
    if (!chartInstance.current) return;

    const data = Array.isArray(series) ? series : [];

    // 使用 setOption 平滑更新数据
    chartInstance.current.setOption({
      series: [
        { data: data.map((p) => [p.ts, p.critical]) },
        { data: data.map((p) => [p.ts, p.warning]) },
        { data: data.map((p) => [p.ts, p.info]) },
      ],
    });
  }, [series]);

  const hasData = series.length > 0;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
      <h3 className="text-base font-semibold text-default mb-2">Alert Trends</h3>
      {!hasData ? (
        <div className="flex items-center justify-center text-muted" style={{ height }}>
          No alert trend data available
        </div>
      ) : (
        <div ref={chartRef} style={{ width: "100%", height }} />
      )}
    </div>
  );
});
