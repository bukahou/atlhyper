"use client";

import { useEffect, memo } from "react";
import type * as echarts from "echarts";
import { useEChart } from "@/hooks/useEChart";
import { useI18n } from "@/i18n/context";
import type { Point } from "@/types/node-metrics";
import { PALETTE, INTERVALS_BY_WINDOW, windowForHours } from "./chartConstants";
import { aggregatePoints, pickAdaptiveInterval } from "./chartUtils";
import { getThemeColors } from "@/lib/chart-theme";
import { joinHtml, safeHtml } from "@/lib/safe-html";
import { historyChartState } from "./historyChartState";

// ==================== Types ====================

export interface MetricChartProps {
  title: string;
  unit: string;
  metricKey: string;
  nodeNames: string[];
  /** Per-node history: { nodeName: { metricKey: Point[] } } */
  historyMap: Record<string, Record<string, Point[]>>;
  /** 历史请求失败的节点 */
  failedNodes: ReadonlySet<string>;
  hours: number;
}

/** 主题切换后重新上色（只改与主题有关的颜色） */
function applyThemeColors(chart: echarts.ECharts) {
  const c = getThemeColors();
  chart.setOption({
    tooltip: {
      backgroundColor: c.tooltipBg,
      borderColor: c.tooltipBorder,
      textStyle: { color: c.tooltipText },
    },
    xAxis: {
      axisLine: { lineStyle: { color: c.lineColor } },
      axisLabel: { color: c.textColor },
    },
    yAxis: {
      nameTextStyle: { color: c.textColor },
      axisLabel: { color: c.textColor },
      splitLine: { lineStyle: { color: c.splitLineColor } },
    },
  });
}

// ==================== Component ====================

export const MetricChart = memo(function MetricChart({
  title,
  unit,
  metricKey,
  nodeNames,
  historyMap,
  failedNodes,
  hours,
}: MetricChartProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const { containerRef, chart } = useEChart({ onThemeChange: applyThemeColors });

  // 基础配置（颜色取当前主题）
  useEffect(() => {
    if (!chart) return;
    const colors = getThemeColors();
    chart.setOption({
      animation: true,
      animationDuration: 300,
      grid: { left: 45, right: 12, top: 10, bottom: 30 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "line" },
        backgroundColor: colors.tooltipBg,
        borderColor: colors.tooltipBorder,
        textStyle: { color: colors.tooltipText, fontSize: 11 },
      },
      legend: { show: false },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: colors.lineColor } },
        axisLabel: { color: colors.textColor, fontSize: 9 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        nameTextStyle: { color: colors.textColor, fontSize: 9 },
        min: 0,
        axisLabel: { color: colors.textColor, fontSize: 9 },
        splitLine: { lineStyle: { color: colors.splitLineColor } },
      },
      series: [],
    });
  }, [chart]);

  // Data update
  useEffect(() => {
    if (!chart) return;

    const now = Date.now();
    const rangeMs = hours * 3600000;

    const pointsInRange = nodeNames.map((name) =>
      (historyMap[name]?.[metricKey] || []).filter((p) => new Date(p.timestamp).getTime() > now - rangeMs),
    );

    // 桶宽全自动：按窗口内数据的实际跨度挑，不提供手动选择 ——
    // 主流做法是范围定桶宽，让用户挑采样粒度是把实现细节漏给了用户
    let earliestMs = Infinity;
    let latestMs = -Infinity;
    for (const point of pointsInRange.flat()) {
      const timestampMs = new Date(point.timestamp).getTime();
      if (timestampMs < earliestMs) earliestMs = timestampMs;
      if (timestampMs > latestMs) latestMs = timestampMs;
    }
    const dataSpanMs = latestMs > earliestMs ? latestMs - earliestMs : 0;
    const intervalSec = pickAdaptiveInterval(dataSpanMs, INTERVALS_BY_WINDOW[windowForHours(hours)]);

    const series = nodeNames.map((name, i) => {
      const aggregated = aggregatePoints(pointsInRange[i], intervalSec);
      return {
        name,
        type: "line" as const,
        smooth: true,
        showSymbol: false,
        color: PALETTE[i % PALETTE.length],
        lineStyle: { width: 1.5 },
        data: aggregated,
      };
    });

    const allTimestamps = series.flatMap((s) => s.data.map((d) => d[0]));
    const xAxisMin = allTimestamps.length > 0 ? Math.min(...allTimestamps) : undefined;
    const xAxisMax = allTimestamps.length > 0 ? now : undefined;

    // Dynamic Y-axis
    const allValues = series.flatMap((s) => s.data.map((d) => d[1]));
    const dataMax = allValues.length > 0 ? Math.max(...allValues) : 0;
    const yMax = Math.max(10, Math.ceil(dataMax * 1.2));

    chart.setOption({
      tooltip: {
        formatter: (params: unknown) => {
          const items = params as { seriesName: string; value: [number, number]; color: string }[];
          if (!items?.length) return "";
          const time = new Date(items[0].value[0]).toLocaleTimeString();
          const lines = items
            .filter((item) => item.value?.[1] != null)
            .map(
              (item) =>
                safeHtml`<span style="color:${item.color}">●</span> ${item.seriesName}: ${item.value[1].toFixed(1)}${unit}`
            );
          return safeHtml`${time}<br/>${joinHtml(lines, safeHtml`<br/>`)}`.toString();
        },
      },
      xAxis: { min: xAxisMin, max: xAxisMax, minInterval: intervalSec * 1000 },
      yAxis: { name: unit, max: yMax },
      series,
    });
  }, [chart, hours, historyMap, nodeNames, metricKey, unit]);

  const chartState = historyChartState(nodeNames, metricKey, historyMap, failedNodes);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3">
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs font-semibold text-default">{title}</span>

      </div>
      <div className="relative">
        <div ref={containerRef} style={{ width: "100%", height: "200px" }} />
        {chartState === "failed" && (
          <div className="absolute inset-0 flex items-center justify-center text-xs text-red-500">
            {nm.overview.loadFailed}
          </div>
        )}
        {chartState === "empty" && (
          <div className="absolute inset-0 flex items-center justify-center text-xs text-muted">
            {nm.overview.noData}
          </div>
        )}
      </div>
      {/* Per-chart legend */}
      <div className="flex flex-wrap gap-x-3 gap-y-0.5 mt-1">
        {nodeNames.map((name, i) => (
          <div key={name} className="flex items-center gap-1">
            <div className="w-2.5 h-0.5 rounded" style={{ backgroundColor: PALETTE[i % PALETTE.length] }} />
            <span className="text-[10px] text-muted">{name}</span>
          </div>
        ))}
      </div>
    </div>
  );
});
