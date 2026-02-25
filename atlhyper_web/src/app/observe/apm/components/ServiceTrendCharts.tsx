"use client";

import { useRef, useEffect, forwardRef, useCallback } from "react";
import * as echarts from "echarts";
import type { APMTimePoint } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";

function getThemeColors() {
  const isDark = document.documentElement.classList.contains("dark");
  return {
    textColor: isDark ? "#9ca3af" : "#6b7280",
    lineColor: isDark ? "#374151" : "#e5e7eb",
    splitLineColor: isDark ? "#1f2937" : "#f3f4f6",
    tooltipBg: isDark ? "#1f2937" : "#fff",
    tooltipBorder: isDark ? "#374151" : "#e5e7eb",
    tooltipText: isDark ? "#e5e7eb" : "#111827",
  };
}

interface ServiceTrendChartsProps {
  t: ApmTranslations;
  points: APMTimePoint[];
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

/** 单个趋势图卡片，内部自管理 ECharts 生命周期 */
const TrendChart = forwardRef<HTMLDivElement, {
  title: string;
  points: APMTimePoint[];
  emptyText: string;
  buildOption: (points: APMTimePoint[], colors: ReturnType<typeof getThemeColors>) => echarts.EChartsOption;
}>(({ title, points, emptyText, buildOption }, _ref) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);

  // 初始化 + 数据更新
  useEffect(() => {
    if (points.length === 0) {
      // 无数据时销毁图表
      if (chartRef.current) {
        chartRef.current.dispose();
        chartRef.current = null;
      }
      return;
    }
    if (!containerRef.current) return;

    // 懒初始化
    if (!chartRef.current) {
      chartRef.current = echarts.init(containerRef.current);
    }

    const c = getThemeColors();
    chartRef.current.setOption(buildOption(points, c), true);
  }, [points, buildOption]);

