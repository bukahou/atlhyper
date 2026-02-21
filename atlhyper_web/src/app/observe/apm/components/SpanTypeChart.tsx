"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { SpanTypeBreakdown } from "@/types/model/apm";

function getThemeColors() {
  const isDark = document.documentElement.classList.contains("dark");
  return {
    textColor: isDark ? "#9ca3af" : "#6b7280",
    tooltipBg: isDark ? "#1f2937" : "#fff",
    tooltipBorder: isDark ? "#374151" : "#e5e7eb",
    tooltipText: isDark ? "#e5e7eb" : "#111827",
  };
}

const TYPE_COLORS: Record<string, string> = {
  HTTP: "#6366f1",
  DB: "#f59e0b",
  Other: "#8b5cf6",
};

interface SpanTypeChartProps {
  title: string;
  breakdown: SpanTypeBreakdown[];
}

export function SpanTypeChart({ title, breakdown }: SpanTypeChartProps) {
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
        legend: { textStyle: { color: c.textColor } },
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
    if (!chartRef.current || breakdown.length === 0) return;
    const c = getThemeColors();

    chartRef.current.setOption({
      tooltip: {
        trigger: "axis",
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBorder,
        textStyle: { color: c.tooltipText, fontSize: 12 },
        formatter: (params: { seriesName: string; value: number }[]) =>
          params.map((p) => `${p.seriesName}: ${p.value.toFixed(1)}%`).join("<br/>"),
      },
      legend: {
        data: breakdown.map((b) => b.type),
        bottom: 0,
        textStyle: { color: c.textColor, fontSize: 11 },
        itemWidth: 12,
        itemHeight: 12,
      },
      grid: { top: 8, right: 16, bottom: 32, left: 16 },
      xAxis: {
        type: "value",
        max: 100,
        show: false,
      },
      yAxis: {
        type: "category",
        data: [""],
        show: false,
      },
      series: breakdown.map((b) => ({
        name: b.type,
        type: "bar" as const,
        stack: "total",
        data: [b.percentage],
        itemStyle: {
          color: TYPE_COLORS[b.type] ?? "#94a3b8",
          borderRadius: 0,
        },
        barWidth: 24,
      })),
      animation: false,
    }, true);
  }, [breakdown]);

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>
      <div ref={containerRef} className="w-full h-[80px]" />
    </div>
  );
}
