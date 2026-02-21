"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { TraceSummary } from "@/types/model/apm";

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
  traces: TraceSummary[];
}

export function LatencyChart({ title, traces }: LatencyChartProps) {
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
        xAxis: { axisLine: { lineStyle: { color: c.lineColor } }, axisLabel: { color: c.textColor } },
        yAxis: { axisLabel: { color: c.textColor }, splitLine: { lineStyle: { color: c.splitLineColor } } },
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
    if (!chartRef.current || traces.length === 0) return;
    const c = getThemeColors();

    const data = traces.map((t) => [new Date(t.timestamp).getTime(), t.durationMs]);

    chartRef.current.setOption({
      tooltip: {
        trigger: "item",
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBorder,
        textStyle: { color: c.tooltipText, fontSize: 12 },
        formatter: (params: { value: number[] }) => {
          const time = new Date(params.value[0]).toLocaleTimeString();
          return `${time}<br/>Latency: ${params.value[1].toFixed(1)}ms`;
        },
      },
      grid: { top: 12, right: 16, bottom: 32, left: 50 },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: c.lineColor } },
        axisLabel: { color: c.textColor, fontSize: 10, formatter: (val: number) => new Date(val).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        name: "ms",
        nameTextStyle: { color: c.textColor, fontSize: 10 },
        axisLabel: { color: c.textColor, fontSize: 10 },
        splitLine: { lineStyle: { color: c.splitLineColor } },
      },
      series: [{
        type: "scatter",
        data,
        symbolSize: 6,
        itemStyle: { color: "#6366f1" },
      }],
      animation: false,
    }, true);
  }, [traces]);

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">{title}</h4>
      <div ref={containerRef} className="w-full h-[200px]" />
    </div>
  );
}