  // resize + 主题切换
  useEffect(() => {
    const handleResize = () => chartRef.current?.resize();
    window.addEventListener("resize", handleResize);

    const observer = new MutationObserver(() => {
      if (!chartRef.current) return;
      const c = getThemeColors();
      chartRef.current.setOption({
        tooltip: { backgroundColor: c.tooltipBg, borderColor: c.tooltipBorder, textStyle: { color: c.tooltipText } },
        xAxis: { axisLabel: { color: c.textColor }, axisLine: { lineStyle: { color: c.lineColor } } },
        yAxis: { axisLabel: { color: c.textColor }, splitLine: { lineStyle: { color: c.splitLineColor } } },
      });
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

    return () => {
      window.removeEventListener("resize", handleResize);
      observer.disconnect();
      chartRef.current?.dispose();
      chartRef.current = null;
    };
  }, []);

  return (
    <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
      <div className="text-xs text-muted font-medium mb-2">{title}</div>
      {points.length === 0 ? (
        <div className="w-full h-48 flex items-center justify-center text-sm text-muted">{emptyText}</div>
      ) : (
        <div ref={containerRef} className="w-full h-48" />
      )}
    </div>
  );
});
TrendChart.displayName = "TrendChart";

export function ServiceTrendCharts({ t, points }: ServiceTrendChartsProps) {
  const buildLatency = useCallback((pts: APMTimePoint[], c: ReturnType<typeof getThemeColors>): echarts.EChartsOption => {
    const times = pts.map((p) => formatTime(p.timestamp));
    const baseAxis = makeBaseAxis(times, c);
    return {
      ...baseAxis,
      tooltip: {
        ...baseAxis.tooltip,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: (params: any) => {
          const lines = params.map((p: { marker: string; seriesName: string; value: number }) =>
            `${p.marker} ${p.seriesName}: ${formatDurationMs(p.value)}`
          );
          return `${params[0].axisValue}<br/>${lines.join("<br/>")}`;
        },
      },
      yAxis: { ...baseAxis.yAxis, axisLabel: { ...baseAxis.yAxis.axisLabel, formatter: (v: number) => formatDurationMs(v) } },
      series: [
        { name: "Avg", type: "line", data: pts.map((p) => p.avgMs), smooth: true, symbol: "none", lineStyle: { width: 2 }, itemStyle: { color: "#3b82f6" } },
        { name: "P99", type: "line", data: pts.map((p) => p.p99Ms), smooth: true, symbol: "none", lineStyle: { width: 2, type: "dashed" }, itemStyle: { color: "#f59e0b" } },
      ],
    };
  }, []);

  const buildThroughput = useCallback((pts: APMTimePoint[], c: ReturnType<typeof getThemeColors>): echarts.EChartsOption => {
    const times = pts.map((p) => formatTime(p.timestamp));
    const baseAxis = makeBaseAxis(times, c);
    return {
      ...baseAxis,
      tooltip: {
        ...baseAxis.tooltip,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: (params: any) => {
          const p = params[0];
          return `${p.axisValue}<br/>${p.marker} ${p.value.toFixed(2)} req/s`;
        },
      },
      yAxis: { ...baseAxis.yAxis, axisLabel: { ...baseAxis.yAxis.axisLabel, formatter: (v: number) => `${v.toFixed(1)}` } },
      series: [
        {
          name: "RPS", type: "line", data: pts.map((p) => p.rps), smooth: true, symbol: "none",
          lineStyle: { width: 2 }, itemStyle: { color: "#22c55e" },
          areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: "rgba(34,197,94,0.3)" },
            { offset: 1, color: "rgba(34,197,94,0.02)" },
          ]) },
        },
      ],
    };
  }, []);

  const buildError = useCallback((pts: APMTimePoint[], c: ReturnType<typeof getThemeColors>): echarts.EChartsOption => {
    const times = pts.map((p) => formatTime(p.timestamp));
    const baseAxis = makeBaseAxis(times, c);
    const errorRates = pts.map((p) => Number(((1 - p.successRate) * 100).toFixed(2)));
    return {
      ...baseAxis,
      tooltip: {
        ...baseAxis.tooltip,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: (params: any) => {
          const lines = params.map((p: { marker: string; seriesName: string; value: number }) => {
            if (p.seriesName === t.errorRate) return `${p.marker} ${p.seriesName}: ${p.value.toFixed(2)}%`;
            return `${p.marker} ${p.seriesName}: ${p.value}`;
          });
          return `${params[0].axisValue}<br/>${lines.join("<br/>")}`;
        },
      },
      yAxis: [
        { ...baseAxis.yAxis, position: "left" },
        { ...baseAxis.yAxis, position: "right", axisLabel: { ...baseAxis.yAxis.axisLabel, formatter: (v: number) => `${v}%` }, splitLine: { show: false } },
      ],
      series: [
        { name: "Errors", type: "bar", data: pts.map((p) => p.errorCount), yAxisIndex: 0, itemStyle: { color: "#ef4444", borderRadius: [2, 2, 0, 0] }, barMaxWidth: 8 },
        { name: t.errorRate, type: "line", data: errorRates, yAxisIndex: 1, smooth: true, symbol: "none", lineStyle: { width: 2, type: "dashed" }, itemStyle: { color: "#f97316" } },
      ],
    };
  }, [t.errorRate]);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <TrendChart title={t.latencyTrend} points={points} emptyText={t.noData} buildOption={buildLatency} />
      <TrendChart title={t.throughputTrend} points={points} emptyText={t.noData} buildOption={buildThroughput} />
      <TrendChart title={t.errorCountTrend} points={points} emptyText={t.noData} buildOption={buildError} />
    </div>
  );
}

function makeBaseAxis(times: string[], c: ReturnType<typeof getThemeColors>) {
  return {
    xAxis: {
      type: "category" as const,
      data: times,
      boundaryGap: false,
      axisLabel: { color: c.textColor, fontSize: 10 },
      axisLine: { lineStyle: { color: c.lineColor } },
    },
    yAxis: {
      type: "value" as const,
      axisLabel: { color: c.textColor, fontSize: 10 },
      splitLine: { lineStyle: { color: c.splitLineColor } },
    },
    tooltip: {
      trigger: "axis" as const,
      backgroundColor: c.tooltipBg,
      borderColor: c.tooltipBorder,
      textStyle: { color: c.tooltipText, fontSize: 12 },
    },
    grid: { top: 8, right: 12, bottom: 24, left: 48 },
  };
}
