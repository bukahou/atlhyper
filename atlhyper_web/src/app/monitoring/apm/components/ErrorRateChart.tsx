"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import type { TraceSummary } from "@/api/apm";

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

interface ErrorRateChartProps {
  title: string;
  traces: TraceSummary[];
}

export function ErrorRateChart({ title, traces }: ErrorRateChartProps) {
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

    // Compute error rate per trace as a running rate
    const sorted = [...traces].sort((a, b) => a.startTime - b.startTime);
    let errorSum = 0;
    const data = sorted.map((t, i) => {
      if (t.hasError) errorSum++;
      const rate = ((errorSum / (i + 1)) * 100);
      return [t.startTime / 1000, parseFloat(rate.toFixed(1))];
    });

    chartRef.current.setOption({
      tooltip: {
        trigger: "axis",
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBorder,
        textStyle: { color: c.tooltipText, fontSize: 12 },
        formatter: (params: { value: number[] }[]) => {
          const p = params[0];
          const time = new Date(p.value[0]).toLocaleTimeString();
          return `${time}<br/>Error rate: ${p.value[1]}%`;
        },
      },
      grid: { top: 12, right: 16, bottom: 32, left: 40 },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: c.lineColor } },
        axisLabel: { color: c.textColor, fontSize: 10, formatter: (val: number) => new Date(val).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        max: 100,
        axisLabel: { color: c.textColor, fontSize: 10, formatter: (v: number) => `${v}%` },
        splitLine: { lineStyle: { color: c.splitLineColor } },
      },
      series: [{
        type: "line",
        data,
        smooth: true,
        lineStyle: { color: "#f97316", width: 2 },
        areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: "rgba(249,115,22,0.2)" },
          { offset: 1, color: "rgba(249,115,22,0)" },
        ])},
        itemStyle: { color: "#f97316" },
        symbol: "circle",
        symbolSize: 4,
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
