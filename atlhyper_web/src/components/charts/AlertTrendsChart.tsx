"use client";

import { useEffect, useRef, memo, useMemo } from "react";
import * as echarts from "echarts";

interface AlertTrendPoint {
  ts: number;
  kinds: Record<string, number>; // 按资源类型统计: {"Pod": 5, "Node": 2}
}

interface AlertTrendsChartProps {
  series: AlertTrendPoint[];
  height?: string;
}

// 资源类型颜色配置
const KIND_COLORS: Record<string, string> = {
  Pod: "#F59E0B",           // Amber
  Node: "#EF4444",          // Red
  Deployment: "#3B82F6",    // Blue
  StatefulSet: "#8B5CF6",   // Purple
  DaemonSet: "#10B981",     // Emerald
  ReplicaSet: "#6366F1",    // Indigo
  Job: "#EC4899",           // Pink
  CronJob: "#14B8A6",       // Teal
  Service: "#F97316",       // Orange
  Ingress: "#06B6D4",       // Cyan
  PersistentVolumeClaim: "#84CC16", // Lime
};

// 获取 Kind 的颜色
const getKindColor = (kind: string): string => {
  return KIND_COLORS[kind] || "#9CA3AF"; // 默认灰色
};

// 使用 memo 避免不必要的重渲染
export const AlertTrendsChart = memo(function AlertTrendsChart({
  series,
  height = "280px",
}: AlertTrendsChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);
  const isInitializedRef = useRef(false);

  // 提取所有出现过的 Kind
  const allKinds = useMemo(() => {
    const kindsSet = new Set<string>();
    for (const point of series) {
      if (point.kinds) {
        Object.keys(point.kinds).forEach((k) => kindsSet.add(k));
      }
    }
    // 按预定义顺序排序，未定义的放最后
    const predefinedOrder = Object.keys(KIND_COLORS);
    return Array.from(kindsSet).sort((a, b) => {
      const aIdx = predefinedOrder.indexOf(a);
      const bIdx = predefinedOrder.indexOf(b);
      if (aIdx === -1 && bIdx === -1) return a.localeCompare(b);
      if (aIdx === -1) return 1;
      if (bIdx === -1) return -1;
      return aIdx - bIdx;
    });
  }, [series]);

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
      grid: { left: 50, right: 16, top: 32, bottom: 30 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        backgroundColor: colors.tooltipBg,
        borderColor: colors.tooltipBorder,
        textStyle: { color: colors.tooltipText },
        formatter: (params: unknown) => {
          const items = params as { value: [number, number]; marker: string; seriesName: string }[];
          if (!items?.length) return "";
          const dt = new Date(items[0].value[0]);
          const time = `${String(dt.getHours()).padStart(2, "0")}:00`;
          let html = `<b>${time}</b>`;
          for (const item of items) {
            const count = item.value[1] || 0;
            if (count > 0) {
              html += `<br/>${item.marker} ${item.seriesName}: ${count}`;
            }
          }
          return html;
        },
      },
      legend: {
        show: true,
        top: 0,
        right: 0,
        itemWidth: 12,
        itemHeight: 12,
        textStyle: { color: colors.textColor, fontSize: 11 },
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
      series: [],
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

    // 为每种 Kind 生成一个 series
    const chartSeries = allKinds.map((kind) => {
      const color = getKindColor(kind);
      return {
        name: kind,
        type: "line" as const,
        stack: "total", // 堆叠显示
        areaStyle: { opacity: 0.4 },
        showSymbol: false,
        smooth: true,
        lineStyle: { width: 1.5, color },
        itemStyle: { color },
        emphasis: { focus: "series" as const },
        data: data.map((p) => [p.ts, p.kinds?.[kind] || 0]),
      };
    });

    chartInstance.current.setOption({
      series: chartSeries,
    }, { replaceMerge: ["series"] });
  }, [series, allKinds]);

  const hasData = series.length > 0;

  // 图表高度 = 290px(总高) - 32px(padding) - 28px(标题) = 230px
  const chartHeight = "230px";

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-4 h-[290px]">
      <h3 className="text-base font-semibold text-default mb-2">Alert Trends</h3>
      {!hasData ? (
        <div className="flex items-center justify-center text-muted" style={{ height: chartHeight }}>
          No alert trend data available
        </div>
      ) : (
        <div ref={chartRef} style={{ width: "100%", height: chartHeight }} />
      )}
    </div>
  );
});
