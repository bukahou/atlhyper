"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { OperationStats } from "@/types/model/apm";
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

interface LatencyChartProps {
  title: string;
  operations: OperationStats[];
}

export function LatencyChart({ title, operations }: LatencyChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const chart = echarts.init(containerRef.current);
    chartRef.current = chart;

    const handleResize = () => chart.resize();
    window.addEventListener("resize", handleResize);

    const observer = new MutationObserver(() => {
      if (!chartRef.current) return;
      const c = getThemeColors();
      chartRef.current.setOption({
        tooltip: { backgroundColor: c.tooltipBg, borderColor: c.tooltipBorder, textStyle: { color: c.tooltipText } },
        xAxis: { axisLabel: { color: c.textColor }, splitLine: { lineStyle: { color: c.splitLineColor } } },
        yAxis: { axisLine: { lineStyle: { color: c.lineColor } }, axisLabel: { color: c.textColor } },
      });
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

    return () => {
      window.removeEventListener("resize", handleResize);
      observer.disconnect();
      chart.dispose();
    };
  }, []);

  useEffect(() => {
    if (!chartRef.current || operations.length === 0) return;
    const c = getThemeColors();

    // Top 10 by avgDurationMs descending, then reverse for bottom-to-top display
    const top = [...operations]
      .sort((a, b) => b.avgDurationMs - a.avgDurationMs)
      .slice(0, 10)
      .reverse();

    const names = top.map((op) => {
      const n = op.operationName;
      return n.length > 30 ? n.slice(0, 27) + "..." : n;
    });

    chartRef.current.setOption({
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "shadow" },
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBorder,
        textStyle: { color: c.tooltipText, fontSize: 12 },
        formatter: (params: { seriesName: string; value: number; dataIndex: number }[]) => {
          const opName = top[params[0]?.dataIndex]?.operationName ?? "";
          let html = `<div style="font-weight:600;margin-bottom:4px">${opName}</div>`;
          for (const p of params) {
            html += `${p.seriesName}: ${formatDurationMs(p.value)}<br/>`;
          }
          return html;
        },
      },
      legend: {
        data: ["Avg", "P50", "P99"],
        right: 0,
        top: 0,
        textStyle: { color: c.textColor, fontSize: 11 },
        itemWidth: 12,
        itemHeight: 8,
      },
      grid: { top: 28, right: 16, bottom: 8, left: 120, containLabel: false },
      xAxis: {
        type: "value",
        axisLabel: { color: c.textColor, fontSize: 10, formatter: (v: number) => formatDurationMs(v) },
        splitLine: { lineStyle: { color: c.splitLineColor } },
      },
      yAxis: {
        type: "category",
        data: names,
        axisLine: { lineStyle: { color: c.lineColor } },
        axisLabel: { color: c.textColor, fontSize: 10 },
        axisTick: { show: false },
      },
      series: [
        {
          name: "Avg",
          type: "bar",
          data: top.map((op) => op.avgDurationMs),
          itemStyle: { color: "#6366f1", borderRadius: [0, 2, 2, 0] },
          barMaxWidth: 8,
        },
        {
          name: "P50",
          type: "bar",
          data: top.map((op) => op.p50Ms),
          itemStyle: { color: "#22c55e", borderRadius: [0, 2, 2, 0] },
          barMaxWidth: 8,
        },
        {
          name: "P99",
          type: "bar",
          data: top.map((op) => op.p99Ms),
          itemStyle: { color: "#f97316", borderRadius: [0, 2, 2, 0] },
          barMaxWidth: 8,
        },
      ],
      animation: false,
    }, true);
  }, [operations]);

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>
      <div ref={containerRef} className="w-full h-[200px]" />
    </div>
  );
}
