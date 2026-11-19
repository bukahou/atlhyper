"use client";

import { useEffect, forwardRef, useCallback } from "react";
import * as echarts from "echarts";
import { useEChart } from "@/hooks/useEChart";
import { getThemeColors, type ChartThemeColors } from "@/lib/chart-theme";
import type { APMTimePoint } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import { formatDurationMs } from "@/lib/format";
import { joinHtml, safeHtml, trustedHtml } from "@/lib/safe-html";

/** 主题切换后重新上色（只改与主题有关的颜色） */
function applyThemeColors(chart: echarts.ECharts) {
  const c = getThemeColors();
  chart.setOption({
    tooltip: { backgroundColor: c.tooltipBg, borderColor: c.tooltipBorder, textStyle: { color: c.tooltipText } },
    xAxis: { axisLabel: { color: c.textColor }, axisLine: { lineStyle: { color: c.lineColor } } },
    yAxis: { axisLabel: { color: c.textColor }, splitLine: { lineStyle: { color: c.splitLineColor } } },
  });
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
  buildOption: (points: APMTimePoint[], colors: ChartThemeColors) => echarts.EChartsOption;
}>(({ title, points, emptyText, buildOption }, _ref) => {
  // 容器只在有数据时渲染：没数据时图表随容器一起销毁（见 useEChart）
  const { containerRef, chart } = useEChart({ onThemeChange: applyThemeColors });

  useEffect(() => {
    if (!chart || points.length === 0) return;
    chart.setOption(buildOption(points, getThemeColors()), true);
  }, [chart, points, buildOption]);

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
  const buildLatency = useCallback((pts: APMTimePoint[], c: ChartThemeColors): echarts.EChartsOption => {
    const times = pts.map((p) => formatTime(p.timestamp));
    const baseAxis = makeBaseAxis(times, c);
    return {
      ...baseAxis,
      tooltip: {
        ...baseAxis.tooltip,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: (params: any) => {
          const lines = params.map((p: { marker: string; seriesName: string; value: number }) =>
            safeHtml`${trustedHtml(p.marker)} ${p.seriesName}: ${formatDurationMs(p.value)}`
          );
          return safeHtml`${params[0].axisValue}<br/>${joinHtml(lines, safeHtml`<br/>`)}`.toString();
        },
      },
      yAxis: { ...baseAxis.yAxis, axisLabel: { ...baseAxis.yAxis.axisLabel, formatter: (v: number) => formatDurationMs(v) } },
      series: [
        { name: "Avg", type: "line", data: pts.map((p) => p.avgMs), smooth: true, symbol: "none", lineStyle: { width: 2 }, itemStyle: { color: "#3b82f6" } },
        { name: "P99", type: "line", data: pts.map((p) => p.p99Ms), smooth: true, symbol: "none", lineStyle: { width: 2, type: "dashed" }, itemStyle: { color: "#f59e0b" } },
      ],
    };
  }, []);

  const buildThroughput = useCallback((pts: APMTimePoint[], c: ChartThemeColors): echarts.EChartsOption => {
    const times = pts.map((p) => formatTime(p.timestamp));
    const baseAxis = makeBaseAxis(times, c);
    return {
      ...baseAxis,
      tooltip: {
        ...baseAxis.tooltip,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: (params: any) => {
          const p = params[0];
          return safeHtml`${p.axisValue}<br/>${trustedHtml(p.marker)} ${p.value.toFixed(2)} req/s`.toString();
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

  const buildError = useCallback((pts: APMTimePoint[], c: ChartThemeColors): echarts.EChartsOption => {
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
            if (p.seriesName === t.errorRate) return safeHtml`${trustedHtml(p.marker)} ${p.seriesName}: ${p.value.toFixed(2)}%`;
            return safeHtml`${trustedHtml(p.marker)} ${p.seriesName}: ${p.value}`;
          });
          return safeHtml`${params[0].axisValue}<br/>${joinHtml(lines, safeHtml`<br/>`)}`.toString();
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

function makeBaseAxis(times: string[], c: ChartThemeColors) {
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
